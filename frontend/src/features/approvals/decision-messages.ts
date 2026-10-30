import type { ApiError } from '@/api/errors';
import type { components } from '@/api/generated/schema';
import { formatMoneyText } from '@/format/decimal';
import { formatDate, formatTime, localDate } from '@/format/intl';
import type { DateFormat } from '@/format/intl';
import { actorText, capitalise, names } from './messages';

type ActorRef = components['schemas']['ActorRef'];
type ApprovalRound = components['schemas']['ApprovalRound'];
type EscalationRunResult = components['schemas']['EscalationRunResult'];
type EscalationEvent = components['schemas']['EscalationEvent'];

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

/** One sentence per level due for escalation that no backup approver could take. */
export function notEscalatedText(events: EscalationEvent[]): string {
  return `Not escalated: ${events
    .map(
      (event) =>
        `'${event.activity.name}' (${event.activity.id}), level ${event.level}: no backup approver can take it.`,
    )
    .join(' ')}`;
}

export function declineBoxText(
  decline: { actor: ActorRef; onBehalfOf: ActorRef[]; at: string },
  dateFormat?: DateFormat,
): string {
  const day = formatDate(localDate(decline.at), dateFormat);
  return `Declined by ${actorText(decline.actor, decline.onBehalfOf)} on ${day}, ${formatTime(decline.at)}`;
}

/** The API's message with the time in the viewer's time zone ("on 12 Mar 2026 at 14:32." if not today). */
export function alreadyDecidedMessage(error: ApiError, dateFormat?: DateFormat): string {
  if (!error.conflict) {
    return error.message;
  }
  return `${error.message} ${when(error.conflict.changedAt, dateFormat)}.`;
}

export function submitConflictMessage(error: ApiError, dateFormat?: DateFormat): string {
  if (!error.conflict) {
    return error.message;
  }
  const { changedBy, changedAt } = error.conflict;
  return `Not submitted — ${changedBy} changed this activity ${when(changedAt, dateFormat)}. Check the changes and submit again.`;
}

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

export function overPlannedCostText({
  activity,
  overBy,
}: components['schemas']['ActivityOverage']): string {
  return `Line items of activity '${activity.name}' are ${formatMoneyText(overBy)} above its planned cost`;
}
