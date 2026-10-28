import { useId } from 'react';
import type { components } from '@/api/generated/schema';
import { useFormatters } from '@/format/useFormatters';
import { subjectKindLabel } from '@/ui/subject-links';
import type { ApprovalRound } from './api';
import {
  historyActorText,
  historyEntryNote,
  historyEntryTitle,
  roundOutcomeText,
  rulesAppliedText,
} from './history-messages';
import { approversText } from './messages';
import styles from './ApprovalHistory.module.css';

type HistoryEntry = components['schemas']['ApprovalHistoryEntry'];

const COMPACT_ENTRIES = 3;

export interface ApprovalHistoryProps {
  rounds: ApprovalRound[];
  compact?: boolean;
}

export function ApprovalHistory({ rounds, compact = false }: ApprovalHistoryProps) {
  const headingId = useId();
  const { money } = useFormatters();
  const hasEntries = rounds.some((round) => round.history.length > 0);

  return (
    <section aria-labelledby={headingId} className={styles.history}>
      <h3 id={headingId} className={styles.heading}>
        Approval history
      </h3>
      {!hasEntries ? (
        <p className={styles.empty}>No approval history yet.</p>
      ) : compact ? (
        <ol role="list" className={styles.entries}>
          {rounds
            .flatMap((round) => round.history)
            .slice(0, COMPACT_ENTRIES)
            .map((entry) => (
              <Entry key={entry.id} entry={entry} withRound />
            ))}
        </ol>
      ) : (
        <ol role="list" className={styles.rounds}>
          {rounds.map((round) => (
            <li key={round.id}>
              <h4 className={styles.roundHeading}>{roundTitle(round)}</h4>
              {round.facts && (
                <p className={styles.facts}>{rulesAppliedText(round.facts, money)}</p>
              )}
              <ol role="list" className={styles.entries}>
                {round.history.map((entry) => (
                  <Entry key={entry.id} entry={entry} />
                ))}
              </ol>
            </li>
          ))}
        </ol>
      )}
    </section>
  );
}

function roundTitle(round: ApprovalRound): string {
  const kind = round.kind === 'activity' ? '' : ` · ${subjectKindLabel(round.kind)}`;
  return `Round ${round.number}${kind} · ${roundOutcomeText(round)} · ${round.workflow.name}`;
}

function Entry({ entry, withRound = false }: { entry: HistoryEntry; withRound?: boolean }) {
  const { dateTime } = useFormatters();
  const note = historyEntryNote(entry);
  const facts = [
    withRound ? `Round ${entry.roundNumber}` : '',
    entry.level > 0 ? entry.levelLabel : '',
  ].filter(Boolean);

  return (
    <li className={styles.entry}>
      <p className={styles.what}>
        <span className={styles.action}>{historyEntryTitle(entry)}</span>
        {facts.map((fact) => ` · ${fact}`).join('')}
        {' · '}
        <time dateTime={entry.at}>{dateTime(entry.at)}</time>
      </p>
      <p>{historyActorText(entry)}</p>
      {note && <p className={styles.meta}>{note}</p>}
      {entry.approvers.length > 0 && (
        <p className={styles.meta}>{approversText(entry.approvers)}</p>
      )}
      {entry.comment && <p className={styles.comment}>{`"${entry.comment}"`}</p>}
      {entry.changes.length > 0 && (
        <ul className={styles.changes}>
          {entry.changes.map((change) => (
            <li key={change.text}>{change.text}</li>
          ))}
        </ul>
      )}
    </li>
  );
}
