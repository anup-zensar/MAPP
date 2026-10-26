import { formatAmount, toCents } from '@/format/decimal';
import styles from './BudgetByMonth.module.css';
import type { BudgetByMonthPeriods } from './api';
import { amountText, overBy } from './phasing-math';

type Money = { amount: string; currency: string };

/** "50,000.00" without the currency (the caption names it); "—" for none. */
function amount(value: Money | null | undefined): string {
  return value ? formatAmount(value.amount) : '—';
}

/** The sum of a row's amounts; null when none of them is set. */
function totalOf(values: (Money | null)[]): string | null {
  const set = values.filter((value): value is Money => value !== null);
  if (set.length === 0) {
    return null;
  }
  return amountText(set.reduce((total, value) => total + toCents(value.amount), 0n));
}

export function PeriodRow({
  label,
  values,
  extra,
  extraCount = 0,
  empty,
}: {
  label: string;
  values: (Money | null)[];
  extra?: Money[];
  extraCount?: number;
  /** Shown across the periods when the row has no amounts ("Not phased"). */
  empty?: string;
}) {
  const columns = extra?.length ?? extraCount;
  if (empty) {
    return (
      <tr>
        <th scope="row">{label}</th>
        <td colSpan={values.length + columns + 1} className={styles.muted}>
          {empty}
        </td>
      </tr>
    );
  }
  const total = totalOf([...values, ...(extra ?? [])]);
  return (
    <tr>
      <th scope="row">{label}</th>
      {values.map((value, index) => (
        <td key={index} className={styles.number}>
          {amount(value)}
        </td>
      ))}
      {extra
        ? extra.map((value, index) => (
            <td key={`extra-${index}`} className={styles.number}>
              {amount(value)}
            </td>
          ))
        : Array.from({ length: extraCount }, (_, index) => (
            <td key={`extra-${index}`} className={styles.number}>
              —
            </td>
          ))}
      <td className={`${styles.number} ${styles.total}`}>{total ?? '—'}</td>
    </tr>
  );
}

/** Budget − planned cost per period; red "Over by …" where the API says the period is over. */
export function DifferenceRow({
  periods,
  extraCount,
}: {
  periods: BudgetByMonthPeriods;
  extraCount: number;
}) {
  return (
    <tr>
      <th scope="row">Difference</th>
      {periods.difference.map((value, index) => {
        const over = periods.over[index];
        const by = over
          ? overBy(
              periods.budget[index]?.amount,
              periods.plannedCost[index]?.amount,
              periods.allocated?.[index]?.amount,
            )
          : 0n;
        return (
          <td key={index} className={`${styles.number} ${over ? styles.over : ''}`}>
            {over && by > 0n ? `Over by ${amountText(by)}` : amount(value)}
          </td>
        );
      })}
      {Array.from({ length: extraCount }, (_, index) => (
        <td key={`extra-${index}`} className={styles.number}>
          —
        </td>
      ))}
      <td className={`${styles.number} ${styles.total}`}>{totalOf(periods.difference) ?? '—'}</td>
    </tr>
  );
}
