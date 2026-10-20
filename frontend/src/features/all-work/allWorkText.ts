/**
 * The words of All work that more than one file uses (plan "User-visible messages"). Tests quote
 * them, so keep them exactly as the plan's table writes them.
 */
import { formatWhole } from '@/lib/decimal';
import { plural } from '@/lib/text';
import type { Filters } from './allWorkTypes';

/** The filters of the filter bar (FR-004), by their place in the filter set. */
export const FILTER_LABELS = {
  year: 'Year',
  plans: 'Plan',
  planStatuses: 'Plan status',
  containerStatuses: 'Campaign / initiative status',
  activityStatuses: 'Activity status',
  owners: 'Owner',
  teams: 'Team',
  businessUnits: 'Business unit',
  activityTypes: 'Activity type',
  strategic: 'Strategic / AOP plans',
  showLevels: 'Show levels',
  attributes: 'Shell attribute',
  planTypes: 'Plan type',
  planCurrencies: 'Plan currency',
} as const;

export const PLAN_STATUS_LABELS: Record<Filters['planStatuses'][number], string> = {
  draft: 'Draft',
  published: 'Published',
};

/**
 * Approval progress first, then Timing (domain model §2.2); the two groups of one filter. The
 * roll-up "inProgress" reads "Partly approved": "In progress" is a running activity (statusWords).
 */
export const CONTAINER_STATUS_LABELS: Record<Filters['containerStatuses'][number], string> = {
  noActivities: 'No activities',
  inProgress: 'Partly approved',
  allApproved: 'All approved',
  notStarted: 'Not started',
  running: 'Running',
  finished: 'Finished',
};

export const ACTIVITY_STATUS_LABELS: Record<Filters['activityStatuses'][number], string> = {
  created: 'Created',
  submitted: 'Submitted',
  declined: 'Declined',
  approved: 'Approved',
  inProgress: 'In progress',
  complete: 'Complete',
};

export const STRATEGIC_LABELS: Record<Filters['strategic'], string> = {
  include: 'Include',
  only: 'Only',
  exclude: 'Exclude',
};

export const SHOW_LEVELS_LABELS: Record<Filters['showLevels'], string> = {
  everything: 'Everything',
  completeBranches: 'Complete branches only',
  campaignsWithoutInitiatives: 'Campaigns without initiatives',
  initiativesWithoutActivities: 'Initiatives without activities',
};

/** The one sentence under the title "All work" (design system §2.3: the title is the menu name). */
export const ALL_WORK_DESCRIPTION =
  'Every campaign, initiative, activity and line item of your plans: filter, compare and export.';

/** The three views, as the tabs name them (FR-001). */
export const VIEW_LABELS = {
  grid: 'Grid',
  calendar: 'Calendar',
  budget: 'Budget & cost',
} as const;

/**
 * Tips on Grid headings whose sums need explaining. Unassigned planned cost adds positive balances
 * only, so Planned cost − Line items can differ from it (functional review X-02).
 */
export const COLUMN_TIPS: Partial<Record<string, string>> = {
  unassigned:
    'Planned cost not yet split into line items. Only positive balances are added: an activity whose line items are above its planned cost counts as zero here.',
};

/** The name of an unassigned planned cost row (FR-012). */
export const UNASSIGNED_ROW_NAME = 'Unassigned planned cost (Sys_Gen_Balance)';

export const CONTEXT_LABEL = 'Shown for context';
export const CONTEXT_TIP =
  "This row doesn't match the filters; it's shown so you can see where the matching rows are.";

/**
 * "12 of 40 activities" (FR-008), written like every list's count (design system §11 rule 14);
 * "activity" when there is only one of them.
 */
export function countText(shown: number, total: number, noun: string): string {
  const singular = noun === 'activities' ? 'activity' : noun.replace(/s$/, '');
  return `${formatWhole(shown)} of ${plural(total, singular, noun)}`;
}

/** "Total · 12 activities" (FR-014). */
export function totalText(activities: number): string {
  return `Total · ${plural(activities, 'activity', 'activities')}`;
}

/** The red line of the Unassigned planned cost card (FR-003). */
export function negativeBalanceText(count: number): string {
  return count === 1
    ? '1 activity has line items above its planned cost'
    : `${count} activities have line items above their planned cost`;
}

/** The message after applying a saved view that names something that no longer exists (FR-031). */
export function leftOutMessage(names: string[]): string {
  return `Some parts of this view no longer exist and were left out: ${names.join(', ')}.`;
}

/** A saved view that was deleted meanwhile, for example in another tab (FR-031). */
export const VIEW_GONE = 'This view no longer exists — it may have been deleted in another tab.';
