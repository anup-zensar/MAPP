/**
 * The words of plans on screen (plan "User-visible messages"): statuses, spreads, goal stages,
 * approval progress, counts and the plan's dates. Pages and tests quote these, never their own copies.
 */
import type { components } from '@/api/generated/schema';
import type { StatusTone } from '@/ui/StatusChip/StatusChip';
import { joinWithAnd, plural } from '@/format/text';

export type PlanStatus = components['schemas']['PlanStatus'];
export type SpreadMethod = components['schemas']['SpreadMethod'];
export type GoalStage = components['schemas']['GoalStage'];
export type ApprovalProgress = components['schemas']['ApprovalProgress'];
type PlanContents = components['schemas']['PlanContents'];
type RecordRef = components['schemas']['RecordRef'];

/** "Draft" (neutral grey) or "Published" (positive green): design system §4. */
export const STATUS: Record<PlanStatus, { word: string; tone: StatusTone }> = {
  draft: { word: 'Draft', tone: 'neutral' },
  published: { word: 'Published', tone: 'positive' },
};

/** The default line-item spread choices, in the form's order. */
export const SPREAD_LABELS: Record<SpreadMethod, string> = {
  evenly: 'Evenly',
  firstMonth: 'All in first month',
  lastMonth: 'All in last month',
  custom: 'Custom',
};

/** The goal stage choices of the form. */
export const STAGE_OPTIONS: Record<GoalStage, string> = {
  inq: 'INQ (inquiries)',
  mql: 'MQL (marketing-qualified leads)',
  revenue: 'Revenue',
};

/** The stage as a short word in tables and sentences. */
export const STAGE_WORDS: Record<GoalStage, string> = {
  inq: 'INQ',
  mql: 'MQL',
  revenue: 'Revenue',
};

/** "Goal amount (INQ)", "Goal amount (MQL)", "Goal amount (Revenue, USD)". */
export function goalAmountLabel(stage: GoalStage): string {
  return stage === 'revenue' ? 'Goal amount (Revenue, USD)' : `Goal amount (${STAGE_WORDS[stage]})`;
}

/** The detail under "Activities approved" and its tone. */
export const APPROVAL_PROGRESS: Record<ApprovalProgress, { word: string; tone: StatusTone }> = {
  noActivities: { word: 'No activities yet', tone: 'neutral' },
  inProgress: { word: 'Partly approved', tone: 'review' },
  allApproved: { word: 'All approved', tone: 'positive' },
};

/**
 * What a plan holds, in words: "2 campaigns, 4 initiatives, 7 activities and 15 line items" (only
 * the levels it has); "" when it holds nothing.
 */
export function contentsText(contents: PlanContents): string {
  const parts = [
    [contents.campaigns, 'campaign', 'campaigns'],
    [contents.initiatives, 'initiative', 'initiatives'],
    [contents.activities, 'activity', 'activities'],
    [contents.lineItems, 'line item', 'line items'],
  ] as const;
  return joinWithAnd(
    parts.filter(([count]) => count > 0).map(([count, one, many]) => plural(count, one, many)),
  );
}

/**
 * A plan's dates in the tree, its whole year, as every date range reads (D-216): "01 Jan – 31 Dec
 * 2026" with `dateRange` of useFormatters(), in the person's date format.
 */
export function yearDates(year: number, dateRange: (start: string, end: string) => string): string {
  return dateRange(`${year}-01-01`, `${year}-12-31`);
}

/** "<n> of <m> activities approved" on cards and rows. */
/** "4 of 7 activities approved"; "No activities yet" rather than "0 of 0" (functional review X-08). */
export function activitiesApprovedText(approved: number, activities: number): string {
  if (activities === 0) {
    return 'No activities yet';
  }
  return `${approved} of ${activities} ${activities === 1 ? 'activity' : 'activities'} approved`;
}

/**
 * The heading of one record's group in the publish dialog and the items-below list: "Plan" for
 * the plan itself, "Campaign 'Retail growth'" for a record below it.
 */
export function groupLabel(record: RecordRef): string {
  if (record.type === 'plan') {
    return 'Plan';
  }
  return `${typeWord(record.type)} '${record.name}'`;
}

/** "Campaign", "Initiative", "Activity", "Line item" for a record type of the contract. */
export function typeWord(type: string): string {
  if (type === 'lineItem') {
    return 'Line item';
  }
  return `${type.charAt(0).toUpperCase()}${type.slice(1)}`;
}

/** "1 field needed" / "3 fields needed". */
export function fieldsNeededText(count: number): string {
  return count === 1 ? '1 field needed' : `${count} fields needed`;
}

/** Under the read-only year of the edit form (FR-018). */
export const YEAR_LOCKED =
  "Year can't change once the plan is saved — copy the plan into the next year or create a new plan.";

/** At the top of a published plan's edit form (FR-022). */
export const PUBLISHED_BANNER =
  'This plan is published. Changes are saved straight away and recorded in the activity log.';

/** Under a changed default finance budget or spread on the edit form (FR-022). */
export const APPLIES_FROM_NOW = 'Applies to line items created from now on.';

/** The plan page and the edit page when the plan is gone. */
export const NOT_FOUND_TITLE = 'Plan not found';
export const NOT_FOUND_TEXT = 'This plan no longer exists — someone may have deleted it.';

/** Under the list's toolbar while "My teams" is on (FR-004). */
export const MY_TEAMS_HINT =
  "Shows your teams' plans. Everyone can still open every plan in this phase.";

/** Under the plan row while the plan has no campaigns (FR-014). */
export const NO_CAMPAIGNS =
  'No campaigns yet — a campaign is a theme that runs for most of the year.';

type PlanAllocation = components['schemas']['PlanAllocation'];

/** "Allocated from 'FY2026 Global marketing budget'": the plan budget card's note (014 FR-031). */
export function allocatedFromText(allocation: PlanAllocation): string {
  return `Allocated from '${allocation.globalBudget.name}'`;
}

/**
 * Why an allocated plan's budget can't change (014 FR-031, message 23): the API's reason when it
 * sends one, else "Allocated from '<global budget>' — ask <owner> to change it."
 */
export function allocationLockText(allocation: PlanAllocation, reason: string | undefined): string {
  if (reason) {
    return reason;
  }
  const whom = allocation.owner ? allocation.owner.name : "the global budget's owner";
  return `${allocatedFromText(allocation)} — ask ${whom} to change it.`;
}

/** Under a cleared budget that was phased (016 FR-022). */
export const PHASING_REMOVED = 'Budget by month will be removed when you save.';
