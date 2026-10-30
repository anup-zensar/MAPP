import type { components } from '@/api/generated/schema';
import { leftOutText } from './chain-messages';
import { names } from './messages';
import styles from './approval-dialogs.module.css';

type SubmissionLevel = components['schemas']['ApprovalSubmissionLevel'];

/** Each level's outcome, with approvers resolved, also those from a record field. */
export function SubmissionOutcomes({ outcomes }: { outcomes: readonly SubmissionLevel[] }) {
  return (
    <ol role="list" className={styles.levels}>
      {outcomes.map((level) => (
        <OutcomeItem key={level.number} level={level} />
      ))}
    </ol>
  );
}

function OutcomeItem({ level }: { level: SubmissionLevel }) {
  const backup = level.backupUsed ? level.approvers[0] : undefined;
  const notes = [
    ...level.excluded.map(leftOutText),
    backup ? `The backup approver ${backup.name} approves this level.` : '',
    level.noApprover ? level.advice : '',
  ].filter(Boolean);

  return (
    <li className={styles.outcome}>
      <span>{outcomeText(level)}</span>
      {notes.map((note) => (
        <span key={note} className={styles.outcomeNote}>
          {note}
        </span>
      ))}
    </li>
  );
}

function outcomeText(level: SubmissionLevel): string {
  switch (level.outcome) {
    case 'skipped':
      return `${level.label} · Not needed — ${level.ruleText}`;
    case 'autoApproved':
      return `${level.label} · Approved automatically — ${level.ruleText}`;
    case 'decide': {
      if (level.approvers.length === 0) {
        return `${level.label} · No approver`;
      }
      const source = level.source && !level.backupUsed ? ` (${level.source})` : '';
      return `${level.label} · ${names(level.approvers)}${source}`;
    }
  }
}
