import { useEffect, useState } from "react";
import { api } from "./api.js";

// Lets the user enter leave days already taken before using this app.
export default function OpeningBalance({ onSaved, refreshKey }) {
  const [year, setYear] = useState(new Date().getFullYear());
  const [summary, setSummary] = useState(null);
  const [used, setUsed] = useState("");
  const [status, setStatus] = useState("");

  useEffect(() => {
    if (!(year >= 2000 && year <= 2100)) return;
    api.getYear(year)
      .then((s) => {
        setSummary(s);
        setUsed(String(s.openingUsed));
      })
      .catch((e) => setStatus(e.message));
  }, [year, refreshKey]);

  async function save(e) {
    e.preventDefault();
    try {
      const s = await api.setOpeningUsed(year, Number(used));
      setSummary(s);
      setStatus("Saved.");
      onSaved();
    } catch (err) {
      setStatus(err.message);
    }
  }

  return (
    <form className="opening" onSubmit={save}>
      <p className="hint-muted">
        Days already used <em>before</em> you started using this app. They count towards “Used”.
      </p>
      <div className="opening-row">
        <label>Year
          <input type="number" value={year} min="2000" max="2100"
                 onChange={(e) => { setYear(Number(e.target.value)); setStatus(""); }} />
        </label>
        <label>Already used days
          <input type="number" value={used} min="0" max="21" step="0.5" required
                 onChange={(e) => { setUsed(e.target.value); setStatus(""); }} />
        </label>
        <button type="submit">Save</button>
      </div>
      {summary && (
        <p className="hint-muted">
          {summary.year}: {summary.openingUsed} opening + {summary.appUsed} from this app ={" "}
          <strong>{summary.used}</strong> used, <strong>{summary.remaining}</strong> remaining.
        </p>
      )}
      {status && <p className="hint-muted">{status}</p>}
    </form>
  );
}
