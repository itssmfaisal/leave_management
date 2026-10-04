package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const dateLayout = "2006-01-02"

type API struct {
	store *Store
}

func (a *API) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/profile", a.getProfile)
	mux.HandleFunc("PUT /api/profile", a.putProfile)
	mux.HandleFunc("GET /api/years/{year}", a.getYear)
	mux.HandleFunc("PUT /api/years/{year}/opening", a.putOpening)
	mux.HandleFunc("GET /api/leaves", a.listLeaves)
	mux.HandleFunc("POST /api/leaves", a.createLeave)
	mux.HandleFunc("PUT /api/leaves/{id}", a.updateLeave)
	mux.HandleFunc("DELETE /api/leaves/{id}", a.deleteLeave)
}

/* ---------------- Profile ---------------- */

func (a *API) getProfile(w http.ResponseWriter, r *http.Request) {
	p, err := a.store.GetProfile()
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (a *API) putProfile(w http.ResponseWriter, r *http.Request) {
	var p Profile
	if !readJSON(w, r, &p) {
		return
	}
	p.Name, p.EmpNo = strings.TrimSpace(p.Name), strings.TrimSpace(p.EmpNo)
	if p.Name == "" || p.EmpNo == "" {
		badRequest(w, "Name and employee number are required.")
		return
	}
	if err := a.store.SaveProfile(p); err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

/* ---------------- Year balance ---------------- */

func (a *API) getYear(w http.ResponseWriter, r *http.Request) {
	year, ok := parseYear(w, r)
	if !ok {
		return
	}
	sum, err := a.store.YearSummary(year)
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sum)
}

func (a *API) putOpening(w http.ResponseWriter, r *http.Request) {
	year, ok := parseYear(w, r)
	if !ok {
		return
	}
	var body struct {
		Used float64 `json:"used"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	if body.Used < 0 || body.Used > Eligibility {
		badRequest(w, "Already used days must be between 0 and 21.")
		return
	}
	if err := a.store.SetOpeningUsed(year, body.Used); err != nil {
		serverError(w, err)
		return
	}
	a.getYear(w, r)
}

/* ---------------- Leaves ---------------- */

func (a *API) listLeaves(w http.ResponseWriter, r *http.Request) {
	leaves, err := a.store.ListLeaves()
	if err != nil {
		serverError(w, err)
		return
	}
	if leaves == nil {
		leaves = []Leave{}
	}
	writeJSON(w, http.StatusOK, leaves)
}

func (a *API) createLeave(w http.ResponseWriter, r *http.Request) {
	var l Leave
	if !readJSON(w, r, &l) {
		return
	}
	l.AppliedOn = time.Now().Format(dateLayout) // always "today", server time zone
	if msg := validateLeave(&l); msg != "" {
		badRequest(w, msg)
		return
	}
	saved, err := a.store.CreateLeave(l)
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, saved)
}

func (a *API) updateLeave(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	existing, err := a.store.GetLeave(id)
	if errors.Is(err, ErrNotFound) {
		notFound(w)
		return
	} else if err != nil {
		serverError(w, err)
		return
	}

	var l Leave
	if !readJSON(w, r, &l) {
		return
	}
	l.Name, l.EmpNo, l.AppliedOn = existing.Name, existing.EmpNo, existing.AppliedOn
	if msg := validateLeave(&l); msg != "" {
		badRequest(w, msg)
		return
	}
	saved, err := a.store.UpdateLeave(id, l)
	if errors.Is(err, ErrNotFound) {
		notFound(w)
		return
	} else if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

func (a *API) deleteLeave(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	switch err := a.store.DeleteLeave(id); {
	case errors.Is(err, ErrNotFound):
		notFound(w)
	case err != nil:
		serverError(w, err)
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

func validateLeave(l *Leave) string {
	l.Name = strings.TrimSpace(l.Name)
	l.EmpNo = strings.TrimSpace(l.EmpNo)
	l.Reason = strings.TrimSpace(l.Reason)

	if l.Name == "" || l.EmpNo == "" {
		return "Name and employee number are required."
	}
	if l.LeaveType != "holiday" && l.LeaveType != "sick" {
		return "Leave type must be holiday or sick."
	}
	if l.Reason == "" {
		return "Reason of leave is required."
	}

	dates := map[string]string{
		"Last working day":    l.LastWorkingDay,
		"Holiday starts from": l.StartDate,
		"Holiday end":         l.EndDate,
		"Re-joining date":     l.RejoinDate,
	}
	for label, v := range dates {
		if _, err := time.Parse(dateLayout, v); err != nil {
			return label + " is not a valid date."
		}
	}
	// ISO dates compare correctly as strings.
	switch {
	case l.EndDate < l.StartDate:
		return "Holiday end cannot be before the start date."
	case l.LastWorkingDay >= l.StartDate:
		return "Last working day must be before the holiday start."
	case l.RejoinDate <= l.EndDate:
		return "Re-joining date must be after the holiday end."
	case l.NumDays <= 0:
		return "Number of days must be greater than 0."
	}
	return ""
}

/* ---------------- Helpers ---------------- */

func parseYear(w http.ResponseWriter, r *http.Request) (int, bool) {
	year, err := strconv.Atoi(r.PathValue("year"))
	if err != nil || year < 2000 || year > 2100 {
		badRequest(w, "Invalid year.")
		return 0, false
	}
	return year, true
}

func parseID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		badRequest(w, "Invalid id.")
		return 0, false
	}
	return id, true
}

func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		badRequest(w, "Invalid JSON body.")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func badRequest(w http.ResponseWriter, msg string) {
	writeJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
}

func notFound(w http.ResponseWriter) {
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "Leave not found."})
}

func serverError(w http.ResponseWriter, err error) {
	log.Printf("error: %v", err)
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error."})
}
