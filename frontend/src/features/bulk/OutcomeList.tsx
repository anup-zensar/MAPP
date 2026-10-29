import { Link } from 'react-router';
import { recordHref } from '@/ui/record-links';
import styles from './bulk.module.css';
import type { BulkOutcome, BulkResult } from './bulk-types';

export interface OutcomeListProps {
  result: BulkResult;
  /** Skipped, failed or both; the records done are never listed. */
  outcome: 'skipped' | 'failed';
  title: string;
}

/** The records an action skipped or failed, each with its ID, a link and every reason (FR-013, FR-021). */
export function OutcomeList({ result, outcome, title }: OutcomeListProps) {
  const outcomes = result.outcomes.filter((o) => o.outcome === outcome);
  if (outcomes.length === 0) {
    return null;
  }
  return (
    <section className={styles.outcomes} aria-label={title}>
      <h3 className={styles.listTitle}>
        {title} ({outcomes.length})
      </h3>
      <ul className={styles.list}>
        {outcomes.map((o) => (
          <OutcomeItem key={o.recordId} outcome={o} />
        ))}
      </ul>
    </section>
  );
}

function OutcomeItem({ outcome }: { outcome: BulkOutcome }) {
  const name = outcome.name || outcome.recordId;
  return (
    <li className={styles.item}>
      <span className={styles.itemHead}>
        {outcome.name ? (
          <Link to={recordHref(outcome.level, outcome.recordId, outcome.planId)}>{name}</Link>
        ) : (
          name
        )}{' '}
        <span className={styles.id}>{outcome.recordId}</span>
      </span>
      <ul className={styles.reasons}>
        {outcome.reasons.map((reason) => (
          <li key={reason}>{reason}</li>
        ))}
      </ul>
    </li>
  );
}

/** The consequences a preview lists before confirming: re-approvals and effects (FR-011, FR-018). */
export function PreviewNotes({ result }: { result: BulkResult }) {
  return (
    <>
      {result.effects.length > 0 && (
        <ul className={styles.effects} aria-label="What else changes">
          {result.effects.map((effect) => (
            <li key={effect.text}>{effect.text}</li>
          ))}
        </ul>
      )}
      <OutcomeList result={result} outcome="skipped" title="Skipped" />
      <OutcomeList result={result} outcome="failed" title="Failed" />
    </>
  );
}
