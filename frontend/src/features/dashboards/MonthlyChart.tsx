import { useState } from 'react';
import { toCents } from '@/format/decimal';
import { useFormatters } from '@/format/useFormatters';
import { Button } from '@/ui/Button/Button';
import { Card } from '@/ui/Card/Card';
import type { DashboardMonths, Money } from './api';
import { FINANCE_HIDDEN, monthLabel } from './dashboard-text';
import styles from './Dashboards.module.css';

const WIDTH = 720;
const HEIGHT = 240;
const BASELINE = 210;
const PLOT = 190;

interface MonthlyChartProps {
  months: DashboardMonths;
}

export function MonthlyChart({ months }: MonthlyChartProps) {
  const [view, setView] = useState<'chart' | 'table'>('chart');
  const { money } = useFormatters();
  const showsActual = months.items.some((month) => month.actual !== null);
  const notes = [
    months.noDates && `Not yet spread (activities without dates): ${money(months.noDates)}`,
    months.otherYears && `Spread into other years: ${money(months.otherYears)}`,
    months.paidOutsideYear && `Paid outside the year: ${money(months.paidOutsideYear)}`,
    !showsActual && `Actual: ${FINANCE_HIDDEN}`,
  ].filter((note) => typeof note === 'string');
  const switcher = (
    <div className={styles.switch} role="group" aria-label="Show as">
      {(['chart', 'table'] as const).map((choice) => (
        <Button
          key={choice}
          size="sm"
          variant={view === choice ? 'secondary' : 'tertiary'}
          aria-pressed={view === choice}
          onClick={() => setView(choice)}
        >
          {choice === 'chart' ? 'Chart' : 'Table'}
        </Button>
      ))}
    </div>
  );
  return (
    <Card title="Planned and actual by month" actions={switcher}>
      {view === 'chart' ? <Bars months={months} /> : <MonthTable months={months} />}
      {notes.map((note) => (
        <p key={note} className={styles.note}>
          {note}
        </p>
      ))}
    </Card>
  );
}

function cents(value: Money | null): bigint {
  return value ? toCents(value.amount) : 0n;
}

function height(value: bigint, top: bigint): number {
  return top > 0n && value > 0n ? Number((value * BigInt(PLOT)) / top) : 0;
}

function Bars({ months }: MonthlyChartProps) {
  const { money } = useFormatters();
  const top = months.items.reduce(
    (largest, m) => [cents(m.planned), cents(m.actual), largest].reduce((a, b) => (a > b ? a : b)),
    0n,
  );
  const group = WIDTH / Math.max(months.items.length, 1);
  const last = months.items.at(-1);
  const summary = `Planned cost by month; ${money(last?.plannedToDate ?? null)} planned for the year and ${money(last?.actualToDate ?? null)} paid so far. The Table view lists every month.`;
  return (
    <>
      <ul className={styles.legend}>
        <li>
          <span className={`${styles.swatch} ${styles.planned}`} aria-hidden="true" />
          Planned cost
        </li>
        <li>
          <span className={`${styles.swatch} ${styles.actual}`} aria-hidden="true" />
          Actual (paid)
        </li>
      </ul>
      <svg
        className={styles.chart}
        viewBox={`0 0 ${WIDTH} ${HEIGHT}`}
        role="img"
        aria-label={summary}
      >
        {months.items.map((m, i) => {
          const planned = height(cents(m.planned), top);
          const actual = height(cents(m.actual), top);
          const x = i * group + group * 0.15;
          const bar = group * 0.35;
          return (
            <g key={m.month}>
              <rect
                className={styles.planned}
                x={x}
                y={BASELINE - planned}
                width={bar}
                height={planned}
              />
              <rect
                className={styles.actual}
                x={x + bar}
                y={BASELINE - actual}
                width={bar}
                height={actual}
              />
              <text className={styles.axisLabel} x={x + bar} y={HEIGHT - 12} textAnchor="middle">
                {monthLabel(m.month).slice(0, 3)}
              </text>
            </g>
          );
        })}
        <line className={styles.axis} x1={0} x2={WIDTH} y1={BASELINE} y2={BASELINE} />
      </svg>
    </>
  );
}

function MonthTable({ months }: MonthlyChartProps) {
  const { money } = useFormatters();
  return (
    <div className={styles.scroll}>
      <table className={styles.table}>
        <caption>
          Planned cost spread by month and actual paid by posting month, with running totals
        </caption>
        <thead>
          <tr>
            {['Month', 'Planned cost', 'Actual', 'Planned to date', 'Actual to date'].map(
              (title, i) => (
                <th key={title} scope="col" className={i > 0 ? styles.number : undefined}>
                  {title}
                </th>
              ),
            )}
          </tr>
        </thead>
        <tbody>
          {months.items.map((m) => (
            <tr key={m.month}>
              <th scope="row">{monthLabel(m.month)}</th>
              <td className={styles.number}>{money(m.planned)}</td>
              <td className={styles.number}>{money(m.actual)}</td>
              <td className={styles.number}>{money(m.plannedToDate)}</td>
              <td className={styles.number}>{money(m.actualToDate)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
