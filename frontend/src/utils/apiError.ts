/**
 * extractApiError returns a human-readable message from an axios error.
 *
 * Resolution order:
 *   1. err.translatedMessage — set by our axios interceptor in api/client.ts
 *      after looking up the backend error code in the i18n bundle.
 *   2. err.response.data.message — raw backend message.
 *   3. err.response.status === 413 → fixed "File too large" hint (no body
 *      from nginx/gin when the request is rejected by size).
 *   4. err.message — last-resort axios message (e.g. "Network Error").
 *   5. undefined — caller should fall back to a generic translated string.
 */
export function extractApiError(err: unknown): string | undefined {
  if (!err || typeof err !== 'object') return undefined;
  const e = err as {
    translatedMessage?: string;
    message?: string;
    response?: { status?: number; data?: { message?: string; code?: string } | string };
  };

  if (e.translatedMessage) return e.translatedMessage;

  const data = e.response?.data;
  if (data && typeof data === 'object' && data.message) {
    return data.message;
  }

  if (e.response?.status === 413) return 'File too large';

  if (e.message) return e.message;

  return undefined;
}
