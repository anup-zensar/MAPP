import type { components } from '@/api/generated/schema';
import { capitalise, daysText, decisionActorText } from './messages';

type ApprovalRound = components['schemas']['ApprovalRound'];
type ApprovalHistoryEntry = components['schemas']['ApprovalHistoryEntry'];
type ApprovalRuleFacts = components['schemas']['ApprovalRuleFacts'];
type Money = components['schemas']['Money'];

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
 * behalf of Daniel Okoro"; MAPP for reminders, escalations and rules.
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
