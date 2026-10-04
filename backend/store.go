package main

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	_ "modernc.org/sqlite"
)

// Yearly leave eligibility in days.
const Eligibility = 21.0

var ErrNotFound = errors.New("not found")

type Profile struct {
	Name  string `json:"name"`
	EmpNo string `json:"empNo"`
}

type Leave struct {
	ID             int64    `json:"id"`
	AppliedOn      string   `json:"appliedOn"`
	Name           string   `json:"name"`
	EmpNo          string   `json:"empNo"`
	LeaveType      string   `json:"leaveType"` // "holiday" | "sick"
	IsSaturday     bool     `json:"isSaturday"`
	LastWorkingDay string   `json:"lastWorkingDay"`
	StartDate      string   `json:"startDate"`
	EndDate        string   `json:"endDate"`
	NumDays        float64  `json:"numDays"`
	RejoinDate     string   `json:"rejoinDate"`
	Reason         string   `json:"reason"`
	Balance        *Balance `json:"balance,omitempty"`
}

// Balance as printed on a leave application.
type Balance struct {
	Eligibility float64 `json:"eligibility"`
	Used        float64 `json:"used"`
	ThisTime    float64 `json:"thisTime"`
	Remaining   float64 `json:"remaining"`
}

// YearSummary is the current state of a calendar year.
type YearSummary struct {
	Year        int     `json:"year"`
	Eligibility float64 `json:"eligibility"`
	OpeningUsed float64 `json:"openingUsed"` // days used before this app, entered manually
	AppUsed     float64 `json:"appUsed"`     // days used by applications saved in this app
	Used        float64 `json:"used"`
	Remaining   float64 `json:"remaining"`
}

// Saturday leave is not deducted from eligibility.
func (l *Leave) Chargeable() float64 {
	if l.IsSaturday {
		return 0
	}
	return l.NumDays
}

func (l *Leave) Year() int {
	y, _ := strconv.Atoi(l.StartDate[:4])
	return y
}

const schema = `
CREATE TABLE IF NOT EXISTS profile (
	id     INTEGER PRIMARY KEY CHECK (id = 1),
	name   TEXT NOT NULL DEFAULT '',
	emp_no TEXT NOT NULL DEFAULT ''
);
INSERT OR IGNORE INTO profile (id) VALUES (1);

CREATE TABLE IF NOT EXISTS opening_balance (
	year INTEGER PRIMARY KEY,
	used REAL NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS leaves (
	id               INTEGER PRIMARY KEY AUTOINCREMENT,
	applied_on       TEXT NOT NULL,
	name             TEXT NOT NULL,
	emp_no           TEXT NOT NULL,
	leave_type       TEXT NOT NULL CHECK (leave_type IN ('holiday', 'sick')),
	is_saturday      INTEGER NOT NULL DEFAULT 0,
	last_working_day TEXT NOT NULL,
	start_date       TEXT NOT NULL,
	end_date         TEXT NOT NULL,
	num_days         REAL NOT NULL,
	rejoin_date      TEXT NOT NULL,
	reason           TEXT NOT NULL,
	created_at       TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`

type Store struct {
	db *sql.DB
}

func OpenStore(path string) (*Store, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1) // SQLite has a single writer; keep it simple.
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

/* ---------------- Profile ---------------- */

func (s *Store) GetProfile() (Profile, error) {
	var p Profile
	err := s.db.QueryRow(`SELECT name, emp_no FROM profile WHERE id = 1`).Scan(&p.Name, &p.EmpNo)
	return p, err
}

func (s *Store) SaveProfile(p Profile) error {
	_, err := s.db.Exec(`UPDATE profile SET name = ?, emp_no = ? WHERE id = 1`, p.Name, p.EmpNo)
	return err
}

/* ---------------- Opening balance ---------------- */

func (s *Store) openingBalances() (map[int]float64, error) {
	rows, err := s.db.Query(`SELECT year, used FROM opening_balance`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := map[int]float64{}
	for rows.Next() {
		var y int
		var used float64
		if err := rows.Scan(&y, &used); err != nil {
			return nil, err
		}
		m[y] = used
	}
	return m, rows.Err()
}

func (s *Store) SetOpeningUsed(year int, used float64) error {
	_, err := s.db.Exec(
		`INSERT INTO opening_balance (year, used) VALUES (?, ?)
		 ON CONFLICT(year) DO UPDATE SET used = excluded.used`, year, used)
	return err
}

/* ---------------- Leaves ---------------- */

// ListLeaves returns all leaves, newest first, each with the balance as it
// stood when that application was made (opening + earlier leaves that year).
func (s *Store) ListLeaves() ([]Leave, error) {
	opening, err := s.openingBalances()
	if err != nil {
		return nil, err
	}

	rows, err := s.db.Query(`
		SELECT id, applied_on, name, emp_no, leave_type, is_saturday, last_working_day,
		       start_date, end_date, num_days, rejoin_date, reason
		FROM leaves ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var leaves []Leave
	used := map[int]float64{}
	for rows.Next() {
		var l Leave
		if err := rows.Scan(&l.ID, &l.AppliedOn, &l.Name, &l.EmpNo, &l.LeaveType, &l.IsSaturday,
			&l.LastWorkingDay, &l.StartDate, &l.EndDate, &l.NumDays, &l.RejoinDate, &l.Reason); err != nil {
			return nil, err
		}
		y := l.Year()
		if _, ok := used[y]; !ok {
			used[y] = opening[y]
		}
		l.Balance = &Balance{
			Eligibility: Eligibility,
			Used:        used[y],
			ThisTime:    l.Chargeable(),
			Remaining:   Eligibility - used[y] - l.Chargeable(),
		}
		used[y] += l.Chargeable()
		leaves = append(leaves, l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Newest first.
	for i, j := 0, len(leaves)-1; i < j; i, j = i+1, j-1 {
		leaves[i], leaves[j] = leaves[j], leaves[i]
	}
	return leaves, nil
}

// CreateLeave saves the application and remembers the employee profile.
func (s *Store) CreateLeave(l Leave) (Leave, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return Leave{}, err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		INSERT INTO leaves (applied_on, name, emp_no, leave_type, is_saturday, last_working_day,
		                    start_date, end_date, num_days, rejoin_date, reason)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		l.AppliedOn, l.Name, l.EmpNo, l.LeaveType, l.IsSaturday, l.LastWorkingDay,
		l.StartDate, l.EndDate, l.NumDays, l.RejoinDate, l.Reason)
	if err != nil {
		return Leave{}, err
	}
	if _, err := tx.Exec(`UPDATE profile SET name = ?, emp_no = ? WHERE id = 1`, l.Name, l.EmpNo); err != nil {
		return Leave{}, err
	}
	if err := tx.Commit(); err != nil {
		return Leave{}, err
	}

	id, _ := res.LastInsertId()
	return s.GetLeave(id)
}

func (s *Store) GetLeave(id int64) (Leave, error) {
	leaves, err := s.ListLeaves()
	if err != nil {
		return Leave{}, err
	}
	for _, l := range leaves {
		if l.ID == id {
			return l, nil
		}
	}
	return Leave{}, ErrNotFound
}

func (s *Store) DeleteLeave(id int64) error {
	res, err := s.db.Exec(`DELETE FROM leaves WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) YearSummary(year int) (YearSummary, error) {
	sum := YearSummary{Year: year, Eligibility: Eligibility}
	err := s.db.QueryRow(`SELECT COALESCE((SELECT used FROM opening_balance WHERE year = ?), 0)`, year).
		Scan(&sum.OpeningUsed)
	if err != nil {
		return sum, err
	}
	err = s.db.QueryRow(`
		SELECT COALESCE(SUM(num_days), 0) FROM leaves
		WHERE is_saturday = 0 AND substr(start_date, 1, 4) = ?`, fmt.Sprintf("%04d", year)).
		Scan(&sum.AppUsed)
	if err != nil {
		return sum, err
	}
	sum.Used = sum.OpeningUsed + sum.AppUsed
	sum.Remaining = Eligibility - sum.Used
	return sum, nil
}
