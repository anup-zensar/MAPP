import { Card } from '@/ui/Card/Card';
import { InfoTip } from '@/ui/InfoTip/InfoTip';
import { SegmentedFilter } from '@/ui/SegmentedFilter/SegmentedFilter';
import type { Money, SpaContribution, SpaPeriod, SpaView } from './api';
import { Bar } from './Bar';
import { FigureButton } from './FigureButton';
import type { TraceRequest } from './FigureButton';
import { PERIODS } from './spa-text';
import styles from './Spa.module.css';

interface SpaPeriodsProps {
  view: SpaView;
  period: 'quarter' | 'month';
  onPeriod: (period: 'quarter' | 'month') => void;
  onTrace: (request: TraceRequest) => void;
}

/** Planned against actual spend by quarter or month of the plan's year (013 FR-017). */
export function SpaPeriods({ view, period, onPeriod, onTrace }: SpaPeriodsProps) {
  const rows = period === 'month' ? view.periods.months : view.periods.quarters;
  return (
    <Card
      title={PERIODS.title}
      actions={
        <SegmentedFilter
          label="Periods"
          value={period}
          onChange={onPeriod}
          options={[
            { value: 'quarter', label: 'Quarters' },
            { value: 'month', label: 'Months' },
          ]}
        />
      }
    >
      <p className={styles.note}>{PERIODS.tip}</p>
      <table className={styles.table}>
        <caption className="visually-hidden">
          {PERIODS.title} by {period}, {view.plan.year}
        </caption>
        <thead>
          <tr>
            <th scope="col">Period</th>
            <th scope="col">{PERIODS.planned}</th>
            <th scope="col">{PERIODS.actual}</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <PeriodRow key={row.key} row={row} onTrace={onTrace} />
          ))}
          <ExtraRow
            label={PERIODS.notScheduled}
            tip={PERIODS.notScheduledTip}
            value={view.periods.notScheduled}
            trace={view.periods.notScheduledTrace}
            onTrace={onTrace}
          />
          <ExtraRow
            label={PERIODS.linesAbove}
            tip={PERIODS.linesAboveTip}
            value={view.periods.linesAbovePlannedCost}
            trace={view.periods.linesAbovePlannedCostTrace}
            onTrace={onTrace}
          />
        </tbody>
      </table>
    </Card>
  );
}

function PeriodRow({ row, onTrace }: { row: SpaPeriod; onTrace: (request: TraceRequest) => void }) {
  return (
    <tr>
      <th scope="row">{row.label}</th>
      <td>
        <div className={styles.barCell}>
          <FigureButton
            figure={`${row.label} planned`}
            value={row.planned}
            trace={row.plannedTrace}
            onTrace={onTrace}
          />
          <Bar length={row.plannedBar} series="planned" />
        </div>
      </td>
      <td>
        <div className={styles.barCell}>
          <FigureButton
            figure={`${row.label} actual spend`}
            value={row.actual}
            trace={row.actualTrace}
            onTrace={onTrace}
          />
          <Bar length={row.actualBar} series="actual" />
        </div>
      </td>
    </tr>
  );
}

interface ExtraRowProps {
  label: string;
  tip: string;
  value: Money | null;
  trace: SpaContribution[];
  onTrace: (request: TraceRequest) => void;
}

/** Not yet scheduled, or line items above planned cost (038 FR-023): shown only when not zero. */
function ExtraRow({ label, tip, value, trace, onTrace }: ExtraRowProps) {
  if (!value) {
    return null;
  }
  return (
    <tr>
      <th scope="row">
        {label}
        <InfoTip label={label} text={tip} />
      </th>
      <td>
        <FigureButton figure={label} value={value} trace={trace} onTrace={onTrace} />
      </td>
      <td />
    </tr>
  );
}
