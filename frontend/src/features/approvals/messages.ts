/**
 * The web texts of feature 008's approvals (plan, "User-visible messages"): every sentence the
 * approval parts show that is built from data. Fixed labels stay in the components; texts with a
 * count, a name or a time are made in this file and its neighbours (chain, history, decision
 * messages), so they read the same everywhere.
 */
import type { components } from '@/api/generated/schema';

type ApprovalPerson = components['schemas']['ApprovalPerson'];
type ActorRef = components['schemas']['ActorRef'];

/** "This activity no longer exists — someone may have deleted it." (API and web). */
export const ACTIVITY_GONE = 'This activity no longer exists — someone may have deleted it.';

/** Shown when an Approved activity cannot be recalled (API reason and web). */
export const APPROVED_NO_RECALL =
  "Approved activities can't be recalled. Changing its planned cost, dates or line-item amounts sends it back for approval.";

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

/** The text with its first letter in upper case. */
export function capitalise(text: string): string {
  return text.charAt(0).toUpperCase() + text.slice(1);
}
