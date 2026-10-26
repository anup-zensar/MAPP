import type { components } from '@/api/generated/schema';
import { formatDate, localDate } from '@/format/intl';
import type { DateFormat } from '@/format/intl';
import type { StatusTone } from '@/ui/StatusChip/StatusChip';
import { capitalise, decisionActorText } from './messages';

type ApprovalPerson = components['schemas']['ApprovalPerson'];
type ApprovalRound = components['schemas']['ApprovalRound'];
type ApprovalChainLevel = components['schemas']['ApprovalChainLevel'];
type ApprovalWorkflowLevel = components['schemas']['ApprovalWorkflowLevel'];
type ApprovalExcludedPerson = components['schemas']['ApprovalExcludedPerson'];

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

/** The state of one level of the chain as a status word, its family and what follows the word. */
export interface ChainState {
  /** The status word, shown in a chip of its one family (design system §4.1): "Approved". */
  word: string;
  tone: StatusTone;
  /** What follows the word: "by Carlos Mendes on 12 Mar 2026", "— submission recalled"; "" for none. */
  detail: string;
}

/**
 * The rules of a workflow level in words (017 FR-009), one line each: "The owner's line manager"
 * or "Role: Approver", "All must approve", "Only needed when planned cost is at least $100,000.00",
 * "Approved automatically when planned cost is below $10,000.00", "Backup approver: Alex Morgan".
 * A level stored before the rules has none.
 */
export function levelRuleLines(level: ApprovalWorkflowLevel): string[] {
  const lines: string[] = [];
  if (level.approverType === 'manager') {
    lines.push("The owner's line manager");
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

/** "Alex Morgan" or "Alex Morgan (inactive)". */
function personName(person: ApprovalPerson): string {
  return person.active ? person.name : `${person.name} (inactive)`;
}

/** "— only when planned cost is at least $100,000.00"; "" without a rule text. */
function ruleDetail(level: Pick<ApprovalChainLevel, 'ruleText'>): string {
  return level.ruleText ? `— ${level.ruleText}` : '';
}
