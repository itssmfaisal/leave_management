import DatePicker from "react-datepicker";
import "react-datepicker/dist/react-datepicker.css";

// "2026-10-21" <-> Date (local midnight), so the form keeps ISO strings.
const toDate = (iso) => {
  if (!iso) return null;
  const [y, m, d] = iso.split("-").map(Number);
  return new Date(y, m - 1, d);
};
const toISO = (date) =>
  date
    ? `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`
    : "";

// Shown as "21 October, 2026" (same as the letter); typing 21/10/2026 also works.
const FORMATS = ["d MMMM, yyyy", "dd/MM/yyyy", "d/M/yyyy"];

export default function DateField({ id, label, value, onChange, min, max, rangeStart, rangeEnd, selects }) {
  return (
    <div className="field">
      <label htmlFor={id}>{label}</label>
      <DatePicker
        id={id}
        required
        autoComplete="off"
        placeholderText="Select a date"
        dateFormat={FORMATS}
        selected={toDate(value)}
        onChange={(date) => onChange(toISO(date))}
        minDate={toDate(min)}
        maxDate={toDate(max)}
        selectsStart={selects === "start"}
        selectsEnd={selects === "end"}
        startDate={toDate(rangeStart)}
        endDate={toDate(rangeEnd)}
        calendarStartDay={6} // week starts on Saturday (Bangladesh work week)
        showMonthDropdown
        showYearDropdown
        dropdownMode="select"
        todayButton="Today"
      />
    </div>
  );
}
