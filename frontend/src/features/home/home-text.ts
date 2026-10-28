/**
 * The words of Home (feature 009 plan, "User-visible messages"): every text built from data, so
 * the cards and their tests read the same. Fixed labels that appear once stay in the components.
 */
import { formatWhole } from '@/format/decimal';
import { joinWithAnd, midSentence } from '@/format/text';
import { formatWaitingTime } from '@/format/intl';
import type { ActivityLogEntry, HomeScope, Money, MyActivityRow } from './api';

/**
 * Home's title (FR-001; design system §2.3: Home is the only page whose title greets), one
 * phrase without a grey highlight: "Welcome back, Priya."
 */
export function greetingText(firstName: string): string {
  return `Welcome back, ${firstName}.`;
}

/** The no-role state of Home (014 edge case "No role"): its heading. */
export const NO_ROLE_TITLE = 'No access to plans yet';

/**
 * "You don't have access to any plans yet. Ask an administrator (Alex Morgan) to give you a
 * role." (014 text 36); without any administrator the brackets go.
 */
export function noRoleText(administrators: string[]): string {
  const who = administrators.length > 0 ? ` (${joinWithAnd(administrators)})` : '';
  return `You don't have access to any plans yet. Ask an administrator${who} to give you a role.`;
}

/** Shown inside a card whose data could not be loaded (FR-022). */
export const SECTION_FAILED = "This section couldn't load";

/** A failed save of the person's choice (001's preference pattern). */
export const CHOICE_NOT_SAVED = "Your choice couldn't be saved.";

/** Card 1's label and tip by scope (FR-005). */
export const PLANS_CARD: Record<HomeScope, { label: string; tip: string }> = {
  myTeams: {
    label: "My teams' plans",
    tip: 'Plans that have at least one of your teams.',
  },
  everything: { label: 'All plans', tip: 'Every plan in SP&A One.' },
};

/** "1 published · 0 draft" (card 1). */
export function planCountsText(published: number, draft: number): string {
  return `${formatWhole(published)} published · ${formatWhole(draft)} draft`;
}

/** "2 declined · 1 submitted" (card 2). */
export function activityCountsText(declined: number, submitted: number): string {
  return `${formatWhole(declined)} declined · ${formatWhole(submitted)} submitted`;
}

/** "oldest waiting 3 days", or "Nothing waiting" when nothing waits (card 3). */
export function oldestWaitingText(since: string | null, now: Date = new Date()): string {
  return since ? `oldest waiting ${formatWaitingTime(since, now)}` : 'Nothing waiting';
}

/** "of $1.2M plan budget" (card 4). */
export function ofPlanBudgetText(budget: string): string {
  return `of ${budget} plan budget`;
}

/** "84%": a whole percentage, without a space (design system §4.4). */
export function percentText(percent: number): string {
  return `${formatWhole(percent)}%`;
}

/** "84% used" (card 4). */
export function percentUsedText(percent: number): string {
  return `${percentText(percent)} used`;
}

/** Whether an amount is zero ("0.00"): nothing is planned, which Home shows as "—" (§4.4). */
export function isZero(money: Money | null | undefined): boolean {
  return !money || /^-?0*(\.0*)?$/.test(money.amount);
}

/** Shown after "—" when no planned cost exists yet (design system §4.4, functional review X-08). */
export const NOTHING_PLANNED = 'Nothing planned yet';

/** "Includes 1 plan with no budget" / "Includes 2 plans with no budget" (card 4, FR-006). */
export function plansWithoutBudgetText(count: number): string {
  return count === 1
    ? 'Includes 1 plan with no budget'
    : `Includes ${formatWhole(count)} plans with no budget`;
}

/** The hint beside the My teams / Everything switch (FR-003). */
export const SCOPE_HINT =
  'Your own activities, what needs your attention and your approvals always cover all plans.';

/** Plans at a glance: its tip, and its subtitle by scope (FR-008). */
export const PLANS_AT_A_GLANCE = {
  tip: 'Every plan your teams work on, the teams on it, and live status rolled up from its activities.',
  subtitle: {
    myTeams: 'Live roll-up across your teams',
    everything: 'Live roll-up across all plans',
  } satisfies Record<HomeScope, string>,
  empty: {
    myTeams: 'None of your teams has a plan yet.',
    everything: 'No plans yet — create the first one on the Plans page.',
  } satisfies Record<HomeScope, string>,
};

/** "3 of 7 approved" (a plan's approval progress, FR-009). */
export function approvedText(approved: number, activities: number): string {
  return `${formatWhole(approved)} of ${formatWhole(activities)} approved`;
}

/** The share of approved activities for the bar, as a whole percentage. */
export function approvedPercent(approved: number, activities: number): number {
  return activities === 0 ? 0 : Math.round((approved * 100) / activities);
}

/** "View all 12 plans" (FR-010). */
export function viewAllPlansText(total: number): string {
  return `View all ${formatWhole(total)} plans`;
}

/** Waiting for my approval: what the card lists (§2.4 E: both list cards have a subtitle). */
export const WAITING_SUBTITLE = 'Activities waiting at a level you approve';

/** Needs your attention: what the card lists. */
export const ATTENTION_SUBTITLE = 'Things you own that have a problem';

/** "View all (12)": the rest of a card's list, on the card or on its page (§2.4 E). */
export function viewAllText(total: number): string {
  return `View all (${formatWhole(total)})`;
}

/** Folds a list opened with View all again. */
export const SHOW_FEWER = 'Show fewer';

/** Shown when nothing waits for the person's decision (FR-021, US3 scenario 6). */
export const NOTHING_WAITING = 'Nothing is waiting for your approval right now.';

/** "Level 1 of 2" (FR-013). */
export function levelText(level: number, levels: number): string {
  return `Level ${formatWhole(level)} of ${formatWhole(levels)}`;
}

/** "waiting 3 days" (FR-013). */
export function waitingText(since: string, now: Date = new Date()): string {
  return `waiting ${formatWaitingTime(since, now)}`;
}

/** Shown when the person owns no activity (FR-021). */
export const NO_ACTIVITIES = "You don't own any activities yet.";

/**
 * An activity's dates in My activities (FR-014), as every date range reads (D-216): "01 Mar –
 * 31 May 2026" (`dateRange` of useFormatters(), in the person's date format), "Ongoing from 01 Mar
 * 2026", "No dates yet", or one date with "—" for the missing one.
 */
export function datesText(
  row: Pick<MyActivityRow, 'startDate' | 'endDate' | 'ongoing'>,
  format: { date: (date: string) => string; dateRange: (start: string, end: string) => string },
): string {
  const { startDate, endDate, ongoing } = row;
  if (!startDate && !endDate) {
    return 'No dates yet';
  }
  if (ongoing && startDate) {
    return `Ongoing from ${format.date(startDate)}`;
  }
  if (startDate && endDate) {
    return format.dateRange(startDate, endDate);
  }
  return `${startDate ? format.date(startDate) : '—'} – ${endDate ? format.date(endDate) : '—'}`;
}

/** "1 line item needs attention" / "3 line items need attention" (FR-014). */
export function lineItemsNeedAttentionText(count: number): string {
  return count === 1
    ? '1 line item needs attention'
    : `${formatWhole(count)} line items need attention`;
}

/** Recent changes: its subtitle by scope and its empty sentence (FR-015, FR-021). */
export const RECENT_CHANGES = {
  subtitle: {
    myTeams: "Latest changes in your teams' plans",
    everything: 'Latest changes in all plans',
  } satisfies Record<HomeScope, string>,
  empty: 'No changes yet in these plans.',
};

/**
 * The changed fields in a sentence (FR-015): "planned cost", "name and GL code", "a, b and c";
 * more than three: "a, b, c and 2 more fields".
 */
export function changedFieldsText(labels: string[]): string {
  const fields = labels.map(midSentence);
  if (fields.length <= 3) {
    return joinWithAnd(fields);
  }
  const more = fields.length - 3;
  return `${fields.slice(0, 3).join(', ')} and ${more} more ${more === 1 ? 'field' : 'fields'}`;
}

/** A sentence of Recent changes, cut where the record's name goes (it may be a link). */
export interface RecentSentence {
  /** "Priya Raman changed planned cost of activity" */
  lead: string;
  /** The record's name at the time: "Launch film". */
  name: string;
  /** The record's ID: "A-1042". */
  id: string;
}

/**
 * "<person> changed <fields> of <type> '<name>' (<id>)" for a change of fields, otherwise
 * "<person> <verb> <type> '<name>' (<id>)" (FR-015).
 */
export function recentSentence(entry: ActivityLogEntry): RecentSentence {
  const { actor, recordTypeLabel, recordName, recordId } = entry;
  const what =
    entry.action === 'changed' && entry.changes.length > 0
      ? `changed ${changedFieldsText(entry.changes.map((change) => change.label))} of`
      : entry.verb;
  return { lead: `${actor.name} ${what} ${recordTypeLabel}`, name: recordName, id: recordId };
}

/** "PR" for Priya Raman: the first letters of the first and the last name. */
export function initialsOf(name: string): string {
  const [first, ...others] = name.trim().split(/\s+/).filter(Boolean);
  if (first === undefined) {
    return '?';
  }
  const last = others.at(-1) ?? '';
  return `${first.charAt(0)}${last.charAt(0)}`.toUpperCase();
}
