import { CircleAlert } from 'lucide-react';
import { Button } from '@/ui/Button/Button';
import { Card } from '@/ui/Card/Card';
import { formatWhole } from '@/format/decimal';
import { OutcomeGroup } from './OutcomeGroup';
import type { RateImportCheck } from './rates-api';
import {
  importLabel,
  NOTHING_TO_IMPORT,
  OUTCOME_WORDS,
  OUTCOMES,
  rowsToImport,
} from './rate-words';
import styles from './rates.module.css';

export interface ReviewCardProps {
  review: RateImportCheck;
  importing: boolean;
  failure: string | null;
  onImport: () => void;
  onCancel: () => void;
}

/** The review of a file: the counts, each outcome's rows, then Import n rates and Cancel. */
export function ReviewCard({ review, importing, failure, onImport, onCancel }: ReviewCardProps) {
  const toImport = rowsToImport(review.counts);

  return (
    <Card title={`Review of '${review.fileName}'`} subtitle="Nothing is saved until you import">
      <ul className={styles.counts} aria-label="Rows by outcome">
        {OUTCOMES.map((outcome) => (
          <li key={outcome}>
            <span className={styles.countLabel}>{OUTCOME_WORDS[outcome]}</span>
            <span className={styles.countValue}>{formatWhole(review.counts[outcome])}</span>
          </li>
        ))}
      </ul>
      {OUTCOMES.map((outcome) => {
        const rows = review.rows.filter((row) => row.outcome === outcome);
        return rows.length === 0 ? null : (
          <OutcomeGroup key={outcome} outcome={outcome} rows={rows} />
        );
      })}
      {failure && (
        <p role="alert" className={styles.error}>
          <CircleAlert aria-hidden="true" />
          {failure}
        </p>
      )}
      <div className={styles.actions}>
        {toImport > 0 ? (
          <Button variant="primary" aria-busy={importing || undefined} onClick={onImport}>
            {importLabel(toImport)}
          </Button>
        ) : (
          <p className={styles.actionsNote}>{NOTHING_TO_IMPORT}</p>
        )}
        <Button onClick={onCancel}>Cancel</Button>
      </div>
    </Card>
  );
}
