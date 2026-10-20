/**
 * The web texts of feature 008's approvals (plan, "User-visible messages"): every sentence the
 * approval parts show that is built from data. Fixed labels stay in the components; texts with a
 * count, a name or a time are made here, so they read the same everywhere.
 */
import { ApiError } from '@/api/errors';
import type { components } from '@/api/schema';
import { formatDate, formatTime, localDate } from '@/lib/format';
import type { DateFormat } from '@/lib/format';
import type { StatusTone } from '@/components';

type ApprovalPerson = components['schemas']['ApprovalPerson'];
type ActorRef = components['schemas']['ActorRef'];
type ApprovalRound = components['schemas']['ApprovalRound'];
type ApprovalChainLevel = components['schemas']['ApprovalChainLevel'];
type ApprovalHistoryEntry = components['schemas']['ApprovalHistoryEntry'];
type ApprovalWorkflowLevel = components['schemas']['ApprovalWorkflowLevel'];
type ApprovalExcludedPerson = components['schemas']['ApprovalExcludedPerson'];
type ApprovalRuleFacts = components['schemas']['ApprovalRuleFacts'];
type EscalationRunResult = components['schemas']['EscalationRunResult'];
type EscalationEvent = components['schemas']['EscalationEvent'];
type Money = components['schemas']['Money'];

/** "This activity no longer exists — someone may have deleted it." (API and web). */
export const ACTIVITY_GONE = 'This activity no longer exists — someone may have deleted it.';

/** Shown when an Approved activity cannot be recalled (API reason and web). */
export const APPROVED_NO_RECALL =
  "Approved activities can't be recalled. Changing its planned cost, dates or line-item amounts sends it back for approval.";

/** Any failure as an ApiError, so its plain-language message can be shown. */
export function asApiError(error: unknown): ApiError {
  return error instanceof ApiError ? error : new ApiError('other');
}

/** The session ended: the sign-in page says so, so nothing else needs to. */
export function isSessionLost(error: ApiError): boolean {
  return error.kind === 'signInRequired' || error.kind === 'sessionExpired';
}

/** "1 activity" / "3 activities". */
export function activitiesCount(count: number): string {
  return count === 1 ? '1 activity' : `${count} activities`;
}

/** Names joined with ", " ("Carlos Mendes, Daniel Okoro"). */
export function names(people: { name: string }[]): string {
  return people.map((person) => person.name).join(', ');
}

/** "Approver: Carlos Mendes" / "Approvers: Carlos Mendes, Daniel Okoro". */
export function approversText(approvers: ApprovalPerson[]): string {
  return `${approvers.length === 1 ? 'Approver' : 'Approvers'}: ${names(approvers)}`;
}

/** "Alex Morgan" or "Alex Morgan on behalf of Daniel Okoro". */
export function actorText(actor: ActorRef, onBehalfOf: ActorRef[]): string {
  return onBehalfOf.length > 0 ? `${actor.name} on behalf of ${names(onBehalfOf)}` : actor.name;
}

/** "Level 1 of 2". */
export function levelOf(level: number, levelCount: number): string {
  return `Level ${level} of ${levelCount}`;
}

/** What a round came to: "Waiting at level 1 of 2", "Approved", "Recalled", … */
export function roundOutcomeText(
  round: Pick<ApprovalRound, 'outcome' | 'currentLevel' | 'levelCount'>,
) {
  switch (round.outcome) {
    case 'open':
      return `Waiting at level ${round.currentLevel} of ${round.levelCount}`;
    case 'approved':
      return 'Approved';
    case 'declined':
      return 'Declined';
    case 'recalled':
      return 'Recalled';
    case 'restarted':
      return 'Approval restarted';
    case 'cancelled':
      return 'Cancelled — activity deleted';
  }
}

/** The label of a history entry's action: "Submitted", "Re-submitted", "Approved", … */
export function historyActionText(action: ApprovalHistoryEntry['action']): string {
  const texts: Record<ApprovalHistoryEntry['action'], string> = {
    submitted: 'Submitted',
    resubmitted: 'Re-submitted',
    approved: 'Approved',
    declined: 'Declined',
    recalled: 'Recalled',
    restarted: 'Approval restarted',
    cancelled: 'Cancelled — activity deleted',
    autoApproved: 'Approved automatically',
    skipped: 'Not needed',
    reminded: 'Reminder sent',
    escalated: 'Escalated',
  };
  return texts[action];
}

/** The title of a history entry: its action, and for an escalation whom it went to (017 FR-034). */
export function historyEntryTitle(
  entry: Pick<ApprovalHistoryEntry, 'action' | 'escalatedTo'>,
): string {
  if (entry.action === 'escalated' && entry.escalatedTo) {
    return `Escalated to ${entry.escalatedTo.name}`;
  }
  return historyActionText(entry.action);
}

/**
 * Who acted in a history entry, in the capacity they acted (017 FR-019, FR-034): "Carlos Mendes",
 * "Carlos Mendes as delegate of Daniel Okoro", "Alex Morgan as backup approver", "Alex Morgan on
 * behalf of Daniel Okoro"; NorthStar for reminders, escalations and rules.
 */
export function historyActorText(entry: ApprovalHistoryEntry): string {
  return decisionActorText(entry);
}

/** What else an entry of 017 says: the rule that applied, or how long the level waited. */
export function historyEntryNote(entry: ApprovalHistoryEntry): string {
  if ((entry.action === 'autoApproved' || entry.action === 'skipped') && entry.rule) {
    return entry.action === 'skipped' ? `Only when ${entry.rule}` : capitalise(entry.rule);
  }
  if ((entry.action === 'reminded' || entry.action === 'escalated') && entry.waitedDays) {
    return `waited ${daysText(entry.waitedDays)}`;
  }
  return '';
}

/**
 * "Rules applied to: planned cost $50,000.00 · Event · Lubricants, Mobility" — the facts the rules
 * of a round used at submission (017 FR-010); "" for rounds submitted before the rules.
 */
export function rulesAppliedText(
  facts: ApprovalRuleFacts | null | undefined,
  money: (value: Money | null) => string,
): string {
  if (!facts) {
    return '';
  }
  const parts = [
    facts.plannedCost ? `planned cost ${money(facts.plannedCost)}` : 'no planned cost',
    facts.activityType?.name ?? '',
    facts.businessUnits.map((unit) => unit.label).join(', '),
  ].filter(Boolean);
  return `Rules applied to: ${parts.join(' · ')}`;
}

/** "1 day" / "3 days" (017: day counts read the same everywhere). */
export function daysText(days: number): string {
  return days === 1 ? '1 day' : `${days} days`;
}

/**
 * The reminders and escalation of a workflow: "Reminder after 3 days · escalation after 5 days",
 * one of the two, or "No reminders or escalation" (017 FR-007, FR-009).
 */
export function timingsText(
  reminderDays: number | null | undefined,
  escalationDays: number | null | undefined,
): string {
  const parts = [
    reminderDays ? `reminder after ${daysText(reminderDays)}` : '',
    escalationDays ? `escalation after ${daysText(escalationDays)}` : '',
  ].filter(Boolean);
  return parts.length > 0 ? capitalise(parts.join(' · ')) : 'No reminders or escalation';
}

/**
 * The rules of a workflow level in words (017 FR-009), one line each: "Line manager" or "Role:
 * Approver", "All must approve", "Only needed when planned cost is at least $100,000.00",
 * "Approved automatically when planned cost is below $10,000.00", "Backup approver: Alex Morgan".
 * A level stored before the rules has none.
 */
export function levelRuleLines(level: ApprovalWorkflowLevel): string[] {
  const lines: string[] = [];
  if (level.approverType === 'manager') {
    lines.push('Line manager');
  } else if (level.approverType === 'role') {
    lines.push(`Role: ${level.role?.name ?? '—'}`);
  }
  if (level.mode === 'all' && level.approverType !== 'manager') {
    lines.push('All must approve');
  }
  if (level.appliesWhen) {
    lines.push(`Only needed when ${level.appliesWhen.text}`);
  }
  if (level.autoApproveWhen) {
    lines.push(`Approved automatically when ${level.autoApproveWhen.text}`);
  }
  if (level.backupApprover) {
    lines.push(`Backup approver: ${personName(level.backupApprover)}`);
  }
  return lines;
}

/**
 * Where a round level's approvers came from, as the chain writes it: "Line manager of Priya
 * Raman", "Role: Approver"; "" for named people (the API says "line manager of …", "role …").
 */
export function sourceText(source: string | undefined): string {
  if (!source) {
    return '';
  }
  if (source.startsWith('role ')) {
    return `Role: ${source.slice('role '.length)}`;
  }
  return capitalise(source);
}

/** Why a person was left out of a level (017 FR-011, 014 FR-028). */
const LEFT_OUT: Record<ApprovalExcludedPerson['reason'], { now: string; then: string }> = {
  owner: { now: 'owns this activity', then: 'owns this activity' },
  submitter: { now: 'is submitting it', then: 'submitted it' },
  inactive: { now: 'is inactive', then: 'inactive' },
  cannotApprove: {
    now: "can't approve — no approving role",
    then: "can't approve — no approving role",
  },
};

/** At submission: "Priya Raman is left out: owns this activity." */
export function leftOutText(excluded: ApprovalExcludedPerson): string {
  return `${excluded.person.name} is left out: ${LEFT_OUT[excluded.reason].now}.`;
}

/** In the chain: "Priya Raman was left out: owns this activity." */
export function wasLeftOutText(excluded: ApprovalExcludedPerson): string {
  return `${excluded.person.name} was left out: ${LEFT_OUT[excluded.reason].then}.`;
}

/** "1 of 3 approved" on a level where every approver must approve (017 FR-015). */
export function progressText(approvedCount: number, requiredCount: number): string {
  return `${approvedCount} of ${requiredCount} approved`;
}

/** "Escalated to Alex Morgan on 14 Mar 2026" (017 FR-027, FR-033). */
export function escalatedText(name: string, at: string, dateFormat?: DateFormat): string {
  return `Escalated to ${name} on ${formatDate(localDate(at), dateFormat)}`;
}

/** "Reminder sent on 13 Mar 2026" (017 FR-026, FR-033). */
export function reminderSentText(at: string, dateFormat?: DateFormat): string {
  return `Reminder sent on ${formatDate(localDate(at), dateFormat)}`;
}

/** What the person deciding acts as (017 FR-019), for the decision dialog; "" for an approver. */
export function capacitySentence(
  capacity: string | null | undefined,
  asDelegateOf: { name: string }[],
  level: number,
): string {
  if (capacity === 'delegate' && asDelegateOf.length > 0) {
    return `You approve as delegate of ${names(asDelegateOf)}.`;
  }
  if (capacity === 'backup') {
    return `You approve as backup approver of level ${level}.`;
  }
  return '';
}

/**
 * The result of "Run reminders and escalations now" (017 FR-028): "2 reminders sent, 1
 * escalation", "1 reminder sent, no escalations", "No reminders, no escalations".
 */
export function escalationRunText(result: Pick<EscalationRunResult, 'reminded' | 'escalated'>) {
  const reminders =
    result.reminded === 0
      ? 'no reminders'
      : result.reminded === 1
        ? '1 reminder sent'
        : `${result.reminded} reminders sent`;
  const escalations =
    result.escalated === 0
      ? 'no escalations'
      : result.escalated === 1
        ? '1 escalation'
        : `${result.escalated} escalations`;
  return capitalise(`${reminders}, ${escalations}`);
}

/**
 * "Not escalated: 'Trade magazine ads' (A-1004), level 1: no backup approver can take it." — the
 * levels due for escalation that nobody could take, one sentence each joined by a space.
 */
export function notEscalatedText(events: EscalationEvent[]): string {
  return `Not escalated: ${events
    .map(
      (event) =>
        `'${event.activity.name}' (${event.activity.id}), level ${event.level}: no backup approver can take it.`,
    )
    .join(' ')}`;
}

/** "Alex Morgan" or "Alex Morgan (inactive)". */
function personName(person: ApprovalPerson): string {
  return person.active ? person.name : `${person.name} (inactive)`;
}

/** The text with its first letter in upper case. */
function capitalise(text: string): string {
  return text.charAt(0).toUpperCase() + text.slice(1);
}

/** The state of one level of the chain as a status word, its family and what follows the word. */
export interface ChainState {
  /** The status word, shown in a chip of its one family (design system §4.1): "Approved". */
  word: string;
  tone: StatusTone;
  /** What follows the word: "by Carlos Mendes on 12 Mar 2026", "— submission recalled"; "" for none. */
  detail: string;
}

/**
 * The state of one level of the chain: "Approved" by Carlos Mendes on 12 Mar 2026 (with "on behalf
 * of …" when so), "Waiting", "Not reached yet", "Approved automatically" or "Not needed" with the
 * rule, or why it was not decided. Each word has one colour everywhere (design system §4.1).
 */
export function chainState(
  level: ApprovalChainLevel,
  outcome: ApprovalRound['outcome'],
  dateFormat?: DateFormat,
): ChainState {
  switch (level.state) {
    case 'approved':
    case 'declined': {
      const approved = level.state === 'approved';
      const word = approved ? 'Approved' : 'Declined';
      const tone: StatusTone = approved ? 'positive' : 'problem';
      if (!level.decision) {
        return { word, tone, detail: '' };
      }
      const { at } = level.decision;
      const by = decisionActorText(level.decision);
      return { word, tone, detail: `by ${by} on ${formatDate(localDate(at), dateFormat)}` };
    }
    case 'waiting':
      return { word: 'Waiting', tone: 'review', detail: '' };
    case 'notReached':
      return { word: 'Not reached yet', tone: 'neutral', detail: '' };
    case 'autoApproved':
      return { word: 'Approved automatically', tone: 'positive', detail: ruleDetail(level) };
    case 'skipped':
      return { word: 'Not needed', tone: 'neutral', detail: ruleDetail(level) };
    case 'notDecided': {
      const why =
        outcome === 'recalled'
          ? 'submission recalled'
          : outcome === 'restarted'
            ? 'approval restarted'
            : 'activity deleted';
      return { word: 'Not decided', tone: 'neutral', detail: `— ${why}` };
    }
  }
}

/** The state of one level as one sentence: "Approved by Carlos Mendes on 12 Mar 2026". */
export function chainStateText(
  level: ApprovalChainLevel,
  outcome: ApprovalRound['outcome'],
  dateFormat?: DateFormat,
): string {
  const { word, detail } = chainState(level, outcome, dateFormat);
  return detail ? `${word} ${detail}` : word;
}

/** "— only when planned cost is at least $100,000.00"; "" without a rule text. */
function ruleDetail(level: Pick<ApprovalChainLevel, 'ruleText'>): string {
  return level.ruleText ? `— ${level.ruleText}` : '';
}

/**
 * Who made a decision, in the capacity it was made (017 FR-019): "Carlos Mendes", "Carlos Mendes
 * as delegate of Daniel Okoro", "Alex Morgan as backup approver", "Alex Morgan on behalf of Daniel
 * Okoro". Decisions recorded before 017 have no capacity and read as before.
 */
export function decisionActorText(decision: {
  actor: ActorRef;
  onBehalfOf: ActorRef[];
  capacity?: string;
  asDelegateOf?: ActorRef | null;
}): string {
  const { actor, onBehalfOf, capacity, asDelegateOf } = decision;
  if (capacity === 'delegate' && asDelegateOf) {
    return `${actor.name} as delegate of ${asDelegateOf.name}`;
  }
  if (capacity === 'backup') {
    return `${actor.name} as backup approver`;
  }
  return actorText(actor, onBehalfOf);
}

/** "Declined by Alex Morgan on behalf of Daniel Okoro on 12 Mar 2026, 14:32". */
export function declineBoxText(
  decline: { actor: ActorRef; onBehalfOf: ActorRef[]; at: string },
  dateFormat?: DateFormat,
): string {
  const day = formatDate(localDate(decline.at), dateFormat);
  return `Declined by ${actorText(decline.actor, decline.onBehalfOf)} on ${day}, ${formatTime(decline.at)}`;
}

/**
 * "Level 1 was already approved by Carlos Mendes at 14:32." — the API's message with the time of
 * the decision in the viewer's own time zone ("on 12 Mar 2026 at 14:32." when not today).
 */
export function alreadyDecidedMessage(error: ApiError, dateFormat?: DateFormat): string {
  if (!error.conflict) {
    return error.message;
  }
  return `${error.message} ${when(error.conflict.changedAt, dateFormat)}.`;
}

/**
 * "Not submitted — Daniel Okoro changed this activity at 14:32. Check the changes and submit
 * again." ("on 12 Mar 2026 at 14:32" when not today).
 */
export function submitConflictMessage(error: ApiError, dateFormat?: DateFormat): string {
  if (!error.conflict) {
    return error.message;
  }
  const { changedBy, changedAt } = error.conflict;
  return `Not submitted — ${changedBy} changed this activity ${when(changedAt, dateFormat)}. Check the changes and submit again.`;
}

/** The message after a level is approved, from the activity's approval as the API answered it. */
export function approvedToast(
  name: string,
  decidedLevel: number,
  round: Pick<ApprovalRound, 'outcome' | 'currentLevel' | 'levels'> | undefined,
): string {
  if (!round || round.outcome !== 'open') {
    return `Activity '${name}' approved`;
  }
  const next = round.levels.find((level) => level.number === round.currentLevel);
  const approvers = next ? names(next.approvers) : '';
  return `Level ${decidedLevel} approved — '${name}' now waits for ${approvers} (level ${round.currentLevel}).`;
}

/** "at 14:32" today, otherwise "on 12 Mar 2026 at 14:32", in the viewer's time zone. */
function when(timestamp: string, dateFormat?: DateFormat): string {
  const time = `at ${formatTime(timestamp)}`;
  const day = localDate(timestamp);
  return day === localDate(new Date()) ? time : `on ${formatDate(day, dateFormat)} ${time}`;
}
