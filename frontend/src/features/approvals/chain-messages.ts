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

const LEFT_OUT: Record<ApprovalExcludedPerson['reason'], { now: string; then: string }> = {
  owner: { now: 'owns this activity', then: 'owns this activity' },
  submitter: { now: 'is submitting it', then: 'submitted it' },
  inactive: { now: 'is inactive', then: 'inactive' },
  cannotApprove: {
    now: "can't approve — no approving role",
    then: "can't approve — no approving role",
  },
};

export interface ChainState {
  word: string;
  tone: StatusTone;
  detail: string;
}

function sourceLine(level: ApprovalWorkflowLevel): string {
  switch (level.approverType) {
    case 'manager':
      return "The owner's line manager";
    case 'role':
      return `Role: ${level.role?.name ?? '—'}`;
    case 'owner':
      return 'Record owner';
    case 'recordField':
      return `Person named on the record: ${level.recordField?.label ?? '—'}`;
    case 'people':
    case undefined:
      return '';
  }
}

export function levelRuleLines(level: ApprovalWorkflowLevel): string[] {
  const lines: string[] = [];
  const source = sourceLine(level);
  if (source) {
    lines.push(source);
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

export function sourceText(source: string | undefined): string {
  if (!source) {
    return '';
  }
  if (source.startsWith('role ')) {
    return `Role: ${source.slice('role '.length)}`;
  }
  return capitalise(source);
}

export function leftOutText(excluded: ApprovalExcludedPerson): string {
  return `${excluded.person.name} is left out: ${LEFT_OUT[excluded.reason].now}.`;
}

export function wasLeftOutText(excluded: ApprovalExcludedPerson): string {
  return `${excluded.person.name} was left out: ${LEFT_OUT[excluded.reason].then}.`;
}

export function progressText(approvedCount: number, requiredCount: number): string {
  return `${approvedCount} of ${requiredCount} approved`;
}

export function escalatedText(name: string, at: string, dateFormat?: DateFormat): string {
  return `Escalated to ${name} on ${formatDate(localDate(at), dateFormat)}`;
}

export function reminderSentText(at: string, dateFormat?: DateFormat): string {
  return `Reminder sent on ${formatDate(localDate(at), dateFormat)}`;
}

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
      return { word: 'Not decided', tone: 'neutral', detail: `— ${notDecidedWhy(outcome)}` };
    }
  }
}

export function chainStateText(
  level: ApprovalChainLevel,
  outcome: ApprovalRound['outcome'],
  dateFormat?: DateFormat,
): string {
  const { word, detail } = chainState(level, outcome, dateFormat);
  return detail ? `${word} ${detail}` : word;
}

function personName(person: ApprovalPerson): string {
  return person.active ? person.name : `${person.name} (inactive)`;
}

function ruleDetail(level: Pick<ApprovalChainLevel, 'ruleText'>): string {
  return level.ruleText ? `— ${level.ruleText}` : '';
}

function notDecidedWhy(outcome: ApprovalRound['outcome']): string {
  switch (outcome) {
    case 'recalled':
      return 'submission recalled';
    case 'restarted':
      return 'approval restarted';
    case 'superseded':
      return 're-submitted';
    case 'cancelled':
    case 'open':
    case 'approved':
    case 'declined':
      return 'activity deleted';
  }
}
