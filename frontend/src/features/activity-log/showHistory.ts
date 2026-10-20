/** The id of a record page's History section (feature 011, FR-011). */
export const HISTORY_ID = 'history';

/**
 * Brings the record page's History section into view and moves the focus to its heading, so
 * keyboard and screen-reader users land there too: what **⋯ More › View history** does on the
 * plan, campaign, initiative, activity and line item pages (FR-011, design system §6.3).
 */
export function showHistory(): void {
  const section = document.getElementById(HISTORY_ID);
  if (!section) {
    return;
  }
  const lessMotion = window.matchMedia?.('(prefers-reduced-motion: reduce)').matches ?? false;
  section.scrollIntoView?.({ behavior: lessMotion ? 'auto' : 'smooth', block: 'start' });
  const heading = section.querySelector<HTMLElement>('h2') ?? section;
  heading.setAttribute('tabindex', '-1');
  heading.focus({ preventScroll: true });
}

/**
 * The Activity log with its Record filter set to one record, and with the items inside it when
 * `includeInside` (FR-013): **Open in activity log** of a History section, and **⋯ More › View
 * history** on the record pages.
 */
export function historyHref(recordId: string, includeInside = false): string {
  const params: Record<string, string> = includeInside
    ? { record: recordId, inside: '1' }
    : { record: recordId };
  return `/activity-log?${new URLSearchParams(params)}`;
}
