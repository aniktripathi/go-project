// Small helpers for showing money and dates.

const inr = new Intl.NumberFormat('en-IN', { style: 'currency', currency: 'INR', minimumFractionDigits: 2 });

export function formatMoney(value) {
  const n = Number(value);
  return inr.format(Number.isFinite(n) ? n : 0);
}

export function formatPercent(value) {
  const n = Number(value);
  return `${(Number.isFinite(n) ? n : 0).toFixed(1)}%`;
}

// "2026-01-10" -> "10 Jan 2026" (no timezone shifts: the string is parsed manually)
export function formatDate(iso) {
  if (!iso) return '';
  const [y, m, d] = iso.slice(0, 10).split('-').map(Number);
  const months = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
  return `${String(d).padStart(2, '0')} ${months[m - 1]} ${y}`;
}

// Today's date as YYYY-MM-DD in the user's own timezone.
export function todayISO() {
  const d = new Date();
  const mm = String(d.getMonth() + 1).padStart(2, '0');
  const dd = String(d.getDate()).padStart(2, '0');
  return `${d.getFullYear()}-${mm}-${dd}`;
}

export const MONTH_NAMES = ['January', 'February', 'March', 'April', 'May', 'June', 'July', 'August', 'September', 'October', 'November', 'December'];

// "2026-01" -> "Jan 26"
export function monthLabel(label) {
  const [y, m] = label.split('-').map(Number);
  return `${MONTH_NAMES[m - 1].slice(0, 3)} ${String(y).slice(2)}`;
}

// Checks the same rules as the backend so users get fast feedback (the server still validates).
export function validateAmount(text) {
  const s = String(text ?? '').trim();
  if (!s) return 'Amount is required.';
  if (!/^\d+(\.\d{1,2})?$|^\.\d{1,2}$/.test(s)) return 'Enter a positive amount with at most 2 decimals.';
  if (Number(s) <= 0) return 'Amount must be greater than zero.';
  if (Number(s) > 999999999.99) return 'Amount is too large.';
  return '';
}
