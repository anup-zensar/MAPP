import type { components } from '@/api/generated/schema';
import { capitalise, daysText, decisionActorText } from './messages';

type ApprovalRound = components['schemas']['ApprovalRound'];
type ApprovalHistoryEntry = components['schemas']['ApprovalHistoryEntry'];
type ApprovalRuleFacts = components['schemas']['ApprovalRuleFacts'];
type Money = components['schemas']['Money'];

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
    case 'superseded':
      return 'Replaced by a new submission';
  }
}

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
    superseded: 'Replaced by a new submission',
  };
  return texts[action];
}

export function historyEntryTitle(
  entry: Pick<ApprovalHistoryEntry, 'action' | 'escalatedTo'>,
): string {
  if (entry.action === 'escalated' && entry.escalatedTo) {
    return `Escalated to ${entry.escalatedTo.name}`;
  }
  return historyActionText(entry.action);
}

export function historyActorText(entry: ApprovalHistoryEntry): string {
  return decisionActorText(entry);
}

export function historyEntryNote(entry: ApprovalHistoryEntry): string {
  if (entry.selfApproval) {
    return 'Self-approval';
  }
  if ((entry.action === 'autoApproved' || entry.action === 'skipped') && entry.rule) {
    return capitalise(entry.rule);
  }
  if ((entry.action === 'reminded' || entry.action === 'escalated') && entry.waitedDays) {
    return `waited ${daysText(entry.waitedDays)}`;
  }
  return '';
}

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
