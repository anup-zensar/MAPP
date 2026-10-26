/**
 * Every text feature 005 shows (plan "User-visible messages (web)"). The API's own messages are
 * shown as received; these are the web app's words around them.
 */
import { joinWithAnd, plural } from '@/format/text';

/** The two levels of this feature. */
export type RecordLevel = 'campaign' | 'initiative';

/** "Campaign", "Initiative": the level at the start of a sentence. */
export const LEVEL_WORD: Record<RecordLevel, string> = {
  campaign: 'Campaign',
  initiative: 'Initiative',
};

// Actions
export const ADD_CAMPAIGN = 'Add campaign';
export const ADD_INITIATIVE = 'Add initiative';
export const VIEW = 'View';
export const EDIT = 'Edit';
export const DELETE = 'Delete';
export const COPY = 'Copy';
export const MOVE_TO = 'Move to…';
export const VIEW_HISTORY = 'View history';
export const BACK_TO_PLAN = 'Back to the plan';
export const SAVE = 'Save';
export const SAVE_AND_ADD_ANOTHER = 'Save and add another';
export const CANCEL = 'Cancel';

/** The campaign row's quick action (tooltip and accessible name). */
export function addInitiativeTo(campaign: string): string {
  return `Add initiative to ${campaign}`;
}

// Counts
export function campaignsCount(count: number): string {
  return plural(count, 'campaign');
}
export function initiativesCount(count: number): string {
  return plural(count, 'initiative');
}
export function activitiesCount(count: number): string {
  return plural(count, 'activity', 'activities');
}
export function lineItemsCount(count: number): string {
  return plural(count, 'line item');
}

/** Under "Total" in the totals row: "2 campaigns · 4 initiatives · 7 activities". */
export function totalsText(campaigns: number, initiatives: number, activities: number): string {
  return `${campaignsCount(campaigns)} · ${initiativesCount(initiatives)} · ${activitiesCount(activities)}`;
}
/** Under the Total row's budget: the campaign budgets added up (short, so the tree fits 1,440 px). */
export const ALLOCATED = 'Allocated';

// Budget
/** Under a planned cost in the plan tree: "38% used", as on the plan cards (short, so the tree fits 1,440 px). */
export function percentOfBudget(percent: number): string {
  return `${percent}% used`;
}
/** Under a budget in the plan tree: what its children may still be given (PL-03, one wording). */
export function remainingText(amount: string): string {
  return `Remaining ${amount}`;
}
export function overAllocatedBy(amount: string): string {
  return `Over-allocated by ${amount}`;
}
/** The tree row's short form of "Over-allocated by …", under the budget (FR-019). */
export function overBy(amount: string): string {
  return `Over by ${amount}`;
}
export const REMAINING_PLAN_BUDGET = 'Remaining plan budget';
export const REMAINING_CAMPAIGN_BUDGET = 'Remaining campaign budget';
export const NO_BUDGET_TO_ALLOCATE_FROM = 'No budget to allocate from';
export const REMAINING_PLAN_BUDGET_TIP =
  "The plan's budget minus the budgets of all its campaigns, including this one.";
export const REMAINING_CAMPAIGN_BUDGET_TIP =
  "The campaign's budget minus the budgets of all its initiatives, including this one.";

/** R4: the campaign's dates leave the plan's year (a warning; it never blocks). */
export function outsidePlanYear(year: number): string {
  return `This campaign runs outside ${year}, the plan's year`;
}

// Status chips
export const NO_ACTIVITIES = 'No activities';
/** The roll-up while some activities below wait (design system §4.1; never "In progress"). */
export const PARTLY_APPROVED = 'Partly approved';
export const ALL_APPROVED = 'All approved';
/** "3 of 5 approved": the roll-up chip of a row or panel (functional review A-02). */
export function approvedOf(approved: number, total: number): string {
  return `${approved} of ${total} approved`;
}
/** "3 of 5 activities approved", "0 of 1 activity approved": the chip's full meaning. */
export function activitiesApprovedOf(approved: number, total: number): string {
  return `${approved} of ${total} ${total === 1 ? 'activity' : 'activities'} approved`;
}
export const TIMING_WORD = {
  notStarted: 'Not started',
  running: 'Running',
  finished: 'Finished',
} as const;

// Panels
export const NEW_CAMPAIGN = 'New campaign';
export const NEW_INITIATIVE = 'New initiative';
export const EDIT_CAMPAIGN = 'Edit campaign';
export const EDIT_INITIATIVE = 'Edit initiative';
export function insidePlan(plan: string): string {
  return `Inside plan '${plan}'`;
}
export function insideCampaign(campaign: string): string {
  return `Inside campaign '${campaign}'`;
}

// Forms
export const BASICS = 'Basics';
export const BUDGET = 'Budget';
export const NAME_LABEL: Record<RecordLevel, string> = {
  campaign: 'Campaign name',
  initiative: 'Initiative name',
};
export const BUDGET_LABEL: Record<RecordLevel, string> = {
  campaign: 'Campaign budget',
  initiative: 'Initiative budget',
};
export const PLAN = 'Plan';
export const CAMPAIGN = 'Campaign';
export const OWNER = 'Owner';
export const DATES = 'Dates';
export const DESCRIPTION = 'Description';
export const FROM_THE_PLAN = 'From the plan';
export const THE_PLANS_YEAR = "The plan's year";
export const FROM_THE_CAMPAIGN = 'From the campaign';
export const THE_CAMPAIGNS_DATES = "The campaign's dates";
export const CHANGE_IT_WITH_MOVE = 'Change it with Move';
/** R3: "Must be inside the campaign's dates (01 Mar 2026 – 31 Dec 2026)". */
export function allowedRange(range: string): string {
  return `Must be inside the campaign's dates (${range})`;
}
export const CONFLICT_NEXT_STEP =
  'Your changes are still here. Save again to replace their changes, or Cancel to see theirs.';

// Toasts
export function createdToast(level: RecordLevel, name: string): string {
  return `${LEVEL_WORD[level]} '${name}' created`;
}
export function savedToast(level: RecordLevel, name: string): string {
  return `${LEVEL_WORD[level]} '${name}' saved`;
}
export function openName(name: string): string {
  return `Open ${name}`;
}

// Details
export const ID = 'ID';
export function createdLine(
  createdBy: string,
  createdOn: string,
  changedBy: string,
  changedOn: string,
): string {
  return `Created by ${createdBy} on ${createdOn}, last changed by ${changedBy} on ${changedOn}`;
}
export const BUDGET_AND_COST = 'Budget and cost';
export const ALLOCATED_TO_INITIATIVES = 'Allocated to initiatives';
export const REMAINING_FOR_INITIATIVES = 'Remaining for initiatives';
export const PLANNED_COST = 'Planned cost';
export const CALCULATED = 'Calculated';
export const BUDGET_USED = 'Budget used';
export const LEFT_TO_PLAN = 'Left to plan';
export const NO_PLANNED_COST = 'No planned cost below yet';
export const NO_BUDGET_SET = 'No budget set';
export const PLANNED_COST_TIP: Record<RecordLevel, string> = {
  campaign: 'Sum of the planned costs of its initiatives.',
  initiative: 'Sum of the planned costs of its activities.',
};
export const BUDGET_USED_TIP = 'Planned cost ÷ budget.';
export const LEFT_TO_PLAN_TIP = 'Budget − planned cost.';
export const GOALS = 'Goals';
export const GOALS_TIP = 'Sum of the goals of the activities below.';
export const INQ = 'INQ';
export const MQL = 'MQL';
export const REVENUE = 'Revenue';
export function initiativesHeading(count: number): string {
  return `Initiatives (${count})`;
}
export const NO_INITIATIVES =
  'No initiatives yet — an initiative groups similar work inside a campaign.';
export function activitiesLine(activities: number, approved: number, lineItems: number): string {
  return `Activities: ${activities} (${approved} approved) · Line items: ${lineItems}`;
}
export const NO_ACTIVITIES_YET = 'No activities yet.';
export function notFound(level: RecordLevel): string {
  return `This ${level} no longer exists — someone may have deleted it.`;
}

// Delete
export interface ContentCounts {
  initiatives?: number;
  activities: number;
  approvedActivities: number;
  lineItems: number;
}

/**
 * "2 initiatives, 7 activities and 15 line items": only the counts that are not zero, singular
 * for 1; "" when nothing is below.
 */
export function contentsSentence(counts: ContentCounts): string {
  const parts: string[] = [];
  if (counts.initiatives) {
    parts.push(initiativesCount(counts.initiatives));
  }
  if (counts.activities) {
    parts.push(activitiesCount(counts.activities));
  }
  if (counts.lineItems) {
    parts.push(lineItemsCount(counts.lineItems));
  }
  return joinWithAnd(parts);
}
export function deleteTitle(level: RecordLevel, name: string): string {
  return `Delete ${level} '${name}'?`;
}
export function withItGo(contents: string): string {
  return `With it go ${contents}.`;
}
export const PLAN_TOTALS_WILL_UPDATE = 'Plan totals will update.';
/** R14: approved activities come back as Created when restored later. */
export function approvedSentence(approved: number): string {
  return approved === 1
    ? '1 of these activities is approved. If it is restored later from the recycle bin, it returns as Created and needs approval again.'
    : `${approved} of these activities are approved. If they are restored later from the recycle bin, they return as Created and need approval again.`;
}
export const MOVES_TO_RECYCLE_BIN = 'It moves to the recycle bin; you can restore it.';
export function deleteLabel(level: RecordLevel): string {
  return `Delete ${level}`;
}

// Move and copy (US6)
export function moveTitle(level: RecordLevel, name: string): string {
  return `Move ${level} '${name}'`;
}
export const MOVE_INTRO: Record<RecordLevel, string> = {
  campaign:
    'Choose the plan to move it to. Its initiatives, activities and line items move with it and keep their IDs, dates and statuses.',
  initiative:
    'Choose the campaign to move it to. Its activities and line items move with it and keep their IDs, dates and statuses.',
};
/** The move dialog without any destination (edge case "No possible destination exists"). */
export function nothingToMoveTo(level: RecordLevel, year: number): string {
  return level === 'campaign'
    ? `There's no other plan in ${year} to move this campaign to.`
    : "There's no other campaign in this plan to move this initiative to.";
}
export const CHOOSE_WHERE_TO_MOVE = 'Choose where to move it.';
export function moveLabel(level: RecordLevel): string {
  return `Move ${level}`;
}
export function movedToast(level: RecordLevel, name: string, destination: string): string {
  return `${LEVEL_WORD[level]} '${name}' moved to '${destination}'`;
}
/** Under the chosen destination when the record's budget would over-allocate it (FR-026). */
export function overAllocateWarning(destination: string, amount: string): string {
  return `This will over-allocate '${destination}' by ${amount}.`;
}
/** FR-024: what changes when a campaign lands in a plan with another model or business unit. */
export function otherModelOrUnit(plan: string): string {
  return `'${plan}' uses another model or business unit:`;
}
export function typesNotAllowedText(model: string, activities: string): string {
  return `These activities have a type the model '${model}' doesn't allow. They keep their type but need an allowed type before they can be submitted: ${activities}`;
}
export function goalsRecalculated(model: string): string {
  return `Goals will be recalculated with the model '${model}'.`;
}
export function hiddenThere(groups: string): string {
  return `Hidden there, values kept: ${groups}`;
}
export function shownThere(groups: string): string {
  return `Shown there: ${groups}`;
}
export function copyTitle(level: RecordLevel, name: string): string {
  return `Copy ${level} '${name}'`;
}
export const NAME_OF_THE_COPY = 'Name of the copy';
export const COPY_INTO = 'Copy into';
export const CURRENT = 'Current';
/** "It copies 2 initiatives, 7 activities and 15 line items." or the record alone. */
export function copiesText(level: RecordLevel, contents: string): string {
  return contents
    ? `It copies ${contents}.`
    : `It copies the ${level} only — there's nothing below it.`;
}
/** R13: what a copy leaves behind. */
export const COPY_RULE =
  'Copies get new IDs. Activities start as Created, without approvals, comments or history.';
export function copyLabel(level: RecordLevel): string {
  return `Copy ${level}`;
}
