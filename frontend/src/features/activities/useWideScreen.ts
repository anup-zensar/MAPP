import { useSyncExternalStore } from 'react';

/** The width from which form pages show their right rail (design system §2.2, §2.4 C). */
const WIDE = '(min-width: 1280px)';

function subscribe(onChange: () => void): () => void {
  const query = window.matchMedia(WIDE);
  query.addEventListener?.('change', onChange);
  return () => query.removeEventListener?.('change', onChange);
}

/** Whether the window is at least 1280 px wide: the form rail shows instead of the count line. */
export function useWideScreen(): boolean {
  return useSyncExternalStore(
    subscribe,
    () => window.matchMedia(WIDE).matches,
    () => false,
  );
}
