import { Card } from '@/ui/Card/Card';
import { toCents } from '@/format/decimal';
import { centsInput, formatMonth, moneyText } from './line-item-math';
import type { LineItem } from './api';
import {
  customTotalText,
  NOT_SPREAD_YET,
  OUTSIDE_TIP,
  SECTION_SPREAD,
  SPREAD_LABELS,
} from './line-item-text';
import styles from './LineItemPage.module.css';

/** The months of the spread, amounts right-aligned under their header (design system §5.4). */
export function LineItemSpreadCard({ lineItem }: { lineItem: LineItem }) {
  const { spread } = lineItem;
  const code = lineItem.currency.code;
  return (
    <Card title={SECTION_SPREAD} subtitle={SPREAD_LABELS[spread.method]}>
      {spread.state === 'notSpread' ? (
        <p className={styles.note}>{NOT_SPREAD_YET}</p>
      ) : (
        <div className={styles.stack}>
          <table className={styles.months}>
            <caption className="visually-hidden">Month amounts</caption>
            <thead>
              <tr>
                <th scope="col">Month</th>
                <th scope="col" className={styles.number}>
                  {`Planned cost (${code})`}
                </th>
              </tr>
            </thead>
            <tbody>
              {spread.months.map((month) => (
                <tr key={month.month}>
                  <td>
                    {formatMonth(month.month)}
                    {month.outsideActivityDates && (
                      <span className={styles.outside}> — {OUTSIDE_TIP}</span>
                    )}
                  </td>
                  <td className={styles.number}>{moneyText(month.amount)}</td>
                </tr>
              ))}
            </tbody>
          </table>
          {spread.method === 'custom' && spread.customTotal && (
            <p className={styles.note}>{customTotal(lineItem)}</p>
          )}
        </div>
      )}
    </Card>
  );
}

/** "12,000.00 of 10,000.00 EUR spread — 2,000.00 EUR too much" for a saved custom spread. */
function customTotal(lineItem: LineItem): string {
  const sum = lineItem.spread.customTotal;
  if (!sum) {
    return '';
  }
  const total = toCents(sum.amount);
  const cost = toCents(lineItem.plannedCost.amount);
  const difference = cost - total;
  return customTotalText(centsInput(total), centsInput(cost), lineItem.currency.code, {
    sign: difference === 0n ? 0 : difference > 0n ? 1 : -1,
    amount: centsInput(difference < 0n ? -difference : difference),
  });
}
