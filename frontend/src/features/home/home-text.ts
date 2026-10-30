import { formatWhole } from '@/format/decimal';
import { joinWithAnd, midSentence } from '@/format/text';
import { formatWaitingTime } from '@/format/intl';
import type { ActivityLogEntry, HomeScope, Money, MyActivityRow } from './api';

/** "Welcome back, Priya.": Home is the only page whose title greets (design system §2.3). */
export function greetingText(firstName: string): string {
  return `Welcome back, ${firstName}.`;
}

export const NO_ROLE_TITLE = 'No access to plans yet';

/** 014 text 36; without any administrator the brackets go. */
export function noRoleText(administrators: string[]): string {
  const who = administrators.length > 0 ? ` (${joinWithAnd(administrators)})` : '';
  return `You don't have access to any plans yet. Ask an administrator${who} to give you a role.`;
}

export const SECTION_FAILED = "This section couldn't load";

export const CHOICE_NOT_SAVED = "Your choice couldn't be saved.";

export const PLANS_CARD: Record<HomeScope, { label: string; tip: string }> = {
  myTeams: {
    label: "My teams' plans",
    tip: 'Plans that have at least one of your teams.',
  },
  everything: { label: 'All plans', tip: 'Every plan in SP&A One.' },
};

export function planCountsText(published: number, draft: number): string {
  return `${formatWhole(published)} published · ${formatWhole(draft)} draft`;
}

export function activityCountsText(declined: number, submitted: number): string {
  return `${formatWhole(declined)} declined · ${formatWhole(submitted)} submitted`;
}

export function ownedYearsText(yearFrom: number | null, yearTo: number | null): string {
  if (yearFrom === null) {
    return '';
  }
  return yearTo !== null && yearTo > yearFrom ? `in ${yearFrom}–${yearTo}` : `in ${yearFrom}`;
}

export function oldestWaitingText(since: string | null, now: Date = new Date()): string {
  return since ? `oldest waiting ${formatWaitingTime(since, now)}` : 'Nothing waiting';
}

export function ofPlanBudgetText(budget: string): string {
  return `of ${budget} plan budget`;
}

export function percentText(percent: number): string {
  return `${formatWhole(percent)}%`;
}

export function percentUsedText(percent: number): string {
  return `${percentText(percent)} used`;
}

/** Whether an amount is zero ("0.00"): nothing is planned, which Home shows as "—" (§4.4). */
export function isZero(money: Money | null | undefined): boolean {
  return !money || /^-?0*(\.0*)?$/.test(money.amount);
}

export const NOTHING_PLANNED = 'Nothing planned yet';

export function plansWithoutBudgetText(count: number): string {
  return count === 1
    ? 'Includes 1 plan with no budget'
    : `Includes ${formatWhole(count)} plans with no budget`;
}

export const SCOPE_HINT =
  'Your own activities, what needs your attention and your approvals always cover all plans.';

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

export function approvedText(approved: number, activities: number): string {
  return `${formatWhole(approved)} of ${formatWhole(activities)} approved`;
}

export function approvedPercent(approved: number, activities: number): number {
  return activities === 0 ? 0 : Math.round((approved * 100) / activities);
}

export function viewAllPlansText(total: number): string {
  return `View all ${formatWhole(total)} plans`;
}

export const WAITING_SUBTITLE = 'Activities waiting at a level you approve';

export const ATTENTION_SUBTITLE = 'Things you own that have a problem';

export function viewAllText(total: number): string {
  return `View all (${formatWhole(total)})`;
}

export const SHOW_FEWER = 'Show fewer';

export const NOTHING_WAITING = 'Nothing is waiting for your approval right now.';

export function levelText(level: number, levels: number): string {
  return `Level ${formatWhole(level)} of ${formatWhole(levels)}`;
}

export function waitingText(since: string, now: Date = new Date()): string {
  return `waiting ${formatWaitingTime(since, now)}`;
}

export const NO_ACTIVITIES = "You don't own any activities yet.";

/** As every date range reads (D-216), plus "No dates yet", or "—" for one missing date. */
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

export function lineItemsNeedAttentionText(count: number): string {
  return count === 1
    ? '1 line item needs attention'
    : `${formatWhole(count)} line items need attention`;
}

export const RECENT_CHANGES = {
  subtitle: {
    myTeams: "Latest changes in your teams' plans",
    everything: 'Latest changes in all plans',
  } satisfies Record<HomeScope, string>,
  empty: 'No changes yet in these plans.',
};

/** "planned cost", "name and GL code", "a, b and c"; over three: "a, b, c and 2 more fields". */
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
  lead: string;
  name: string;
  id: string;
}

export function recentSentence(entry: ActivityLogEntry): RecentSentence {
  const { actor, recordTypeLabel, recordName, recordId } = entry;
  const what =
    entry.action === 'changed' && entry.changes.length > 0
      ? `changed ${changedFieldsText(entry.changes.map((change) => change.label))} of`
      : entry.verb;
  return { lead: `${actor.name} ${what} ${recordTypeLabel}`, name: recordName, id: recordId };
}

export function initialsOf(name: string): string {
  const [first, ...others] = name.trim().split(/\s+/).filter(Boolean);
  if (first === undefined) {
    return '?';
  }
  const last = others.at(-1) ?? '';
  return `${first.charAt(0)}${last.charAt(0)}`.toUpperCase();
}
