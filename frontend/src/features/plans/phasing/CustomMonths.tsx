import { useWatch } from 'react-hook-form';
import type { UseFormReturn } from 'react-hook-form';
import { Button } from '@/ui/Button/Button';
import { FormField } from '@/ui/FormField/FormField';
import { MoneyField } from '@/ui/MoneyField/MoneyField';
import { SegmentedFilter } from '@/ui/SegmentedFilter/SegmentedFilter';
import { formatMoneyText, toCents } from '@/format/decimal';
import styles from './BudgetByMonth.module.css';
import {
  amountText,
  monthKeys,
  monthLabel,
  quarterLabels,
  quarterTotals,
  readAmount,
  splitEvenly,
  withQuarter,
} from './phasing-math';
import { MONTH_PROBLEM } from './schemas';
import type { PhasingValues } from './schemas';

type Money = { amount: string; currency: string };

export type Entry = 'monthly' | 'quarterly';

export interface CustomMonthsProps {
  form: UseFormReturn<PhasingValues>;
  year: number;
  currency: string;
  /** The record's budget in cents; null without one. */
  budget: bigint | null;
  /** The record's planned cost of each month, for Copy from planned cost (FR-021). */
  plannedByMonth: (Money | null)[];
  /** Planned cost Copy from planned cost leaves out: in other years, and without dates. */
  leftOut: { otherYears: Money | null; noDates: Money | null };
  entry: Entry;
  onEntryChange: (entry: Entry) => void;
  /** What Copy from planned cost left out; null for nothing. */
  copiedNote: string | null;
  onCopied: (note: string | null) => void;
}

/**
 * Custom phasing (016 FR-021): the months typed Monthly or Quarterly (a quarter spreads over its
 * months by the Evenly rule), with Split evenly and Copy from planned cost.
 */
export function CustomMonths({
  form,
  year,
  currency,
  budget,
  plannedByMonth,
  leftOut,
  entry,
  onEntryChange,
  copiedNote,
  onCopied,
}: CustomMonthsProps) {
  const keys = monthKeys(year);
  const months = useWatch({ control: form.control, name: 'months' });
  const quarters = useWatch({ control: form.control, name: 'quarters' });
  const cents = months.map((text) => readAmount(text) ?? 0n);
  const totals = quarterTotals(cents);

  function fill(next: bigint[]) {
    form.setValue(
      'months',
      next.map((amount) => amountText(amount)),
      { shouldDirty: true },
    );
    form.setValue('quarters', null);
  }

  function typeQuarter(quarter: number, text: string) {
    const typed = quarters ?? totals.map((amount) => amountText(amount));
    form.setValue(
      'quarters',
      typed.map((value, index) => (index === quarter ? text : value)),
    );
    const amount = readAmount(text);
    if (amount !== null) {
      form.setValue(
        'months',
        withQuarter(cents, quarter, amount).map((value) => amountText(value)),
        { shouldDirty: true },
      );
    }
  }

  function copyPlannedCost() {
    fill(plannedByMonth.map((money) => (money ? toCents(money.amount) : 0n)));
    const parts = [
      leftOut.otherYears && `${formatMoneyText(leftOut.otherYears)} in other years`,
      leftOut.noDates && `${formatMoneyText(leftOut.noDates)} without dates`,
    ].filter(Boolean);
    onCopied(parts.length > 0 ? `Left out: ${parts.join(' and ')}.` : null);
  }

  return (
    <>
      <div className={styles.tools}>
        <SegmentedFilter
          label="Type amounts"
          options={[
            { value: 'monthly', label: 'Monthly' },
            { value: 'quarterly', label: 'Quarterly' },
          ]}
          value={entry}
          onChange={(next) => {
            onEntryChange(next);
            form.setValue('quarters', null);
          }}
        />
        {budget !== null && (
          <Button size="sm" onClick={() => fill(splitEvenly(budget, 12))}>
            Split evenly
          </Button>
        )}
        <Button size="sm" onClick={copyPlannedCost}>
          Copy from planned cost
        </Button>
      </div>
      {copiedNote && <p className={styles.note}>{copiedNote}</p>}
      <div className={styles.grid}>
        {entry === 'monthly'
          ? keys.map((key, index) => (
              <FormField
                key={key}
                control={form.control}
                name={`months.${index}`}
                render={(field) => (
                  <MoneyField
                    label={monthLabel(key)}
                    currency={currency}
                    {...field}
                    name={`month-${key}`}
                    error={readAmount(field.value) === null ? MONTH_PROBLEM : undefined}
                  />
                )}
              />
            ))
          : quarterLabels(year).map((label, quarter) => (
              <MoneyField
                key={label}
                label={label}
                name={`quarter-${quarter + 1}`}
                currency={currency}
                value={quarters?.[quarter] ?? amountText(totals[quarter] ?? 0n)}
                onChange={(text) => typeQuarter(quarter, text)}
                note="Spread evenly over its three months"
              />
            ))}
      </div>
    </>
  );
}
