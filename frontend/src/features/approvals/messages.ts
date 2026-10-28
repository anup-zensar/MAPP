// Every approval sentence built from data (a count, a name or a time), made here so it reads the same
// everywhere; fixed labels stay in the components.
import type { components } from '@/api/generated/schema';

type ApprovalPerson = components['schemas']['ApprovalPerson'];
type ActorRef = components['schemas']['ActorRef'];

export const ACTIVITY_GONE = 'This activity no longer exists — someone may have deleted it.';

export const APPROVED_NO_RECALL =
  "Approved activities can't be recalled. Changing its planned cost or dates sends it back for approval.";

export function activitiesCount(count: number): string {
  return count === 1 ? '1 activity' : `${count} activities`;
}

export function names(people: { name: string }[]): string {
  return people.map((person) => person.name).join(', ');
}

export function approversText(approvers: ApprovalPerson[]): string {
  return `${approvers.length === 1 ? 'Approver' : 'Approvers'}: ${names(approvers)}`;
}

export function actorText(actor: ActorRef, onBehalfOf: ActorRef[]): string {
  return onBehalfOf.length > 0 ? `${actor.name} on behalf of ${names(onBehalfOf)}` : actor.name;
}

export function levelOf(level: number, levelCount: number): string {
  return `Level ${level} of ${levelCount}`;
}

export function daysText(days: number): string {
  return days === 1 ? '1 day' : `${days} days`;
}

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

// Decisions recorded before 017 have no capacity and read as an approver or on-behalf decision.
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

export function capitalise(text: string): string {
  return text.charAt(0).toUpperCase() + text.slice(1);
}
