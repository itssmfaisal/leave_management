// All dates are handled as "YYYY-MM-DD" strings to avoid time-zone shifts.

const MONTHS = [
  "January", "February", "March", "April", "May", "June",
  "July", "August", "September", "October", "November", "December",
];

function toISO(y, m, d) {
  return `${y}-${String(m + 1).padStart(2, "0")}-${String(d).padStart(2, "0")}`;
}

function parseISO(iso) {
  const [y, m, d] = iso.split("-").map(Number);
  return { y, m: m - 1, d };
}

export function todayISO() {
  const d = new Date();
  return toISO(d.getFullYear(), d.getMonth(), d.getDate());
}

export function yearOf(iso) {
  return iso ? parseISO(iso).y : new Date().getFullYear();
}

export function addDays(iso, n) {
  const { y, m, d } = parseISO(iso);
  const dt = new Date(Date.UTC(y, m, d + n));
  return toISO(dt.getUTCFullYear(), dt.getUTCMonth(), dt.getUTCDate());
}

export function daysInclusive(startIso, endIso) {
  const a = parseISO(startIso);
  const b = parseISO(endIso);
  return Math.round((Date.UTC(b.y, b.m, b.d) - Date.UTC(a.y, a.m, a.d)) / 86400000) + 1;
}

// "2026-09-29" -> "29 September, 2026"
export function formatDate(iso) {
  if (!iso) return "";
  const { y, m, d } = parseISO(iso);
  return `${d} ${MONTHS[m]}, ${y}`;
}

// "2026-09-29" -> "29 Sep 26"
export function formatShort(iso) {
  if (!iso) return "";
  const { y, m, d } = parseISO(iso);
  return `${d} ${MONTHS[m].slice(0, 3)} ${String(y).slice(2)}`;
}

// 1 -> "01", 1.5 -> "1.5"
export function formatDays(n) {
  if (n === "" || n == null || isNaN(n)) return "";
  const num = Number(n);
  return Number.isInteger(num) ? String(num).padStart(2, "0") : String(num);
}
