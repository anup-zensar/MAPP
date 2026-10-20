import { ArrowDown, ArrowUp, ArrowUpDown, CircleAlert } from 'lucide-react';
import { Link } from 'react-router';
import { StatusChip } from '@/components';
import type { StatusTone } from '@/components';
import { recordHref } from '@/lib/recordLinks';
import { useFormatters } from '@/lib/useFormatters';
import type { FinancialTransactionRow, FinancialTransactionType } from './financialsApi';
import { transactionHref } from './financialsLinks';
import type { SortColumn, SortDirection } from './financialsLinks';
import {
  currencyMissingText,
  LINE_ITEM_IN_BIN,
  NOT_LINKED,
  recordLabel,
  TYPE_WORDS,
} from './financialsText';
import { ScrollBox } from './ScrollBox';
import styles from './TransactionsTable.module.css';

/** The type chips: one tone per type, the word carries the meaning. */
const TYPE_TONES: Record<FinancialTransactionType, StatusTone> = {
  purchaseOrder: 'active',
  invoice: 'done',
  payment: 'positive',
  accrual: 'neutral',
};

export interface TransactionsTableProps {
  rows: FinancialTransactionRow[];
  sort: SortColumn;
  direction: SortDirection;
  onSort: (column: SortColumn, direction: SortDirection) => void;
  /** The list's address query, kept behind the details panel ("?q=keynote"). */
  search: string;
  /** One card per transaction (phone width) instead of the table. */
  narrow: boolean;
}

/**
 * The transactions (FR-021): transaction (a link that opens its details panel), type, posting
 * date, vendor, amount in its own currency with the display currency under it, the linked line
 * item and activity or a "Not linked" chip, and the SWBS code. Every heading but SWBS code sorts.
 * On a phone it becomes one card per transaction (US3 scenario 6).
 */
export function TransactionsTable({
  rows,
  sort,
  direction,
  onSort,
  search,
  narrow,
}: TransactionsTableProps) {
  if (narrow) {
    return (
      <ul className={styles.cards} aria-label="Transactions">
        {rows.map((row) => (
          <TransactionCard key={row.id} row={row} search={search} />
        ))}
      </ul>
    );
  }

  const heading = (column: SortColumn, label: string, numeric = false) => (
    <SortHeading
      column={column}
      label={label}
      numeric={numeric}
      sort={sort}
      direction={direction}
      onSort={onSort}
    />
  );

  return (
    <ScrollBox label="Transactions" className={styles.scroll}>
      <table className={styles.table}>
        <caption className="visually-hidden">Transactions</caption>
        <thead>
          <tr>
            {heading('documentNumber', 'Transaction')}
            {heading('type', 'Type')}
            {heading('postingDate', 'Posting date')}
            {heading('vendor', 'Vendor')}
            {heading('amount', 'Amount', true)}
            {heading('linkedTo', 'Linked to')}
            <th scope="col">SWBS code</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <TransactionRow key={row.id} row={row} search={search} />
          ))}
        </tbody>
      </table>
    </ScrollBox>
  );
}

interface SortHeadingProps {
  column: SortColumn;
  label: string;
  numeric: boolean;
  sort: SortColumn;
  direction: SortDirection;
  onSort: (column: SortColumn, direction: SortDirection) => void;
}

/** A heading that sorts by its column; a second click turns the order round. */
function SortHeading({ column, label, numeric, sort, direction, onSort }: SortHeadingProps) {
  const current = sort === column;
  const Icon = !current ? ArrowUpDown : direction === 'asc' ? ArrowUp : ArrowDown;
  // Dates and amounts start with the newest or largest; words start with A.
  const first: SortDirection = column === 'postingDate' || column === 'amount' ? 'desc' : 'asc';

  return (
    <th
      scope="col"
      className={numeric ? styles.numeric : undefined}
      aria-sort={current ? (direction === 'asc' ? 'ascending' : 'descending') : 'none'}
    >
      <button
        type="button"
        className={styles.sort}
        onClick={() => onSort(column, current ? (direction === 'asc' ? 'desc' : 'asc') : first)}
      >
        {label}
        <Icon aria-hidden="true" className={current ? undefined : styles.idle} />
      </button>
    </th>
  );
}

function TransactionRow({ row, search }: { row: FinancialTransactionRow; search: string }) {
  const { date } = useFormatters();

  return (
    <tr>
      <th scope="row">
        <Link to={transactionHref(row.id, search)} className={styles.name}>
          {row.name}
        </Link>
        {row.description && <span className={styles.secondary}>{row.description}</span>}
      </th>
      <td>
        <StatusChip tone={TYPE_TONES[row.type]}>{TYPE_WORDS[row.type]}</StatusChip>
      </td>
      <td className={styles.nowrap}>{date(row.postingDate)}</td>
      <td>
        <Vendor row={row} />
      </td>
      <td className={styles.numeric}>
        <Amount row={row} />
      </td>
      <td>
        <LinkedTo row={row} />
      </td>
      <td className={styles.nowrap}>{row.swbsCode || '—'}</td>
    </tr>
  );
}

/** The phone layout of one transaction: name, type and date, amount, and where it is linked. */
function TransactionCard({ row, search }: { row: FinancialTransactionRow; search: string }) {
  const { date } = useFormatters();

  return (
    <li className={styles.card}>
      <div className={styles.cardTop}>
        <Link to={transactionHref(row.id, search)} className={styles.name}>
          {row.name}
        </Link>
        <span className={styles.cardAmount}>
          <Amount row={row} />
        </span>
      </div>
      <div className={styles.cardMeta}>
        <StatusChip tone={TYPE_TONES[row.type]}>{TYPE_WORDS[row.type]}</StatusChip>
        <span>{date(row.postingDate)}</span>
        {row.vendor && <span>{row.vendor.name}</span>}
      </div>
      <LinkedTo row={row} />
    </li>
  );
}

function Vendor({ row }: { row: FinancialTransactionRow }) {
  if (!row.vendor) {
    return <>—</>;
  }
  return (
    <>
      <span className={styles.vendor}>{row.vendor.name}</span>
      <span className={styles.secondary}>{row.vendor.code}</span>
    </>
  );
}

/** "9,800.00 EUR" with "$10,652.17" under it when the display currency differs. */
function Amount({ row }: { row: FinancialTransactionRow }) {
  const { money } = useFormatters();
  const differs = row.amountDisplay !== null && row.amountDisplay.currency !== row.amount.currency;

  return (
    <>
      <span className={styles.money}>{money(row.amount)}</span>
      {differs && <span className={styles.secondary}>{money(row.amountDisplay)}</span>}
      {row.currencyMissing && (
        <span className={styles.missing}>
          <CircleAlert aria-hidden="true" />
          <span>{currencyMissingText(row.amount.currency)}</span>
        </span>
      )}
    </>
  );
}

/** The line item and activity, each a link; or the "Not linked" or recycle-bin chip. */
function LinkedTo({ row }: { row: FinancialTransactionRow }) {
  const { link } = row;
  if (link.state === 'notLinked') {
    return (
      <span className={styles.linked}>
        <StatusChip tone="review">{NOT_LINKED}</StatusChip>
      </span>
    );
  }
  if (link.state === 'lineItemDeleted') {
    return (
      <span className={styles.linked}>
        <StatusChip tone="neutral">{LINE_ITEM_IN_BIN}</StatusChip>
        {link.lineItemId && <span className={styles.secondary}>{link.lineItemId}</span>}
      </span>
    );
  }
  const planId = link.path.find((record) => record.type === 'plan')?.id;
  const lineItem = link.path.find((record) => record.type === 'lineItem');
  const activity = link.path.find((record) => record.type === 'activity');

  return (
    <span className={styles.linked}>
      {lineItem ? (
        <Link to={recordHref('lineItem', lineItem.id)}>{recordLabel(lineItem)}</Link>
      ) : (
        link.lineItemId
      )}
      {activity && (
        <Link to={recordHref('activity', activity.id, planId)} className={styles.activityLink}>
          {activity.name}
        </Link>
      )}
    </span>
  );
}
