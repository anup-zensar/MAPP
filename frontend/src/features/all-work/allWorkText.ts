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

/** Approval progress first, then Timing (domain model §2.2); the two groups of one filter. */
export const CONTAINER_STATUS_LABELS: Record<Filters['containerStatuses'][number], string> = {
  noActivities: 'No activities',
  inProgress: 'In progress',
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

/** The three views, as the pill tabs name them (FR-001). */
export const VIEW_LABELS = {
  grid: 'Grid',
  calendar: 'Calendar',
  budget: 'Budget & cost',
} as const;

/** The name of an unassigned planned cost row (FR-012). */
export const UNASSIGNED_ROW_NAME = 'Unassigned planned cost (Sys_Gen_Balance)';

export const CONTEXT_LABEL = 'Shown for context';
export const CONTEXT_TIP =
  "This row doesn't match the filters; it's shown so you can see where the matching rows are.";

/** "Showing 12 of 40 activities" (FR-008); "activity" when there is only one of them. */
export function countText(shown: number, total: number, noun: string): string {
  const singular = noun === 'activities' ? 'activity' : noun.replace(/s$/, '');
  return `Showing ${formatWhole(shown)} of ${plural(total, singular, noun)}`;
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
