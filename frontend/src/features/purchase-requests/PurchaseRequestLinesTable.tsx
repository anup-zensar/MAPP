import { Link } from 'react-router';
import { ScrollBox } from '@/features/financials';
import { Card } from '@/ui/Card/Card';
import { InfoTip } from '@/ui/InfoTip/InfoTip';
import { StatusChip } from '@/ui/StatusChip/StatusChip';
import { recordHref } from '@/ui/record-links';
import { formatMoneyText } from '@/format/decimal';
import { useFormatters } from '@/format/useFormatters';
import type { PurchaseRequest, PurchaseRequestLine } from './api';
import { HxKey } from './HxKey';
import { CHANGED, LINES_TITLE, poSourceText } from './page-text';
import {
  FIELD_LABELS,
  FILLED_BY_GSAP,
  GSAP_SOURCE_WORDS,
  HX_LABEL,
  SWBS_LABEL,
} from './purchase-request-text';
import styles from './PurchaseRequestPage.module.css';

const EMPTY = '—';

function Header({ label, tip }: { label: string; tip?: string }) {
  return (
    <th scope="col">
      {tip ? (
        <span className={styles.headerTip}>
          {label}
          <InfoTip label={label} text={tip} />
        </span>
      ) : (
        label
      )}
    </th>
  );
}

/**
 * The PR's lines (032 FR-011, FR-020, FR-043; 033 FR-004): the line item, its HX key and SWBS code,
 * the amounts, and GSAP's PO fields — read-only, filled by GSAP, or by the simulated GSAP.
 */
export function PurchaseRequestLinesTable({
  purchaseRequest,
}: {
  purchaseRequest: PurchaseRequest;
}) {
  const source = GSAP_SOURCE_WORDS[purchaseRequest.gsapSource];
  return (
    <Card title={LINES_TITLE} subtitle={source ? poSourceText(source) : undefined}>
      <ScrollBox label={LINES_TITLE} className={styles.scroll}>
        <table className={styles.table}>
          <thead>
            <tr>
              <Header label="#" />
              <Header label={FIELD_LABELS.lineItem} />
              <Header label={FIELD_LABELS.hxKey} tip={HX_LABEL} />
              <Header label={FIELD_LABELS.lineAmount} />
              <Header label={FIELD_LABELS.deliveryDate} />
              <Header label={FIELD_LABELS.overallLimit} />
              <Header label={FIELD_LABELS.swbsCode} tip={SWBS_LABEL} />
              <Header label={FIELD_LABELS.sentOn} />
              <Header label={FIELD_LABELS.poNumber} tip={FILLED_BY_GSAP} />
              <Header label={FIELD_LABELS.poLine} tip={FILLED_BY_GSAP} />
              <Header label={FIELD_LABELS.poLineStatus} tip={FILLED_BY_GSAP} />
              <Header label={FIELD_LABELS.poDate} tip={FILLED_BY_GSAP} />
              <Header label={FIELD_LABELS.poValue} tip={FILLED_BY_GSAP} />
              <Header label={FIELD_LABELS.lineInvoiced} tip={FILLED_BY_GSAP} />
            </tr>
          </thead>
          <tbody>
            {purchaseRequest.lines.map((line) => (
              <LineRow key={line.id} line={line} changePending={purchaseRequest.changePending} />
            ))}
          </tbody>
        </table>
      </ScrollBox>
    </Card>
  );
}

function LineRow({ line, changePending }: { line: PurchaseRequestLine; changePending: boolean }) {
  const { date } = useFormatters();
  const optionalDate = (value: string | null) => (value ? date(value) : EMPTY);
  return (
    <tr>
      <td className={styles.number}>{line.position}</td>
      <th scope="row">
        <span className={styles.withReason}>
          <Link to={recordHref('lineItem', line.lineItem.id)}>
            {`${line.lineItem.name} (${line.lineItem.id})`}
          </Link>
          {changePending && line.changed && <StatusChip tone="review">{CHANGED}</StatusChip>}
        </span>
      </th>
      <td>
        <HxKey value={line.hxKey} compact />
      </td>
      <td className={styles.number}>{formatMoneyText(line.amount)}</td>
      <td>{date(line.deliveryDate)}</td>
      <td className={styles.number}>
        {line.overallLimit ? formatMoneyText(line.overallLimit) : EMPTY}
      </td>
      <td>{line.lineItem.swbsCode || EMPTY}</td>
      <td>{optionalDate(line.sentOn)}</td>
      <td>{line.poNumber || EMPTY}</td>
      <td>{line.poLine || EMPTY}</td>
      <td>{line.poLineStatus || EMPTY}</td>
      <td>{optionalDate(line.poDate)}</td>
      <td className={styles.number}>{line.poValue ? formatMoneyText(line.poValue) : EMPTY}</td>
      <td className={styles.number}>{formatMoneyText(line.invoiced)}</td>
    </tr>
  );
}
