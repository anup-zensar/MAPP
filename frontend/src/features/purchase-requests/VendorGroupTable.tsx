import { ChevronDown, ChevronRight } from 'lucide-react';
import { useState } from 'react';
import { Link } from 'react-router';
import { InfoTip } from '@/ui/InfoTip/InfoTip';
import { purchaseRequestHref, recordHref } from '@/ui/record-links';
import { localDate } from '@/format/intl';
import { useFormatters } from '@/format/useFormatters';
import type {
  PurchaseRequestListPage,
  PurchaseRequestRow,
  PurchaseRequestRowLine,
  PurchaseRequestVendorGroup,
} from './api';
import { HxKey } from './HxKey';
import { countText, NOT_COMPLETED, TABLE_LABEL, WAITING_MARK } from './list-text';
import { FIELD_LABELS, FILLED_BY_GSAP, HX_LABEL, linesText } from './purchase-request-text';
import { PurchaseRequestStatusChip } from './PurchaseRequestStatusChip';
import styles from './VendorGroupTable.module.css';

const COLUMNS = [
  'Purchase request',
  'ID',
  'Status',
  'Amount',
  'Invoiced',
  'Owner',
  'Created',
  'Approval workflow',
  'GSAP PR number',
];

/**
 * The Purchase requests page's table (032 FR-044, FR-047): a heading row per vendor with its
 * number of purchase requests and totals as the API counts them, its purchase requests, and under
 * each one, when opened, its lines with their line items.
 */
export function VendorGroupTable({ page }: { page: PurchaseRequestListPage }) {
  const [open, setOpen] = useState<string[]>([]);
  const toggle = (id: string) =>
    setOpen((current) =>
      current.includes(id) ? current.filter((each) => each !== id) : [...current, id],
    );

  return (
    <div className={styles.scroll} role="region" aria-label={TABLE_LABEL} tabIndex={0}>
      <table className={styles.table}>
        <caption className="visually-hidden">{TABLE_LABEL}</caption>
        <thead>
          <tr>
            {COLUMNS.map((column) => (
              <th key={column} scope="col">
                {column}
              </th>
            ))}
          </tr>
        </thead>
        {page.vendors.map((group) => (
          <VendorGroup
            key={group.vendor}
            group={group}
            rows={page.items.filter((row) => sameVendor(row.vendor, group.vendor))}
            open={open}
            onToggle={toggle}
          />
        ))}
      </table>
    </div>
  );
}

function sameVendor(vendor: string, group: string): boolean {
  return vendor.trim().toLowerCase() === group.trim().toLowerCase();
}

interface VendorGroupProps {
  group: PurchaseRequestVendorGroup;
  rows: PurchaseRequestRow[];
  open: string[];
  onToggle: (id: string) => void;
}

function VendorGroup({ group, rows, open, onToggle }: VendorGroupProps) {
  const { money } = useFormatters();
  return (
    <tbody>
      <tr className={styles.vendor}>
        <th scope="rowgroup" colSpan={3}>
          <span className={styles.vendorName}>{group.label || NOT_COMPLETED}</span>
          <span className={styles.muted}> · {countText(group.purchaseRequests)}</span>
        </th>
        <td className={styles.numeric}>{money(group.amountDisplay)}</td>
        <td className={styles.numeric}>{money(group.invoicedDisplay)}</td>
        <td colSpan={COLUMNS.length - 5} />
      </tr>
      {rows.map((row) => (
        <PurchaseRequestRows
          key={row.id}
          row={row}
          open={open.includes(row.id)}
          onToggle={() => onToggle(row.id)}
        />
      ))}
    </tbody>
  );
}

interface RowProps {
  row: PurchaseRequestRow;
  open: boolean;
  onToggle: () => void;
}

function PurchaseRequestRows({ row, open, onToggle }: RowProps) {
  const { money, date } = useFormatters();
  const Icon = open ? ChevronDown : ChevronRight;
  return (
    <>
      <tr>
        <th scope="row" className={styles.name}>
          {row.lines.length > 0 && (
            <button
              type="button"
              className={styles.toggle}
              aria-expanded={open}
              aria-label={`${open ? 'Hide' : 'Show'} the lines of ${row.name}`}
              onClick={onToggle}
            >
              <Icon aria-hidden="true" />
            </button>
          )}
          <Link to={purchaseRequestHref(row.id)}>{row.name}</Link>
        </th>
        <td>{row.id}</td>
        <td>
          <PurchaseRequestStatusChip status={row.status} changePending={row.changePending} />
          {row.waitingForGsap && <span className={styles.waiting}>{WAITING_MARK}</span>}
        </td>
        <td className={styles.numeric}>{money(row.amountDisplay)}</td>
        <td className={styles.numeric}>{money(row.invoicedDisplay)}</td>
        <td>{row.owner ? row.owner.name : '—'}</td>
        <td>{date(localDate(row.createdAt))}</td>
        <td>{row.approvalWorkflow?.name ?? '—'}</td>
        <td>{row.gsapPrNumber || '—'}</td>
      </tr>
      {open && (
        <tr className={styles.linesRow}>
          <td colSpan={COLUMNS.length}>
            <LinesTable name={row.name} lines={row.lines} />
          </td>
        </tr>
      )}
    </>
  );
}

const LINE_COLUMNS: { label: string; tip?: string }[] = [
  { label: FIELD_LABELS.lineItem },
  { label: FIELD_LABELS.hxKey, tip: HX_LABEL },
  { label: FIELD_LABELS.lineAmount },
  { label: FIELD_LABELS.deliveryDate },
  { label: FIELD_LABELS.sentOn },
  { label: FIELD_LABELS.poNumber, tip: FILLED_BY_GSAP },
  { label: FIELD_LABELS.poLine, tip: FILLED_BY_GSAP },
  { label: FIELD_LABELS.poLineStatus, tip: FILLED_BY_GSAP },
  { label: FIELD_LABELS.lineInvoiced, tip: FILLED_BY_GSAP },
];

function LinesTable({ name, lines }: { name: string; lines: PurchaseRequestRowLine[] }) {
  const { money, date } = useFormatters();
  return (
    <table className={styles.lines}>
      <caption className={styles.linesCaption}>{`${linesText(lines.length)} of ${name}`}</caption>
      <thead>
        <tr>
          {LINE_COLUMNS.map((column) => (
            <th key={column.label} scope="col">
              {column.label}
              {column.tip && <InfoTip label={column.label} text={column.tip} />}
            </th>
          ))}
        </tr>
      </thead>
      <tbody>
        {lines.map((line) => (
          <tr key={line.id}>
            <th scope="row">
              <Link to={recordHref('lineItem', line.lineItem.id)}>
                {`${line.lineItem.name} (${line.lineItem.id})`}
              </Link>
            </th>
            <td>
              <HxKey value={line.hxKey} compact />
            </td>
            <td className={styles.numeric}>{money(line.amount)}</td>
            <td>{date(line.deliveryDate)}</td>
            <td>{line.sentOn ? date(line.sentOn) : '—'}</td>
            <td>{line.poNumber || '—'}</td>
            <td>{line.poLine || '—'}</td>
            <td>{line.poLineStatus || '—'}</td>
            <td className={styles.numeric}>{money(line.invoiced)}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
