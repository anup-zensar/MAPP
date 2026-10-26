import { CircleAlert } from 'lucide-react';
import { useId, useState } from 'react';
import type { ReactNode } from 'react';
import { Link } from 'react-router';
import { ApiError } from '@/api/errors';
import { Button } from '@/ui/Button/Button';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { InfoTip } from '@/ui/InfoTip/InfoTip';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { recordHref } from '@/ui/record-links';
import { useFormatters } from '@/format/useFormatters';
import {
  cardTip,
  emptyText,
  FINANCIALS_TITLE,
  financialsHref,
  NO_ACCESS,
  NO_DATA,
  sourceText,
  swbsCodesText,
  TRANSACTIONS_PAGE_SIZE,
  TYPE_WORD,
  useFinancialRecord,
  useRecordTransactions,
  viewAllText,
} from './gsapFinancials';
import type {
  FinancialFiguresData,
  FinancialLevel,
  FinancialRecordView,
  FinancialTransactionRow,
} from './gsapFinancials';
import { GsapFigures } from './GsapFigures';
import styles from './FinancialsSection.module.css';
import { formatMonth, moneyText } from './lineItemMath';
import { LABEL } from './lineItemText';

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
            <Latest
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
            <Transactions lineItemId={lineItemId} count={view.figures.transactionCount} />
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

/** The four GSAP figures as table cells, numbers right-aligned under their headers. */
function FigureCells({ figures }: { figures: FinancialFiguresData }) {
  const over = figures.committedAbovePlanned;
  return (
    <>
      <td className={styles.number}>
        <span className={over ? styles.negative : undefined}>{moneyText(figures.committed)}</span>
        {over && (
          <span className={styles.flag}>
            <CircleAlert aria-hidden="true" />
            Above planned cost
          </span>
        )}
      </td>
      <td className={styles.number}>{moneyText(figures.invoiced)}</td>
      <td className={styles.number}>{moneyText(figures.paid)}</td>
      <td className={styles.number}>{moneyText(figures.accrued)}</td>
    </>
  );
}

function FigureHeads({ code, first }: { code: string; first: string }) {
  return (
    <thead>
      <tr>
        <th scope="col">{first}</th>
        <th scope="col" className={styles.number}>{`Committed (${code})`}</th>
        <th scope="col" className={styles.number}>{`Invoiced (${code})`}</th>
        <th scope="col" className={styles.number}>{`Paid (${code})`}</th>
        <th scope="col" className={styles.number}>{`Accrued (${code})`}</th>
      </tr>
    </thead>
  );
}

/** **By line item**: each line item's name as a link and its four figures (FR-015). */
function Breakdown({ view }: { view: FinancialRecordView }) {
  const code = view.figures.committed.currency;
  return (
    <TableBlock title="By line item">
      <table className={styles.table}>
        <caption className="visually-hidden">Financial figures by line item</caption>
        <FigureHeads code={code} first="Line item" />
        <tbody>
          {view.breakdown.map((row) => (
            <tr key={row.record.id}>
              <td className={styles.name}>
                <Link to={recordHref('lineItem', row.record.id)}>{row.record.name}</Link>
              </td>
              <FigureCells figures={row.figures} />
            </tr>
          ))}
        </tbody>
      </table>
    </TableBlock>
  );
}

/** A sub-heading and a table that scrolls sideways inside the card, never the page. */
function TableBlock({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className={styles.block}>
      <h3 className={styles.subheading}>{title}</h3>
      <div
        className={styles.scroll}
        role="region"
        aria-label={`${title}, scrolls sideways`}
        tabIndex={0}
      >
        {children}
      </div>
    </div>
  );
}

function TransactionRows({ rows }: { rows: FinancialTransactionRow[] }) {
  const { date } = useFormatters();
  return (
    <tbody>
      {rows.map((row) => (
        <tr key={row.id}>
          <td className={styles.name}>
            <span className={styles.rowName}>{row.name}</span>
            <span className={styles.rowMeta}>
              {TYPE_WORD[row.type]}
              {row.accrualMonth ? ` · ${formatMonth(row.accrualMonth)}` : ''}
              {row.vendor ? ` · ${row.vendor.name}` : ''}
            </span>
          </td>
          <td className={styles.date}>{date(row.postingDate)}</td>
          <td className={styles.number}>{moneyText(row.amount)}</td>
          <td className={styles.number}>
            {row.amountDisplay ? moneyText(row.amountDisplay) : '—'}
          </td>
        </tr>
      ))}
    </tbody>
  );
}

function TransactionHeads({ code }: { code: string }) {
  return (
    <thead>
      <tr>
        <th scope="col">Transaction</th>
        <th scope="col">Posting date</th>
        <th scope="col" className={styles.number}>
          Amount
        </th>
        <th scope="col" className={styles.number}>{`Amount (${code})`}</th>
      </tr>
    </thead>
  );
}

/** **Latest transactions**: at most 5, newest first, and the link to all of them (FR-015). */
function Latest({
  rows,
  count,
  recordId,
}: {
  rows: FinancialTransactionRow[];
  count: number;
  recordId: string;
}) {
  const { displayCurrency } = useFormatters();
  return (
    <TableBlock title="Latest transactions">
      <table className={styles.table}>
        <caption className="visually-hidden">The latest transactions, newest first</caption>
        <TransactionHeads code={displayCurrency} />
        <TransactionRows rows={rows} />
      </table>
      <p className={styles.more}>
        <Link to={financialsHref(recordId)}>{viewAllText(count)}</Link>
      </p>
    </TableBlock>
  );
}

/** **Transactions**: every transaction of the line item, newest first, 50 per page (FR-017). */
function Transactions({ lineItemId, count }: { lineItemId: string; count: number }) {
  const { displayCurrency } = useFormatters();
  const [page, setPage] = useState(1);
  const query = useRecordTransactions(lineItemId, page);
  const pages = Math.max(1, Math.ceil(count / TRANSACTIONS_PAGE_SIZE));
  return (
    <TableBlock title="Transactions">
      {query.data ? (
        <table className={styles.table}>
          <caption className="visually-hidden">
            The line item&apos;s transactions, newest first
          </caption>
          <TransactionHeads code={displayCurrency} />
          <TransactionRows rows={query.data.items} />
        </table>
      ) : query.isError ? (
        <ErrorState error={query.error} onRetry={() => void query.refetch()} />
      ) : (
        <Skeleton lines={3} />
      )}
      {pages > 1 && (
        <div className={styles.pager}>
          <Button size="sm" disabled={page <= 1} onClick={() => setPage(page - 1)}>
            Previous
          </Button>
          <span>{`Page ${page} of ${pages}`}</span>
          <Button size="sm" disabled={page >= pages} onClick={() => setPage(page + 1)}>
            Next
          </Button>
        </div>
      )}
    </TableBlock>
  );
}
