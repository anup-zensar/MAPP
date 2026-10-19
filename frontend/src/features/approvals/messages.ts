/**
 * The web texts of feature 008's approvals (plan, "User-visible messages"): every sentence the
 * approval parts show that is built from data. Fixed labels stay in the components; texts with a
 * count, a name or a time are made here, so they read the same everywhere.
 */
import { ApiError } from '@/api/errors';
import type { components } from '@/api/schema';
import { formatDate, formatTime, localDate } from '@/lib/format';
import type { DateFormat } from '@/lib/format';

type ApprovalPerson = components['schemas']['ApprovalPerson'];
type ActorRef = components['schemas']['ActorRef'];
type ApprovalRound = components['schemas']['ApprovalRound'];
type ApprovalChainLevel = components['schemas']['ApprovalChainLevel'];
type ApprovalHistoryEntry = components['schemas']['ApprovalHistoryEntry'];

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
  };
  return texts[action];
}

/**
 * The state of one level of the chain: "Approved by Carlos Mendes on 12 Mar 2026" (with "on behalf
 * of …" when so), "Waiting", "Not reached yet", or why it was not decided.
 */
export function chainStateText(
  level: ApprovalChainLevel,
  outcome: ApprovalRound['outcome'],
  dateFormat?: DateFormat,
): string {
  switch (level.state) {
    case 'approved':
    case 'declined': {
      const verb = level.state === 'approved' ? 'Approved' : 'Declined';
      if (!level.decision) {
        return verb;
      }
      const { actor, onBehalfOf, at } = level.decision;
      return `${verb} by ${actorText(actor, onBehalfOf)} on ${formatDate(localDate(at), dateFormat)}`;
    }
    case 'waiting':
      return 'Waiting';
    case 'notReached':
      return 'Not reached yet';
    case 'notDecided':
      if (outcome === 'recalled') {
        return 'Not decided — submission recalled';
      }
      if (outcome === 'restarted') {
        return 'Not decided — approval restarted';
      }
      return 'Not decided — activity deleted';
  }
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
