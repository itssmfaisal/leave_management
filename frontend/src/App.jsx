import { useCallback, useEffect, useRef, useState } from "react";
import { api } from "./api.js";
import { addDays, daysInclusive, formatShort, todayISO, yearOf } from "./dates.js";
import Letter from "./Letter.jsx";
import OpeningBalance from "./OpeningBalance.jsx";

const ELIGIBILITY = 21;

const EMPTY_FORM = {
  leaveType: "holiday",
  isSaturday: false,
  lastWorkingDay: "",
  startDate: "",
  endDate: "",
  numDays: "",
  rejoinDate: "",
  reason: "",
};

export default function App() {
  const [profile, setProfile] = useState({ name: "", empNo: "" });
  const [form, setForm] = useState(EMPTY_FORM);
  const [leaves, setLeaves] = useState([]);
  const [yearSummary, setYearSummary] = useState(null);
  const [refreshKey, setRefreshKey] = useState(0);
  const [printTarget, setPrintTarget] = useState(null);
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);
  const [profileStatus, setProfileStatus] = useState("");

  // Fields that are auto-filled until the user edits them by hand.
  const touched = useRef(new Set());

  const formYear = yearOf(form.startDate);
  const refresh = useCallback(() => setRefreshKey((k) => k + 1), []);

  /* ---------------- Data loading ---------------- */

  useEffect(() => {
    api.getProfile().then(setProfile).catch((e) => setError(e.message));
  }, []);

  useEffect(() => {
    api.listLeaves().then(setLeaves).catch((e) => setError(e.message));
  }, [refreshKey]);

  useEffect(() => {
    api.getYear(formYear).then(setYearSummary).catch((e) => setError(e.message));
  }, [formYear, refreshKey]);

  /* ---------------- Printing ---------------- */

  useEffect(() => {
    if (!printTarget) return;
    const prevTitle = document.title;
    // Becomes the default PDF file name.
    document.title = `Leave_${printTarget.name.replace(/\s+/g, "_")}_${printTarget.startDate}`;
    window.print();
    document.title = prevTitle;
    setPrintTarget(null);
  }, [printTarget]);

  /* ---------------- Form handling ---------------- */

  function update(key, value) {
    if (["lastWorkingDay", "numDays", "rejoinDate"].includes(key)) touched.current.add(key);

    setForm((prev) => {
      const next = { ...prev, [key]: value };
      const { startDate: start, endDate: end } = next;
      if (key === "startDate" || key === "endDate") {
        if (start && !touched.current.has("lastWorkingDay")) next.lastWorkingDay = addDays(start, -1);
        if (start && end && end >= start && !touched.current.has("numDays")) next.numDays = String(daysInclusive(start, end));
        if (end && !touched.current.has("rejoinDate")) next.rejoinDate = addDays(end, 1);
      }
      return next;
    });
  }

  function updateProfile(key, value) {
    setProfile((prev) => ({ ...prev, [key]: value }));
    setProfileStatus("");
  }

  async function handleSaveProfile() {
    try {
      setProfile(await api.saveProfile(profile));
      setProfileStatus("Saved.");
    } catch (err) {
      setProfileStatus(err.message);
    }
  }

  function clearForm() {
    setForm(EMPTY_FORM);
    touched.current.clear();
    setError("");
  }

  async function handleSubmit(e) {
    e.preventDefault();
    setError("");
    setSaving(true);
    try {
      const saved = await api.createLeave({ ...profile, ...form, numDays: Number(form.numDays) });
      clearForm();
      refresh();
      setPrintTarget(saved);
    } catch (err) {
      setError(err.message);
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete(leave) {
    if (!confirm(`Delete leave from ${formatShort(leave.startDate)}? Balances will be recalculated.`)) return;
    try {
      await api.deleteLeave(leave.id);
      refresh();
    } catch (err) {
      setError(err.message);
    }
  }

  /* ---------------- Live preview ---------------- */

  const used = yearSummary?.used ?? 0;
  const thisTime = form.isSaturday ? 0 : Number(form.numDays) || 0;
  const previewBalance = { eligibility: ELIGIBILITY, used, thisTime, remaining: ELIGIBILITY - used - thisTime };
  const previewLeave = { ...profile, ...form, appliedOn: todayISO() };

  const letterLeave = printTarget ?? previewLeave;
  const letterBalance = printTarget?.balance ?? previewBalance;

  return (
    <div className="app">
      <section className="panel no-print">
        <h1>Leave Application</h1>

        <form onSubmit={handleSubmit} autoComplete="off">
          <fieldset>
            <legend>Employee</legend>
            <label>Name
              <input type="text" required placeholder="S M Faisal" value={profile.name}
                     onChange={(e) => updateProfile("name", e.target.value)} />
            </label>
            <label>Employee Number
              <input type="text" required placeholder="10039" value={profile.empNo}
                     onChange={(e) => updateProfile("empNo", e.target.value)} />
            </label>
            <div className="profile-actions">
              <button type="button" onClick={handleSaveProfile}>Save employee</button>
              {profileStatus && <span className="hint-muted">{profileStatus}</span>}
            </div>
          </fieldset>

          <fieldset>
            <legend>Leave details</legend>

            <div className="row">
              <span className="label-text">Leave type</span>
              <label className="inline">
                <input type="radio" checked={form.leaveType === "holiday"} onChange={() => update("leaveType", "holiday")} /> Holiday
              </label>
              <label className="inline">
                <input type="radio" checked={form.leaveType === "sick"} onChange={() => update("leaveType", "sick")} /> Sick Leave
              </label>
            </div>

            <div className="row">
              <span className="label-text">Is it a Saturday leave?</span>
              <label className="inline">
                <input type="radio" checked={!form.isSaturday} onChange={() => update("isSaturday", false)} /> No
              </label>
              <label className="inline">
                <input type="radio" checked={form.isSaturday} onChange={() => update("isSaturday", true)} /> Yes
              </label>
            </div>
            {form.isSaturday && <p className="hint">Saturday leave is not deducted from eligibility.</p>}

            <label>Last working day
              <input type="date" required value={form.lastWorkingDay} onChange={(e) => update("lastWorkingDay", e.target.value)} />
            </label>
            <label>Holiday starts from
              <input type="date" required value={form.startDate} onChange={(e) => update("startDate", e.target.value)} />
            </label>
            <label>Holiday end
              <input type="date" required value={form.endDate} onChange={(e) => update("endDate", e.target.value)} />
            </label>
            <label>Number of days
              <input type="number" required min="0.5" step="0.5" value={form.numDays} onChange={(e) => update("numDays", e.target.value)} />
            </label>
            <label>Re-joining date
              <input type="date" required value={form.rejoinDate} onChange={(e) => update("rejoinDate", e.target.value)} />
            </label>
            <label>Reason of leave
              <textarea rows="3" required placeholder="Casual Leave" value={form.reason} onChange={(e) => update("reason", e.target.value)} />
            </label>
          </fieldset>

          <div className="summary">
            <div>Eligibility<strong>{ELIGIBILITY}</strong></div>
            <div>Used<strong>{used}</strong></div>
            <div>This time<strong>{thisTime}</strong></div>
            <div className={previewBalance.remaining < 0 ? "negative" : ""}>
              Remaining<strong>{previewBalance.remaining}</strong>
            </div>
          </div>
          {error && <p className="error">{error}</p>}

          <div className="actions">
            <button type="submit" className="primary" disabled={saving}>
              {saving ? "Saving…" : "Save & Download PDF"}
            </button>
            <button type="button" onClick={clearForm}>Clear form</button>
          </div>
        </form>

        <h2>Already used leave</h2>
        <OpeningBalance refreshKey={refreshKey} onSaved={refresh} />

        <h2>Leave history</h2>
        <div className="history-wrap">
          <table className="history">
            <thead>
              <tr><th>Applied</th><th>From</th><th>To</th><th>Days</th><th>Sat.</th><th>Reason</th><th /></tr>
            </thead>
            <tbody>
              {leaves.length === 0 && (
                <tr><td colSpan="7" className="empty">No leave saved yet.</td></tr>
              )}
              {leaves.map((l) => (
                <tr key={l.id}>
                  <td>{formatShort(l.appliedOn)}</td>
                  <td>{formatShort(l.startDate)}</td>
                  <td>{formatShort(l.endDate)}</td>
                  <td>{l.numDays}</td>
                  <td>{l.isSaturday ? "Yes" : "No"}</td>
                  <td>{l.reason}</td>
                  <td>
                    <button type="button" className="small" onClick={() => setPrintTarget(l)}>PDF</button>{" "}
                    <button type="button" className="small danger" onClick={() => handleDelete(l)}>✕</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>

      <section className="preview">
        <Letter leave={letterLeave} balance={letterBalance} />
      </section>
    </div>
  );
}
