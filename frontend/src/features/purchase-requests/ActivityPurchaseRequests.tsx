import { Link } from 'react-router';
import { recordHref } from '@/ui/record-links';
import { formatMoneyText, formatWhole } from '@/format/decimal';
import { useActivityPurchaseRequests } from './api';
import type { ActivityPurchaseRequests as Section } from './api';
import { FIELD_LABELS, PURCHASE_REQUESTS } from './purchase-request-text';
import { SectionFrame, SectionNotLoaded } from './SectionFrame';
import { ACTIVITY_TIP, AVAILABLE, NO_ACTIVITY_PRS, TOTAL_REQUESTED } from './section-text';
import styles from './PurchaseRequestSections.module.css';

/**
 * An activity's Purchase requests section (032 FR-018): its Total requested in the plan currency
 * and, per line item, what is requested and still available, as the API counts them.
 */
export function ActivityPurchaseRequests({ activityId }: { activityId: string }) {
  const section = useActivityPurchaseRequests(activityId);
  return (
    <SectionFrame tip={ACTIVITY_TIP}>
      {section.data ? (
        <Loaded section={section.data} />
      ) : (
        <SectionNotLoaded error={section.error} onRetry={() => void section.refetch()} />
      )}
    </SectionFrame>
  );
}

function Loaded({ section }: { section: Section }) {
  return (
    <>
      <dl className={styles.figures}>
        <div className={styles.figure}>
          <dt className={styles.figureLabel}>{TOTAL_REQUESTED}</dt>
          <dd className={styles.figureValue}>{formatMoneyText(section.requested)}</dd>
        </div>
      </dl>
      {section.lineItems.length === 0 ? (
        <p className={styles.note}>{NO_ACTIVITY_PRS}</p>
      ) : (
        <div className={styles.scroll}>
          <table className={styles.table}>
            <caption className="visually-hidden">Purchase requests by line item</caption>
            <thead>
              <tr>
                <th scope="col">{FIELD_LABELS.lineItem}</th>
                <th scope="col" className={styles.numeric}>
                  {TOTAL_REQUESTED}
                </th>
                <th scope="col" className={styles.numeric}>
                  {AVAILABLE}
                </th>
                <th scope="col" className={styles.numeric}>
                  {PURCHASE_REQUESTS}
                </th>
              </tr>
            </thead>
            <tbody>
              {section.lineItems.map((row) => (
                <tr key={row.lineItemId}>
                  <th scope="row">
                    <Link to={`${recordHref('lineItem', row.lineItemId)}#purchase-requests`}>
                      {row.name}
                    </Link>
                    <span className={styles.secondary}>{row.lineItemId}</span>
                  </th>
                  <td className={styles.numeric}>{formatMoneyText(row.requested)}</td>
                  <td className={styles.numeric}>{formatMoneyText(row.availableToRequest)}</td>
                  <td className={styles.numeric}>{formatWhole(row.purchaseRequests)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </>
  );
}
