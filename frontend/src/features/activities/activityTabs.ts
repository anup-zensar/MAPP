import type { TabItem } from '@/components';

/**
 * The tabs of an activity's page (design system §2.4 B, plan decision 5): **Overview** (the
 * working view: Ready to submit, Financials and the line items), **Details** (details, Shell
 * attributes, goals), **Approval** and **History**. The chosen tab is kept in the address
 * (`?tab=details`); Overview has no parameter.
 */
export type ActivityTab = 'overview' | 'details' | 'approval' | 'history';

export const ACTIVITY_TABS: TabItem<ActivityTab>[] = [
  { value: 'overview', label: 'Overview' },
  { value: 'details', label: 'Details' },
  { value: 'approval', label: 'Approval' },
  { value: 'history', label: 'History' },
];

/**
 * The tab of each #part other pages link to: Home's "Open approval" (#approval), a notification
 * (#approval), "Open line items" (#line-items), a completeness chip (#ready, #completeness), the
 * Financials card (#financials) and View history (#history).
 */
const TAB_OF_PART: Record<string, ActivityTab> = {
  '#approval': 'approval',
  '#history': 'history',
  '#line-items': 'overview',
  '#financials': 'overview',
  '#ready': 'overview',
  '#completeness': 'overview',
};

function isTab(value: string | null): value is ActivityTab {
  return ACTIVITY_TABS.some((tab) => tab.value === value);
}

/** The tab an address shows: its #part's tab first, then ?tab=, else Overview. */
export function tabOf(search: string, hash: string): ActivityTab {
  const fromPart = TAB_OF_PART[hash];
  if (fromPart) {
    return fromPart;
  }
  const chosen = new URLSearchParams(search).get('tab');
  return isTab(chosen) ? chosen : 'overview';
}

/** The address's search part for a tab: "" for Overview, "?tab=details" otherwise. */
export function tabSearch(tab: ActivityTab): string {
  return tab === 'overview' ? '' : `?tab=${tab}`;
}
