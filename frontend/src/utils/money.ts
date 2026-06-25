/**
 * Format an integer NT$ amount for display, e.g. 12345 → "NT$ 12,345".
 * Null/undefined coalesces to 0 so callers don't each re-implement the guard.
 */
export function formatNT(amount?: number | null): string {
  return `NT$ ${(amount ?? 0).toLocaleString()}`;
}
