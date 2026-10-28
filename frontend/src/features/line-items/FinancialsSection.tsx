import { useId } from 'react';
import type { ReactNode } from 'react';
import { ApiError } from '@/api/errors';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { InfoTip } from '@/ui/InfoTip/InfoTip';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { useFormatters } from '@/format/useFormatters';
import { Breakdown, LatestTransactions, LineItemTransactions } from './FinancialsTables';
import {
  cardTip,
  emptyText,
  FINANCIALS_TITLE,
  NO_ACCESS,
  NO_DATA,
  sourceText,
  useFinancialRecord,
} from './gsap-financials';
import type { FinancialLevel, FinancialRecordView } from './gsap-financials';
import { GsapFigures } from './GsapFigures';
import styles from './FinancialsSection.module.css';

/** Latest transactions shown on the activity page (015 FR-015). */
const LATEST = 5;

/** The Financials card: open, never collapsed (015 FR-015, FR-017). */
function Frame({ level, children }: { level: FinancialLevel; children: ReactNode }) {
  const titleId = useId();
  return (
    <section id="financials" className={styles.card} aria-labelledby={titleId}>
      <div className={styles.header}>
        <h2 id={titleId} className={styles.title}>
          {FINANCIALS_TITLE}
        </h2>
        <InfoTip label={FINANCIALS_TITLE} text={cardTip(level)} />
      </div>
      <div className={styles.body}>{children}</div>
    </section>
  );
}

/** Loading, refused or failed: the card's body instead of the figures. */
function NotLoaded({ error, onRetry }: { error: unknown; onRetry: () => void }) {
  // Refused: the in-place note of 015 (T044); the access rules decide who sees the card.
  if (error instanceof ApiError && (error.kind === 'notAllowed' || error.kind === 'forbidden')) {
    return <p className={styles.note}>{NO_ACCESS}</p>;
  }
  // An API without GSAP data answers 404 for the record (before 015 is served).
  if (error instanceof ApiError && error.kind === 'notFound') {
    return <p className={styles.note}>{NO_DATA}</p>;
  }
  return error ? <ErrorState error={error} onRetry={onRetry} /> : <Skeleton lines={4} />;
}

export function ActivityFinancials({ activityId }: { activityId: string }) {
  const record = useFinancialRecord(activityId);
  const view = record.data;
  return (
    <Frame level="activity">
      {!view ? (
        <NotLoaded error={record.error} onRetry={() => void record.refetch()} />
      ) : (
        <div className={styles.stack}>
          <Figures view={view} level="activity" />
          {view.breakdown.length > 0 && <Breakdown view={view} />}
          {view.figures.transactionCount > 0 && (
            <LatestTransactions
              rows={view.latest.slice(0, LATEST)}
              count={view.figures.transactionCount}
              recordId={activityId}
            />
          )}
          <Source view={view} />
        </div>
      )}
    </Frame>
  );
}

export function LineItemFinancials({ lineItemId }: { lineItemId: string }) {
  const record = useFinancialRecord(lineItemId);
  const view = record.data;
  return (
    <Frame level="line item">
      {!view ? (
        <NotLoaded error={record.error} onRetry={() => void record.refetch()} />
      ) : (
        <div className={styles.stack}>
          <Figures view={view} level="line item" />
          {view.figures.transactionCount > 0 && (
            <LineItemTransactions lineItemId={lineItemId} count={view.figures.transactionCount} />
          )}
          <Source view={view} />
        </div>
      )}
    </Frame>
  );
}

function Figures({ view, level }: { view: FinancialRecordView; level: FinancialLevel }) {
  return (
    <>
      {view.source.dataAsOf && view.figures.transactionCount === 0 && (
        <p className={styles.note}>{emptyText(level)}</p>
      )}
      <GsapFigures figures={view.figures} accrualMonth={view.source.accrualMonth} />
    </>
  );
}

function Source({ view }: { view: FinancialRecordView }) {
  const { dateTime } = useFormatters();
  return (
    <p className={styles.source}>
      {view.source.dataAsOf ? sourceText(view.source, dateTime) : NO_DATA}
    </p>
  );
}
