import { Plus } from 'lucide-react';
import { useId } from 'react';
import { Button } from '@/ui/Button/Button';
import { problemOf } from '@/ui/FormPanel/problemOf';
import { MoneyField } from '@/ui/MoneyField/MoneyField';
import type { FieldProblem } from '@/ui/FormErrorSummary/FormErrorSummary';
import { cx } from '@/lib/cx';
import { monthFieldId } from './lineItemFormState';
import type { CustomMonth } from './lineItemFormState';
import {
  addMonths,
  amountText,
  centsInput,
  formatMonth,
  isOutsideDates,
  parseAmount,
  spreadMonths,
} from './lineItemMath';
import type { ActivityDates, SpreadMethod } from './lineItemMath';
import {
  ADD_MONTH_AFTER,
  ADD_MONTH_BEFORE,
  customTotalText,
  LABEL,
  LEAVING_CUSTOM,
  monthAmountLabel,
  NOT_SPREAD_YET,
  ONGOING_EVENLY,
  ONGOING_LAST,
  OUTSIDE_TIP,
  previewLine,
  SPREAD_LABELS,
  USE_MONTH_TOTAL,
} from './lineItemText';
import styles from './SpreadEditor.module.css';

const METHODS: SpreadMethod[] = ['evenly', 'firstMonth', 'lastMonth', 'custom'];

export interface SpreadEditorProps {
  method: SpreadMethod;
  /** The planned cost as typed. */
  plannedCost: string;
  currency: string;
  dates: ActivityDates;
  customMonths: CustomMonth[];
  /** Custom amounts were typed or saved: leaving Custom says they will be replaced. */
  customTyped: boolean;
  /** Where the method came from ("The plan's default"), until it is changed. */
  source?: string;
  problems: FieldProblem[];
  onMethodChange: (method: SpreadMethod) => void;
  onMonthChange: (month: string, amount: string) => void;
  onAddMonth: (month: string) => void;
  /** Sets the planned cost to the month total (FR-018). */
  onUseTotal: (amount: string) => void;
}

/**
 * Spread over time (FR-017 – FR-021): Evenly, All in first month, All in last month or Custom. The
 * first three show a read-only preview of their months; Custom shows a month grid with Add month
 * before / after, months outside the activity's dates marked as allowed, and the running total.
 */
export function SpreadEditor({
  method,
  plannedCost,
  currency,
  dates,
  customMonths,
  customTyped,
  source,
  problems,
  onMethodChange,
  onMonthChange,
  onAddMonth,
  onUseTotal,
}: SpreadEditorProps) {
  const groupId = useId();
  const cost = parseAmount(plannedCost) ?? 0n;

  return (
    <div className={styles.editor}>
      <fieldset id="spread" tabIndex={-1} className={styles.methods}>
        <legend className="visually-hidden">{LABEL.spread}</legend>
        <div className={styles.radios}>
          {METHODS.map((option) => (
            <label key={option} className={styles.radio}>
              <input
                type="radio"
                name={`${groupId}-spread`}
                value={option}
                checked={method === option}
                onChange={() => onMethodChange(option)}
              />
              {SPREAD_LABELS[option]}
            </label>
          ))}
        </div>
        {source && <p className={styles.note}>{source}</p>}
        {problemOf(problems, 'spread') && (
          <p className={styles.error}>{problemOf(problems, 'spread')}</p>
        )}
      </fieldset>

      {method !== 'custom' && customTyped && <p className={styles.note}>{LEAVING_CUSTOM}</p>}

      {method === 'custom' ? (
        <CustomGrid
          plannedCost={cost}
          currency={currency}
          dates={dates}
          months={customMonths}
          problems={problems}
          onMonthChange={onMonthChange}
          onAddMonth={onAddMonth}
          onUseTotal={onUseTotal}
        />
      ) : (
        <Preview method={method} cost={cost} currency={currency} dates={dates} />
      )}
    </div>
  );
}

function Preview({
  method,
  cost,
  currency,
  dates,
}: {
  method: Exclude<SpreadMethod, 'custom'>;
  cost: bigint;
  currency: string;
  dates: ActivityDates;
}) {
  const spread = spreadMonths(method, cost, dates);
  if (spread.state === 'notSpread') {
    return <p className={styles.note}>{NOT_SPREAD_YET}</p>;
  }
  const ongoingNote =
    dates.ongoing && method === 'evenly'
      ? ONGOING_EVENLY
      : dates.ongoing && method === 'lastMonth'
        ? ONGOING_LAST
        : null;
  return (
    <>
      {ongoingNote && <p className={styles.note}>{ongoingNote}</p>}
      <ul className={styles.preview} aria-label="Preview of the spread">
        {spread.months.map(({ month, cents }) => (
          <li key={month}>{previewLine(formatMonth(month), amountText(cents, currency))}</li>
        ))}
      </ul>
    </>
  );
}

interface CustomGridProps {
  plannedCost: bigint;
  currency: string;
  dates: ActivityDates;
  months: CustomMonth[];
  problems: FieldProblem[];
  onMonthChange: (month: string, amount: string) => void;
  onAddMonth: (month: string) => void;
  onUseTotal: (amount: string) => void;
}

function CustomGrid({
  plannedCost,
  currency,
  dates,
  months,
  problems,
  onMonthChange,
  onAddMonth,
  onUseTotal,
}: CustomGridProps) {
  const sum = months.reduce((total, month) => total + (parseAmount(month.amount) ?? 0n), 0n);
  const difference = plannedCost - sum;
  const first = months[0]?.month;
  const last = months[months.length - 1]?.month;

  return (
    <div className={styles.custom}>
      <div className={styles.monthGrid}>
        {months.map(({ month, amount }) => {
          const outside = isOutsideDates(month, dates);
          return (
            <div key={month} className={cx(styles.month, outside && styles.outside)}>
              <span className={styles.monthName} aria-hidden="true">
                {formatMonth(month)}
              </span>
              <MoneyField
                label={monthAmountLabel(formatMonth(month))}
                labelHidden
                name={monthFieldId(month)}
                currency={currency}
                value={amount}
                onChange={(value) => onMonthChange(month, value)}
                error={problemOf(problems, monthFieldId(month))}
                note={outside ? OUTSIDE_TIP : undefined}
              />
            </div>
          );
        })}
      </div>
      <div className={styles.addMonths}>
        {first && (
          <Button size="sm" icon={<Plus />} onClick={() => onAddMonth(addMonths(first, -1))}>
            {ADD_MONTH_BEFORE}
          </Button>
        )}
        {last && (
          <Button size="sm" icon={<Plus />} onClick={() => onAddMonth(addMonths(last, 1))}>
            {ADD_MONTH_AFTER}
          </Button>
        )}
      </div>
      <p
        className={cx(styles.total, difference < 0n && styles.tooMuch)}
        role="status"
        aria-live="polite"
      >
        {customTotalText(centsInput(sum), centsInput(plannedCost), currency, {
          sign: difference === 0n ? 0 : difference > 0n ? 1 : -1,
          amount: centsInput(difference < 0n ? -difference : difference),
        })}
      </p>
      {difference !== 0n && (
        <Button size="sm" onClick={() => onUseTotal(centsInput(sum))}>
          {USE_MONTH_TOTAL}
        </Button>
      )}
    </div>
  );
}
