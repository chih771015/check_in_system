/**
 * Format an integer amount for display, e.g. 12345 → "12,345".
 * Currency is intentionally omitted for now (待定幣別).
 * Null/undefined coalesces to 0 so callers don't each re-implement the guard.
 */
export function formatNT(amount?: number | null): string {
  return (amount ?? 0).toLocaleString();
}
