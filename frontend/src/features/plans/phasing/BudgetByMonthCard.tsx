import { TriangleAlert } from 'lucide-react';
import { useEffect, useId, useRef, useState } from 'react';
import { useLocation } from 'react-router';
import { Button } from '@/ui/Button/Button';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { SegmentedFilter } from '@/ui/SegmentedFilter/SegmentedFilter';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { formatAmount, toCents } from '@/format/decimal';
import styles from './BudgetByMonth.module.css';
import { useBudgetByMonth } from './phasingApi';
import type { BudgetByMonthPeriods, BudgetPhasing, PhasingLevel } from './phasingApi';
import { amountText, MONTH_NAMES, overBy } from './phasingMath';
import { PhasingEditor } from './PhasingEditor';
import { phasingAnchor } from './phasingLinks';

type Money = { amount: string; currency: string };

export interface BudgetByMonthCardProps {
  level: PhasingLevel;
  planId: string;
  /** The record whose budget is phased; `budgetPhasing` is 016's (absent before 016's API). */
  record: {
    id: string;
    name: string;
    version: number;
    budget: Money | null;
    budgetPhasing?: BudgetPhasing;
  };
  /** The plan's year: its 12 months. */
  year: number;
  /** The plan currency: every amount of the card is in it (016 FR-013). */
  currency: string;
  /** 'page' on the plan page (a card, h2), 'panel' in a campaign or initiative panel (h3). */
  variant?: 'page' | 'panel';
}

const CHILDREN: Record<PhasingLevel, string | null> = {
  plan: 'campaigns',
  campaign: 'initiatives',
  initiative: null,
};

/**
 * **Budget by month** (016 FR-024): the record's phased budget, its planned cost by month, the
 * difference and — for plans and campaigns — what their children's phased budgets take, Monthly or
 * Quarterly, with Other years and No dates when they hold amounts. A period whose planned cost or
 * children go above its budget is red with "Over by …"; nothing here blocks. **Edit** opens the
 * editor (FR-023). Shown only by an API that phases budgets (016).
 */
export function BudgetByMonthCard({
  level,
  planId,
  record,
  year,
  currency,
  variant = 'page',
}: BudgetByMonthCardProps) {
  const headingId = useId();
  const phasing = record.budgetPhasing;
  const byMonth = useBudgetByMonth(planId, phasing !== undefined);
  const [view, setView] = useState<'months' | 'quarters'>(
    variant === 'panel' ? 'quarters' : 'months',
  );
  const [editing, setEditing] = useState(false);
  const section = useRef<HTMLElement>(null);
  const anchor = phasingAnchor(record.id);
  // A completeness link ("Budget by month" needed to publish) leads here with #budget-by-month-…:
  // the card comes into view and takes the focus, after the page or panel has placed its own.
  const arrived = useLocation().hash === `#${anchor}` && phasing !== undefined;
  useEffect(() => {
    if (!arrived) {
      return;
    }
    const timer = window.setTimeout(() => {
      section.current?.scrollIntoView?.({ block: 'start' });
      section.current?.focus({ preventScroll: true });
    }, 0);
    return () => window.clearTimeout(timer);
  }, [arrived]);
  if (!phasing) {
    return null;
  }
  const Heading = variant === 'page' ? 'h2' : 'h3';
  const row = byMonth.data?.rows.find((candidate) => candidate.recordId === record.id);
  // The API names the plan currency of every amount; the record's own until it has answered.
  const amountsIn = byMonth.data?.currency ?? currency;
  const children = CHILDREN[level];

  function body() {
    if (byMonth.isError) {
      return <ErrorState error={byMonth.error} onRetry={() => void byMonth.refetch()} />;
    }
    if (!row) {
      return <Skeleton lines={3} />;
    }
    const periods = view === 'months' ? row.months : row.quarters;
    const labels = view === 'months' ? MONTH_NAMES.map((name) => name) : ['Q1', 'Q2', 'Q3', 'Q4'];
    const extra = [
      ...(row.otherYears ? [{ label: 'Other years', planned: row.otherYears }] : []),
      ...(row.noDates ? [{ label: 'No dates', planned: row.noDates }] : []),
    ];
    return (
      <div className={styles.scroll} tabIndex={0} aria-label="Budget by month, scrolls sideways">
        <table className={styles.table}>
          <caption className="visually-hidden">
            Budget by {view === 'months' ? 'month' : 'quarter'} of {record.name} in {amountsIn},{' '}
            {year}
          </caption>
          <thead>
            <tr>
              <th scope="col">
                <span className="visually-hidden">Row</span>
              </th>
              {labels.map((label) => (
                <th key={label} scope="col" className={styles.number}>
                  {label}
                </th>
              ))}
              {extra.map((column) => (
                <th key={column.label} scope="col" className={styles.number}>
                  {column.label}
                </th>
              ))}
              <th scope="col" className={styles.number}>
                Total
              </th>
            </tr>
          </thead>
          <tbody>
            <PeriodRow
              label="Budget"
              values={periods.budget}
              extraCount={extra.length}
              empty={row.method === 'none' ? 'Not phased' : undefined}
            />
            <PeriodRow
              label="Planned cost"
              values={periods.plannedCost}
              extra={extra.map((column) => column.planned)}
            />
            <DifferenceRow periods={periods} extraCount={extra.length} />
            {children && periods.allocated && (
              <PeriodRow
                label={`Allocated to ${children}`}
                values={periods.allocated}
                extraCount={extra.length}
              />
            )}
          </tbody>
        </table>
      </div>
    );
  }

  const notPhased = row && children && row.childrenNotPhased > 0 ? row.childrenNotPhased : 0;

  return (
    <section
      ref={section}
      aria-labelledby={headingId}
      className={variant === 'page' ? styles.card : styles.section}
      id={anchor}
      tabIndex={-1}
    >
      <div className={styles.header}>
        <div>
          <Heading id={headingId} className={styles.title}>
            Budget by month
          </Heading>
          <p className={styles.caption}>
            Amounts in {amountsIn}, the plan currency · {year}
          </p>
        </div>
        <div className={styles.actions}>
          <SegmentedFilter
            label="Show by"
            options={[
              { value: 'months', label: 'Monthly' },
              { value: 'quarters', label: 'Quarterly' },
            ]}
            value={view}
            onChange={setView}
          />
          {record.budget && (
            <Button size="sm" aria-label="Edit budget by month" onClick={() => setEditing(true)}>
              Edit
            </Button>
          )}
        </div>
      </div>
      {phasing.problem && (
        <p className={styles.problem}>
          <TriangleAlert aria-hidden="true" />
          <span>Budget by month: {phasing.problem}. Needed to publish.</span>
        </p>
      )}
      {!record.budget && <p className={styles.note}>Set a budget before phasing it.</p>}
      {body()}
      {notPhased > 0 && children && (
        <p className={styles.note}>
          {notPhased === 1
            ? `1 ${children.slice(0, -1)} is not phased`
            : `${notPhased} ${children} are not phased`}
        </p>
      )}
      {editing && row && (
        <PhasingEditor
          open
          onOpenChange={setEditing}
          level={level}
          record={record}
          phasing={phasing}
          year={year}
          currency={amountsIn}
          plannedByMonth={row.months.plannedCost}
          leftOut={{ otherYears: row.otherYears, noDates: row.noDates }}
        />
      )}
    </section>
  );
}

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

function PeriodRow({
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
function DifferenceRow({
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
