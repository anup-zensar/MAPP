import { useSyncExternalStore } from 'react';

/** The phone width of design system §2.2: below 600 px. */
const PHONE_QUERY = '(max-width: 599px)';

function subscribe(onChange: () => void): () => void {
  const query = window.matchMedia(PHONE_QUERY);
  query.addEventListener('change', onChange);
  return () => query.removeEventListener('change', onChange);
}

function isPhone(): boolean {
  return window.matchMedia(PHONE_QUERY).matches;
}

/**
 * True while the window is phone-sized (below 600 px). Inbox-type lists such as the approvals
 * switch from a table to stacked cards there (design system §5.4), so only one of the two is in the
 * page and screen readers never hear a row twice.
 */
export function useIsPhone(): boolean {
  return useSyncExternalStore(subscribe, isPhone, () => false);
}
