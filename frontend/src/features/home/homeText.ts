/**
 * The words of Home (feature 009 plan, "User-visible messages"): every text built from data, so
 * the cards and their tests read the same. Fixed labels that appear once stay in the components.
 */
import { formatWhole } from '@/lib/decimal';
import { joinWithAnd, midSentence } from '@/lib/text';
import { formatWaitingTime, localDate } from '@/lib/format';
import type {
  ActivityLogEntry,
  HomeAttentionReason,
  HomeScope,
  Money,
  MyActivityRow,
} from './homeApi';

/** The one line under "Welcome back, <first name>." (FR-001). */
export const HOME_DESCRIPTION = 'Your plans, your activities and what needs you today.';

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
  everything: { label: 'All plans', tip: 'Every plan in NorthStar.' },
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

/** "84 % used" (card 4). */
export function percentUsedText(percent: number): string {
  return `${formatWhole(percent)} % used`;
}

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

/** "Show all 12" / "View all 12": the rest of a card's list (FR-012 – FR-014). */
export function showAllText(total: number): string {
  return `Show all ${formatWhole(total)}`;
}

export function viewAllText(total: number): string {
  return `View all ${formatWhole(total)}`;
}

/** How a reason's sentence shows amounts and dates: the signed-in person's formatters. */
export interface ReasonFormatters {
  /** "$1,200.00" for an amount. */
  money: (value: Money | null | undefined) => string;
  /** "12 Mar 2026" (or the person's format) for a calendar date "YYYY-MM-DD". */
  date: (date: string) => string;
}

/**
 * "3 fields needed · 1 item below needs attention": what is missing on a record and below it
 * (FR-011 e); only the parts that apply.
 */
export function missingFieldsText(missing: number, itemsBelow: number): string {
  const parts: string[] = [];
  if (missing > 0) {
    parts.push(missing === 1 ? '1 field needed' : `${formatWhole(missing)} fields needed`);
  }
  if (itemsBelow > 0) {
    parts.push(
      itemsBelow === 1
        ? '1 item below needs attention'
        : `${formatWhole(itemsBelow)} items below need attention`,
    );
  }
  return parts.join(' · ');
}

/** The sentence of one reason a record needs attention (FR-011). */
export function reasonText(reason: HomeAttentionReason, format: ReasonFormatters): string {
  switch (reason.kind) {
    case 'declined': {
      const by = reason.declinedBy ? ` by ${reason.declinedBy.name}` : '';
      const on = reason.declinedAt ? ` on ${format.date(localDate(reason.declinedAt))}` : '';
      const comment = reason.comment ? `: '${reason.comment}'` : '';
      return `Declined${by}${on}${comment}`;
    }
    case 'negativeBalance':
      return `Line items exceed the planned cost by ${format.money(reason.amount)}`;
    case 'overAllocated':
      return `Over-allocated by ${format.money(reason.amount)}`;
    case 'overBudget':
      return `Planned cost is ${format.money(reason.amount)} over budget`;
    case 'missingFields':
      return missingFieldsText(reason.missing.length, reason.itemsBelow);
  }
}

/** Where each reason is fixed: the link's words and the section of the record's page (FR-012). */
export const REASON_LINK: Record<HomeAttentionReason['kind'], { label: string; hash: string }> = {
  declined: { label: 'Open approval', hash: '#approval' },
  negativeBalance: { label: 'Open line items', hash: '#line-items' },
  overAllocated: { label: 'Open budget', hash: '#budget' },
  overBudget: { label: 'Open budget', hash: '#budget' },
  missingFields: { label: 'Show fields', hash: '#completeness' },
};

/** "Open approval for activity 'Launch film'": a reason link's name names its record. */
export function reasonLinkName(label: string, typeLabel: string, name: string): string {
  return `${label} for ${typeLabel} '${name}'`;
}

/** Shown when nothing the person owns has a problem (FR-021, US3 scenario 6). */
export const NOTHING_NEEDS_ATTENTION = 'Nothing needs your attention right now.';

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
 * An activity's dates in My activities (FR-014): "01 Mar 2026 – 31 May 2026", "Ongoing from
 * 01 Mar 2026", or "No dates yet"; `date` writes one date in the person's format.
 */
export function datesText(
  row: Pick<MyActivityRow, 'startDate' | 'endDate' | 'ongoing'>,
  date: (date: string) => string,
): string {
  const { startDate, endDate, ongoing } = row;
  if (!startDate && !endDate) {
    return 'No dates yet';
  }
  if (ongoing && startDate) {
    return `Ongoing from ${date(startDate)}`;
  }
  return `${startDate ? date(startDate) : '—'} – ${endDate ? date(endDate) : '—'}`;
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
  /** While the Activity log (feature 011) is still "Coming soon". */
  comingSoon: {
    title: 'Coming soon',
    text: 'Recent changes will show here once the Activity log is built.',
  },
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
  const words = name.trim().split(/\s+/).filter(Boolean);
  if (words.length === 0) {
    return '?';
  }
  const first = words[0][0];
  const last = words.length > 1 ? words[words.length - 1][0] : '';
  return `${first}${last}`.toUpperCase();
}
