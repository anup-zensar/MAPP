import { TriangleAlert } from 'lucide-react';
import { useEffect, useId, useRef, useState } from 'react';
import { useLocation } from 'react-router';
import { FieldHelpTip } from '@/features/settings';
import type { Availability } from '@/session/access';
import { allows } from '@/session/gate-action';
import { Button } from '@/ui/Button/Button';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { SegmentedFilter } from '@/ui/SegmentedFilter/SegmentedFilter';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import styles from './BudgetByMonth.module.css';
import { useBudgetByMonth } from './api';
import type { BudgetPhasing, PhasingLevel } from './api';
import { MONTH_NAMES } from './phasing-math';
import { PhasingEditor } from './PhasingEditor';
import { phasingAnchor } from './phasing-links';
import { DifferenceRow, PeriodRow } from './PeriodRows';

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
  /** Whether the reader may edit the record: Edit is offered only then (038 FR-029). */
  access?: Availability;
  /** 'page' on the plan page (a card, h2), 'panel' in a campaign or initiative panel (h3). */
  variant?: 'page' | 'panel';
}

const CHILDREN: Record<PhasingLevel, string | null> = {
  plan: 'campaigns',
  campaign: 'initiatives',
  initiative: null,
};

/**
 * The record's phased budget, planned cost, difference and children's phased budgets by month or
 * quarter; a period over its budget is red with "Over by …", nothing blocks. **Edit** opens the editor.
 */
export function BudgetByMonthCard({
  level,
  planId,
  record,
  year,
  currency,
  access,
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
      <div
        className={styles.scroll}
        role="region"
        aria-label="Budget by month, scrolls sideways"
        tabIndex={0}
      >
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
              helpKey="phasing.budget"
              values={periods.budget}
              extraCount={extra.length}
              empty={row.method === 'none' ? 'Not phased' : undefined}
            />
            <PeriodRow
              label="Planned cost"
              helpKey="phasing.plannedCost"
              values={periods.plannedCost}
              extra={extra.map((column) => column.planned)}
            />
            <DifferenceRow periods={periods} extraCount={extra.length} />
            {children && periods.allocated && (
              <PeriodRow
                label={`Allocated to ${children}`}
                helpKey="phasing.allocated"
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
          <div className={styles.titleRow}>
            <Heading id={headingId} className={styles.title}>
              Budget by month
            </Heading>
            <FieldHelpTip helpKey={`${level}.budgetPhasing`} label="Budget by month" />
          </div>
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
          {record.budget && allows(access) && (
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
      {row && row.negativeBalanceCount > 0 && (
        <p className={styles.note}>{aboveCostNote(row.negativeBalanceCount)}</p>
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

/** Why the months add up to more than the planned cost (038 FR-024). */
function aboveCostNote(count: number): string {
  return count === 1
    ? '1 activity has line items above its planned cost, so its months add up to more than its planned cost.'
    : `${count} activities have line items above their planned cost, so their months add up to more than their planned cost.`;
}
