import { useState } from 'react';
import type { components } from '@/api/generated/schema';
import { InfoTip } from '@/ui/InfoTip/InfoTip';
import { SegmentedFilter } from '@/ui/SegmentedFilter/SegmentedFilter';
import { cx } from '@/ui/cx';
import { formatAmount } from '@/format/decimal';
import { monthColumns, quarterColumns } from './line-item-math';
import {
  EMPTY,
  GRID_NO_DATES,
  GRID_TITLE,
  gridCaption,
  LINE_ITEM_COLUMN,
  MONTHLY,
  OUTSIDE_DATES,
  OUTSIDE_TIP,
  QUARTERLY,
  TOTAL,
} from './line-item-text';
import styles from './SpreadGrid.module.css';

type Grid = components['schemas']['SpreadGrid'];
type Money = components['schemas']['Money'];
type Period = 'monthly' | 'quarterly';

/** Where the browser remembers Monthly or Quarterly (007 plan, "Line items section"). */
const GRID_PERIOD_KEY = 'mapp.lineItemGrid';

export interface SpreadGridProps {
  grid: Grid;
  /** The activity has no dates yet: its line items are not spread (D-032). */
  noDates: boolean;
}

/**
 * The activity's line items by month, in USD (or the display currency), with a totals row (FR-003,
 * FR-020): Monthly or Quarterly (display only, remembered per browser), months outside the
 * activity's dates marked, the "No dates yet" column, and the line-item names kept in view while
 * the months scroll sideways inside the card.
 */
export function SpreadGrid({ grid, noDates }: SpreadGridProps) {
  const [period, setPeriod] = useState<Period>(readPeriod);
  const shown = period === 'quarterly' ? quarterColumns(grid) : monthColumns(grid);
  const code = grid.total.currency;

  function choose(next: Period) {
    setPeriod(next);
    try {
      localStorage.setItem(GRID_PERIOD_KEY, next);
    } catch {
      // Storage may be blocked: the choice then lasts until the page is left.
    }
  }

  return (
    <div className={styles.grid}>
      <div className={styles.head}>
        <h3 className={styles.title}>{GRID_TITLE}</h3>
        <SegmentedFilter
          label="Show the spread by"
          options={[
            { value: 'monthly', label: MONTHLY },
            { value: 'quarterly', label: QUARTERLY },
          ]}
          value={period}
          onChange={choose}
        />
      </div>
      {noDates && <p className={styles.note}>{GRID_NO_DATES}</p>}
      <div className={styles.scroll} role="region" aria-label={gridCaption(code)} tabIndex={0}>
        <table className={styles.table}>
          <caption className="visually-hidden">{gridCaption(code)}</caption>
          <thead>
            <tr>
              <th scope="col" className={styles.sticky}>
                {LINE_ITEM_COLUMN}
              </th>
              {shown.columns.map((column) => (
                <th key={column.key} scope="col" className={styles.number}>
                  <span className={styles.period}>{column.label}</span>
                  {column.outsideActivityDates && (
                    <span className={styles.outside}>
                      {OUTSIDE_DATES}
                      <InfoTip label={`${column.label} outside the dates`} text={OUTSIDE_TIP} />
                    </span>
                  )}
                </th>
              ))}
              <th scope="col" className={styles.number}>
                {TOTAL}
              </th>
            </tr>
          </thead>
          <tbody>
            {shown.rows.map((row) => (
              <tr key={row.lineItemId}>
                <th scope="row" className={styles.sticky}>
                  {row.name}
                </th>
                {row.cells.map((cell, index) => (
                  <td key={shown.columns[index].key} className={styles.number}>
                    {amount(cell)}
                  </td>
                ))}
                <td className={styles.number}>{amount(row.total)}</td>
              </tr>
            ))}
          </tbody>
          <tfoot>
            <tr>
              <th scope="row" className={styles.sticky}>
                {TOTAL}
              </th>
              {shown.totals.map((cell, index) => (
                <td key={shown.columns[index].key} className={cx(styles.number, styles.strong)}>
                  {amount(cell)}
                </td>
              ))}
              <td className={cx(styles.number, styles.strong)}>{amount(shown.total)}</td>
            </tr>
          </tfoot>
        </table>
      </div>
    </div>
  );
}

function amount(money: Money | null): string {
  return money ? formatAmount(money.amount) : EMPTY;
}

/** The remembered choice; Monthly when there is none or storage cannot be read. */
function readPeriod(): Period {
  try {
    return localStorage.getItem(GRID_PERIOD_KEY) === 'quarterly' ? 'quarterly' : 'monthly';
  } catch {
    return 'monthly';
  }
}
