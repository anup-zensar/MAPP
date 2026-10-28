import { Link } from 'react-router';
import type { components } from '@/api/generated/schema';
import { recordHref } from '@/ui/record-links';
import { useFormatters } from '@/format/useFormatters';
import styles from './RecordFinancials.module.css';

type FinancialRecordView = components['schemas']['FinancialRecordView'];
type Level = components['schemas']['AttributeLevel'];

const BREAKDOWN_HEADING: Record<Level, string> = {
  plan: 'By campaign',
  campaign: 'By initiative',
  initiative: 'By activity',
  activity: 'By line item',
  lineItem: '',
  purchaseRequest: '',
  purchaseRequestLine: '',
};

/** "By campaign" / "By initiative" / "By activity": each child with its four figures. */
export function FinancialsBreakdown({
  view,
  planId,
}: {
  view: FinancialRecordView;
  planId: string;
}) {
  const { money } = useFormatters();
  const heading = BREAKDOWN_HEADING[view.record.type];
  return (
    <div className={styles.scroll}>
      <table className={styles.table}>
        <caption className={styles.tableTitle}>{heading}</caption>
        <thead>
          <tr>
            <th scope="col">Name</th>
            <th scope="col" className={styles.number}>
              Committed
            </th>
            <th scope="col" className={styles.number}>
              Invoiced
            </th>
            <th scope="col" className={styles.number}>
              Paid
            </th>
            <th scope="col" className={styles.number}>
              Accrued
            </th>
          </tr>
        </thead>
        <tbody>
          {view.breakdown.map(({ record, figures }) => (
            <tr key={record.id}>
              <th scope="row">
                <Link to={recordHref(record.type, record.id, planId)}>{record.name}</Link>
                {figures.committedAbovePlanned && (
                  <span className={styles.rowProblem}> · Committed above planned cost</span>
                )}
              </th>
              <td className={styles.number}>{money(figures.committed)}</td>
              <td className={styles.number}>{money(figures.invoiced)}</td>
              <td className={styles.number}>{money(figures.paid)}</td>
              <td className={styles.number}>{money(figures.accrued)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
