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
  swbsCodesText,
  useFinancialRecord,
} from './gsap-financials';
import type { FinancialLevel, FinancialRecordView } from './gsap-financials';
import { GsapFigures } from './GsapFigures';
import styles from './FinancialsSection.module.css';
import { LABEL } from './line-item-text';

/** Latest transactions shown on the activity page (015 FR-015). */
const LATEST = 5;

/**
 * The white card of the Financials section (design system §5.3): the title "Financials" with its ⓘ
 * (015 plan, "User-visible messages"), open — never collapsed (FR-015, FR-017).
 */
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
  // Refused: the in-place note of 015 (T044); 014's access pass decides who sees the card.
  if (error instanceof ApiError && (error.kind === 'notAllowed' || error.kind === 'forbidden')) {
    return <p className={styles.note}>{NO_ACCESS}</p>;
  }
  // No figures for the record yet: an API without GSAP data answers 404 (before 015 is served).
  if (error instanceof ApiError && error.kind === 'notFound') {
    return <p className={styles.note}>{NO_DATA}</p>;
  }
  return error ? <ErrorState error={error} onRetry={onRetry} /> : <Skeleton lines={4} />;
}

/**
 * The activity page's **Financials** card (015 US1, FR-011 – FR-015), open and in the Overview,
 * right after the summary: the five figures with the % committed meter and the red warnings, the
 * **By line item** table, the **Latest transactions** (at most 5, newest first), **View all n
 * transactions in Financials** and "GSAP data as of …". Amounts are in the reader's display currency,
 * as the API answers them; the web app never converts (D-061).
 */
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

/**
 * The line-item page's **Financials** section (015 FR-017): the figures, the **Transactions** table of
 * all its transactions (paged by 50) and the SWBS code from GSAP data.
 */
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
          <p className={styles.swbs}>
            <span className={styles.swbsLabel}>{LABEL.swbsCode}:</span>{' '}
            {swbsCodesText(view.swbsCodes)}
          </p>
          <Source view={view} />
        </div>
      )}
    </Frame>
  );
}

/** The figures, or the sentence of a record without transactions (FR-014) above them. */
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

/** "GSAP data as of 19 Oct 2026, 09:00 · Demo data (simulated GSAP)", or "No GSAP data loaded yet." */
function Source({ view }: { view: FinancialRecordView }) {
  const { dateTime } = useFormatters();
  return (
    <p className={styles.source}>
      {view.source.dataAsOf ? sourceText(view.source, dateTime) : NO_DATA}
    </p>
  );
}
