import { Plus } from 'lucide-react';
import { Link } from 'react-router';
import { LinkButton } from '@/ui/Button/Button';
import { purchaseRequestHref } from '@/ui/record-links';
import { formatMoneyText } from '@/format/decimal';
import { useFormatters } from '@/format/useFormatters';
import { useLineItemPurchaseRequests } from './api';
import type { LineItemPurchaseRequests as Section, Money } from './api';
import { FIELD_LABELS, NEW_PURCHASE_REQUEST } from './purchase-request-text';
import { PurchaseRequestStatusChip } from './PurchaseRequestStatusChip';
import { SectionFrame, SectionNotLoaded } from './SectionFrame';
import {
  aboveText,
  AVAILABLE,
  convertedText,
  LINE_ITEM_TIP,
  NO_LINE_ITEM_PRS,
  TOTAL_REQUESTED,
} from './section-text';
import styles from './PurchaseRequestSections.module.css';

/**
 * A line item's Purchase requests section (032 FR-001, FR-018): what its purchase requests reserve,
 * what is still available, the purchase requests the person can see and New purchase request.
 */
export function LineItemPurchaseRequests({ lineItemId }: { lineItemId: string }) {
  const section = useLineItemPurchaseRequests(lineItemId);
  const data = section.data;
  const create = data?.create;
  return (
    <SectionFrame
      tip={LINE_ITEM_TIP}
      action={
        create?.available && (
          <LinkButton
            to={`/purchase-requests/new?${new URLSearchParams({ lineItemId })}`}
            icon={<Plus />}
            size="sm"
          >
            {NEW_PURCHASE_REQUEST}
          </LinkButton>
        )
      }
    >
      {data ? (
        <Loaded section={data} />
      ) : (
        <SectionNotLoaded error={section.error} onRetry={() => void section.refetch()} />
      )}
    </SectionFrame>
  );
}

function Loaded({ section }: { section: Section }) {
  const { create, notice } = section;
  return (
    <>
      {notice && <p className={styles.note}>{notice}</p>}
      <Figures requested={section.requested} available={section.availableToRequest} />
      {!create.available && create.reason && create.reason !== notice && (
        <p className={styles.note}>{create.reason}</p>
      )}
      {section.purchaseRequests.length === 0 ? (
        <p className={styles.note}>{NO_LINE_ITEM_PRS}</p>
      ) : (
        <Rows section={section} />
      )}
    </>
  );
}

function Figures({ requested, available }: { requested: Money; available: Money }) {
  const above = available.amount.startsWith('-');
  return (
    <dl className={styles.figures}>
      <div className={styles.figure}>
        <dt className={styles.figureLabel}>{TOTAL_REQUESTED}</dt>
        <dd className={styles.figureValue}>{formatMoneyText(requested)}</dd>
      </div>
      <div className={styles.figure}>
        <dt className={styles.figureLabel}>{AVAILABLE}</dt>
        <dd className={styles.figureValue}>{formatMoneyText(available)}</dd>
        {above && (
          <p className={styles.above}>
            {aboveText(formatMoneyText({ ...available, amount: available.amount.slice(1) }))}
          </p>
        )}
      </div>
    </dl>
  );
}

function Rows({ section }: { section: Section }) {
  const { money, date } = useFormatters();
  return (
    <div className={styles.scroll}>
      <table className={styles.table}>
        <caption className="visually-hidden">{`Purchase requests on ${section.lineItemId}`}</caption>
        <thead>
          <tr>
            <th scope="col">Purchase request</th>
            <th scope="col">{FIELD_LABELS.status}</th>
            <th scope="col" className={styles.numeric}>
              {FIELD_LABELS.lineAmount}
            </th>
            <th scope="col">{FIELD_LABELS.sentOn}</th>
            <th scope="col">{FIELD_LABELS.poNumber}</th>
            <th scope="col">{FIELD_LABELS.owner}</th>
          </tr>
        </thead>
        <tbody>
          {section.purchaseRequests.map((row) => (
            <tr key={row.lineId}>
              <th scope="row">
                <Link to={purchaseRequestHref(row.purchaseRequestId)}>{row.name}</Link>
                <span className={styles.secondary}>{row.purchaseRequestId}</span>
              </th>
              <td>
                <PurchaseRequestStatusChip status={row.status} changePending={row.changePending} />
              </td>
              <td className={styles.numeric}>
                {money(row.amount)}
                {row.amount.currency !== row.amountInLineItemCurrency.currency && (
                  <span className={styles.secondary}>
                    {convertedText(formatMoneyText(row.amountInLineItemCurrency))}
                  </span>
                )}
              </td>
              <td>{row.sentOn ? date(row.sentOn) : '—'}</td>
              <td>{row.poNumber || '—'}</td>
              <td>{row.owner ? row.owner.name : '—'}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
