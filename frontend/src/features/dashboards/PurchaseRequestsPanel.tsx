import { useFormatters } from '@/format/useFormatters';
import { Card } from '@/ui/Card/Card';
import type { DashboardPurchaseRequests } from './api';
import { word } from './dashboard-text';
import styles from './Dashboards.module.css';

interface PurchaseRequestsPanelProps {
  purchaseRequests: DashboardPurchaseRequests;
}

export function PurchaseRequestsPanel({ purchaseRequests }: PurchaseRequestsPanelProps) {
  const { money } = useFormatters();
  if (!purchaseRequests.available) {
    return (
      <Card title="Purchase requests">
        <p className={styles.note}>Purchase requests are switched off in Settings.</p>
      </Card>
    );
  }
  return (
    <Card
      title="Purchase requests"
      subtitle={`${purchaseRequests.waitingForGsap} approved and waiting for GSAP`}
    >
      <table className={styles.table}>
        <caption>Purchase requests on these plans by status</caption>
        <thead>
          <tr>
            <th scope="col">Status</th>
            <th scope="col" className={styles.number}>
              Purchase requests
            </th>
            <th scope="col" className={styles.number}>
              Amount
            </th>
          </tr>
        </thead>
        <tbody>
          {purchaseRequests.byStatus.map((row) => (
            <tr key={row.status}>
              <th scope="row">{word(row.status)}</th>
              <td className={styles.number}>{row.count}</td>
              <td className={styles.number}>{money(row.amount)}</td>
            </tr>
          ))}
        </tbody>
        <tfoot>
          <tr>
            <th scope="row">Total</th>
            <td className={styles.number}>{purchaseRequests.count}</td>
            <td className={styles.number}>{money(purchaseRequests.amount)}</td>
          </tr>
        </tfoot>
      </table>
    </Card>
  );
}
