import { CircleAlert } from 'lucide-react';
import { useState } from 'react';
import type { ReactNode } from 'react';
import { Link } from 'react-router';
import { Button } from '@/ui/Button/Button';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { recordHref } from '@/ui/record-links';
import { useFormatters } from '@/format/useFormatters';
import {
  financialsHref,
  TRANSACTIONS_PAGE_SIZE,
  TYPE_WORD,
  useRecordTransactions,
  viewAllText,
} from './gsap-financials';
import type {
  FinancialFiguresData,
  FinancialRecordView,
  FinancialTransactionRow,
} from './gsap-financials';
import styles from './FinancialsSection.module.css';
import { moneyText } from './line-item-math';

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
export function Breakdown({ view }: { view: FinancialRecordView }) {
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
  const { date, month } = useFormatters();
  return (
    <tbody>
      {rows.map((row) => (
        <tr key={row.id}>
          <td className={styles.name}>
            <span className={styles.rowName}>{row.name}</span>
            <span className={styles.rowMeta}>
              {TYPE_WORD[row.type]}
              {row.accrualMonth ? ` · ${month(row.accrualMonth)}` : ''}
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
export function LatestTransactions({
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
export function LineItemTransactions({ lineItemId, count }: { lineItemId: string; count: number }) {
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
