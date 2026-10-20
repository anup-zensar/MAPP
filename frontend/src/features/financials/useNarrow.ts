import { useSyncExternalStore } from 'react';

/** Below this width the transactions table becomes one card per transaction (015 plan). */
export const NARROW_QUERY = '(max-width: 639px)';

function subscribe(onChange: () => void): () => void {
  const query = window.matchMedia(NARROW_QUERY);
  query.addEventListener('change', onChange);
  return () => query.removeEventListener('change', onChange);
}

/** True on a phone-width window; follows the window as it is resized. */
export function useNarrow(): boolean {
  return useSyncExternalStore(
    subscribe,
    () => window.matchMedia(NARROW_QUERY).matches,
    () => false,
  );
}
