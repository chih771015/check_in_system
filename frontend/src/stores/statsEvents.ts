/**
 * Tiny pub/sub so deep admin actions that change actual-paid amounts can ask
 * the far-away AppLayout monthly-total banner to refresh, without threading a
 * callback through the whole component tree.
 *
 * Lightweight by design (做法 A): only the most common entry point — an admin
 * setting an actual amount — emits here. Rarer amount-affecting actions
 * (mark-no-show, schedule delete) are caught by the banner's refetch-on-focus
 * fallback rather than wiring every call site.
 */
type Listener = () => void;

const listeners = new Set<Listener>();

/** Notify subscribers that actual-paid amounts changed (banner should refetch). */
export function notifyActualAmountChanged(): void {
  listeners.forEach((l) => l());
}

/** Subscribe to actual-amount-change notifications. Returns an unsubscribe fn. */
export function onActualAmountChanged(listener: Listener): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}
