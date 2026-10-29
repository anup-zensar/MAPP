import { formatWhole } from '@/format/decimal';
import { plural } from '@/format/text';
import type { FinanceFilter, Filters, PlanType } from './all-work-types';

export function isLabelled<T extends string>(labels: Record<T, string>) {
  return (value: string): value is T => Object.hasOwn(labels, value);
}

export const FILTER_LABELS = {
  year: 'Year',
  years: 'Years',
  plans: 'Plan',
  planStatuses: 'Plan status',
  containerStatuses: 'Campaign / initiative status',
  activityStatuses: 'Activity status',
  owners: 'Owner',
  teams: 'Team',
  businessUnits: 'Business unit',
  activityTypes: 'Activity type',
  strategic: 'Strategic / AOP plans',
  bcpir: 'BCPIR plans',
  showLevels: 'Show levels',
  attributes: 'Shell attribute',
  planTypes: 'Plan type',
  planCurrencies: 'Plan currency',
  finance: 'Financials',
} as const;

export const PLAN_TYPE_LABELS: Record<PlanType, string> = {
  operational: 'Operational',
  strategic: 'Strategic',
  crossMarket: 'Cross-market',
};

export const FINANCE_LABELS: Record<FinanceFilter, string> = {
  hasPurchaseOrder: 'Has a PO',
  noPurchaseOrder: 'No PO yet',
  invoiced: 'Invoiced',
  committedAbovePlanned: 'Committed above planned cost',
};

export const MAX_YEARS = 5;

export const TOO_MANY_YEARS = 'Choose at most 5 years.';

export function yearsText(from: number, to: number): string {
  return to > from ? `${from}–${to}` : String(from);
}

export const PLAN_STATUS_LABELS: Record<Filters['planStatuses'][number], string> = {
  draft: 'Draft',
  published: 'Published',
};

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

export const ALL_WORK_DESCRIPTION = 'Filter, compare and export all planned work.';

export const VIEW_LABELS = {
  grid: 'Grid',
  calendar: 'Calendar',
  budget: 'Budget & cost',
} as const;

export const PERIOD_LABELS = {
  month: 'Monthly',
  quarter: 'Quarterly',
  year: 'Yearly',
} as const;

export const COMPARE_PHASED = 'Compare with phased budget';
export const PHASED_BUDGET_ROW = 'Phased budget';
export const NOT_PHASED = 'Not phased';

export const COLUMN_TIPS: Partial<Record<string, string>> = {
  unassigned:
    'Planned cost not yet split into line items. Only positive balances are added: an activity whose line items are above its planned cost counts as zero here.',
};

export const UNASSIGNED_ROW_NAME = 'Unassigned planned cost (Sys_Gen_Balance)';

export const CONTEXT_LABEL = 'Shown for context';
export const CONTEXT_TIP =
  "This row doesn't match the filters; it's shown so you can see where the matching rows are.";

export function countText(shown: number, total: number, noun: string): string {
  const singular = noun === 'activities' ? 'activity' : noun.replace(/s$/, '');
  return `${formatWhole(shown)} of ${plural(total, singular, noun)}`;
}

export function totalText(activities: number): string {
  return `Total · ${plural(activities, 'activity', 'activities')}`;
}

export function negativeBalanceText(count: number): string {
  return count === 1
    ? '1 activity has line items above its planned cost'
    : `${count} activities have line items above their planned cost`;
}

export function leftOutMessage(names: string[]): string {
  return `Some parts of this view no longer exist and were left out: ${names.join(', ')}.`;
}

export const VIEW_GONE = 'This view no longer exists — it may have been deleted in another tab.';
