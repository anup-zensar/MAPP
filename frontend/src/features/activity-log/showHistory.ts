/** The id of a record page's History section (feature 011, FR-011). */
export const HISTORY_ID = 'history';

/**
 * The id of the History part of a campaign's or initiative's side panel. The panel opens over the
 * plan page, whose own History section already uses HISTORY_ID.
 */
export function panelHistoryId(recordId: string): string {
  return `${HISTORY_ID}-${recordId}`;
}

/**
 * Brings a record's History section into view and moves the focus to its heading, so keyboard and
 * screen-reader users land there too: what **⋯ More › View history** does on the plan, campaign,
 * initiative, activity and line item pages (FR-011, design system §6.3).
 *
 * It runs a moment later, on purpose: a menu that calls it gives the focus back to its "⋯" button
 * when it closes, and that must not win over the heading (nor scroll the page back up).
 */
export function showHistory(id: string = HISTORY_ID): void {
  window.setTimeout(() => {
    const section = document.getElementById(id);
    if (!section) {
      return;
    }
    const lessMotion = window.matchMedia?.('(prefers-reduced-motion: reduce)').matches ?? false;
    section.scrollIntoView?.({ behavior: lessMotion ? 'auto' : 'smooth', block: 'start' });
    const heading = section.querySelector<HTMLElement>('h2, h3') ?? section;
    heading.setAttribute('tabindex', '-1');
    heading.focus({ preventScroll: true });
  }, 0);
}

/**
 * The Activity log with its Record filter set to one record, and with the items inside it when
 * `includeInside` (FR-013): **Open in activity log** of a History section.
 */
export function historyHref(recordId: string, includeInside = false): string {
  const params: Record<string, string> = includeInside
    ? { record: recordId, inside: '1' }
    : { record: recordId };
  return `/activity-log?${new URLSearchParams(params)}`;
}
