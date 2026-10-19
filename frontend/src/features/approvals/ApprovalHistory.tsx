import { useId } from 'react';
import type { components } from '@/api/schema';
import { useFormatters } from '@/lib/useFormatters';
import type { ApprovalRound } from './approvalsApi';
import { actorText, approversText, historyActionText, roundOutcomeText } from './messages';
import styles from './ApprovalHistory.module.css';

type HistoryEntry = components['schemas']['ApprovalHistoryEntry'];

/** How many entries the compact history (the approvals page's preview) shows. */
const COMPACT_ENTRIES = 3;

export interface ApprovalHistoryProps {
  /** Every round, newest first (as the API answers them). */
  rounds: ApprovalRound[];
  /** Only the latest 3 entries, without the round headings (the approvals page's preview). */
  compact?: boolean;
}

/**
 * "Approval history" (FR-025): the rounds newest first ("Round 2 · Waiting at level 1 of 1 ·
 * Lubricants EU approval"), each with its entries newest first — when, what, which level, the
 * level's approvers at the time, who acted (on whose behalf), the comment and, for a restart, what
 * changed.
 */
export function ApprovalHistory({ rounds, compact = false }: ApprovalHistoryProps) {
  const headingId = useId();
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
              <h4 className={styles.roundHeading}>
                {`Round ${round.number} · ${roundOutcomeText(round)} · ${round.workflow.name}`}
              </h4>
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

/** One entry: "Approved · Level 1 · 12 Mar 2026, 14:32", who, the approvers, comment, changes. */
function Entry({ entry, withRound = false }: { entry: HistoryEntry; withRound?: boolean }) {
  const { dateTime } = useFormatters();
  const facts = [
    withRound ? `Round ${entry.roundNumber}` : '',
    entry.level > 0 ? entry.levelLabel : '',
  ].filter(Boolean);

  return (
    <li className={styles.entry}>
      <p className={styles.what}>
        <span className={styles.action}>{historyActionText(entry.action)}</span>
        {facts.map((fact) => ` · ${fact}`).join('')}
        {' · '}
        <time dateTime={entry.at}>{dateTime(entry.at)}</time>
      </p>
      <p>{actorText(entry.actor, entry.onBehalfOf)}</p>
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
