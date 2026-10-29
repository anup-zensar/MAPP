import { zodResolver } from '@hookform/resolvers/zod';
import { useState } from 'react';
import { useForm, useWatch } from 'react-hook-form';
import { ApiError, conflictMessage } from '@/api/errors';
import { FormDialog } from '@/ui/FormDialog/FormDialog';
import { FormField } from '@/ui/FormField/FormField';
import { FailureMessage } from '@/ui/FormPanel/FormPanel';
import { useToast } from '@/ui/Toast/useToast';
import type { SaveFailure } from '@/ui/FormPanel/problem-of';
import { CHECK_ON_SAVE } from '@/ui/useFormProblems';
import { formatCents, toCents } from '@/format/decimal';
import { useFormatters } from '@/format/useFormatters';
import { useSetPhasing } from './api';
import type { BudgetPhasing, PhasingLevel, PhasingMethod } from './api';
import styles from './BudgetByMonth.module.css';
import { CustomMonths } from './CustomMonths';
import type { Entry } from './CustomMonths';
import {
  amountText,
  balanceOf,
  balanceText,
  monthKeys,
  readAmount,
  splitEvenly,
} from './phasing-math';
import { monthProblems, namedProblems, phasingSchema } from './schemas';
import type { PhasingValues } from './schemas';

type Money = { amount: string; currency: string };

export interface PhasingEditorProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  level: PhasingLevel;
  record: { id: string; name: string; version: number; budget: Money | null };
  phasing: BudgetPhasing;
  /** The plan's year: the 12 months that can be phased. */
  year: number;
  /** The plan currency (016 FR-011): every amount here is typed in it. */
  currency: string;
  /** The record's planned cost of each month, for Copy from planned cost (FR-021). */
  plannedByMonth: (Money | null)[];
  /** Planned cost Copy from planned cost leaves out: in other years, and without dates. */
  leftOut: { otherYears: Money | null; noDates: Money | null };
}

const METHODS: { value: PhasingMethod; label: string; text: string }[] = [
  { value: 'none', label: 'Not phased', text: 'The budget is for the whole year.' },
  {
    value: 'evenly',
    label: 'Evenly',
    text: 'The budget over the 12 months; it follows the budget when it changes.',
  },
  { value: 'custom', label: 'Custom', text: 'An amount for each month.' },
];

/**
 * The Budget by month editor (016 FR-020 – FR-023): Not phased, Evenly or Custom; Custom months
 * typed Monthly or Quarterly (a quarter spreads over its months by the Evenly rule), with Split
 * evenly, Copy from planned cost and a running total. Saving never changes the record's status;
 * a Custom phasing that does not add up still saves and shows as a problem to publish (FR-022).
 */
export function PhasingEditor({
  open,
  onOpenChange,
  level,
  record,
  phasing,
  year,
  currency,
  plannedByMonth,
  leftOut,
}: PhasingEditorProps) {
  const { show } = useToast();
  const { dateFormat } = useFormatters();
  const save = useSetPhasing(level, record.id);
  const keys = monthKeys(year);
  const budget = record.budget ? toCents(record.budget.amount) : null;
  const form = useForm<PhasingValues>({
    resolver: zodResolver(phasingSchema),
    defaultValues: {
      method: phasing.method,
      months: keys.map((_, index) => {
        const amount = phasing.monthly[index]?.amount;
        return amount ? amountText(toCents(amount)) : '';
      }),
      quarters: null,
    },
    ...CHECK_ON_SAVE,
  });
  const values = useWatch({ control: form.control, compute: (all: PhasingValues) => all });
  const [entry, setEntry] = useState<Entry>('monthly');
  const [copiedNote, setCopiedNote] = useState<string | null>(null);
  const [failure, setFailure] = useState<SaveFailure | null>(null);
  const cents = values.months.map((text) => readAmount(text) ?? 0n);
  const evenly = budget === null ? [] : splitEvenly(budget, 12);

  async function submit({ method, months }: PhasingValues) {
    const amounts = months.map((text) => readAmount(text) ?? 0n);
    const sent =
      method === 'custom'
        ? keys.flatMap((month, index) => {
            const amount = amounts[index] ?? 0n;
            return amount === 0n ? [] : [{ month, amount: formatCents(amount) }];
          })
        : [];
    try {
      await save.mutateAsync({ method, months: sent, version: record.version });
      show({ message: `Budget by month of '${record.name}' saved` });
      onOpenChange(false);
    } catch (error) {
      const apiError = error instanceof ApiError ? error : new ApiError('other');
      if (apiError.kind === 'conflict') {
        setFailure({
          message: conflictMessage(apiError, level, dateFormat),
          nextStep: 'Close this and open Budget by month again to see their change.',
        });
        return;
      }
      setFailure({
        message:
          apiError.kind === 'validation' && apiError.fields.length > 0
            ? namedProblems(apiError.fields, sent)
            : apiError.message,
      });
    }
  }

  const balance = balanceOf(values.method === 'evenly' ? evenly : cents, budget);
  const first = evenly[0] ?? 0n;
  const december = evenly[11] ?? 0n;

  return (
    <FormDialog
      open={open}
      onOpenChange={onOpenChange}
      title={`Budget by month — ${record.name}`}
      submitLabel="Save budget by month"
      onSubmit={() =>
        void form.handleSubmit(submit, () =>
          setFailure({ message: monthProblems(form.getValues('months'), keys) }),
        )()
      }
      busy={save.isPending}
    >
      <div className={styles.editor}>
        {failure && <FailureMessage failure={failure} />}
        <FormField
          control={form.control}
          name="method"
          render={(field) => (
            <fieldset className={styles.methods}>
              <legend className={styles.legend}>How to phase the budget</legend>
              {METHODS.map((choice) => (
                <label key={choice.value} className={styles.method}>
                  <input
                    type="radio"
                    name="method"
                    value={choice.value}
                    checked={field.value === choice.value}
                    onChange={() => field.onChange(choice.value)}
                  />
                  <span>
                    <span className={styles.methodLabel}>{choice.label}</span>
                    <span className={styles.methodText}>{choice.text}</span>
                  </span>
                </label>
              ))}
            </fieldset>
          )}
        />

        {values.method === 'evenly' && budget !== null && (
          <p className={styles.note}>
            Each month gets {amountText(first)} {currency}
            {december !== first && `; December gets ${amountText(december)} ${currency}`}.
          </p>
        )}

        {values.method === 'custom' && (
          <CustomMonths
            form={form}
            year={year}
            currency={currency}
            budget={budget}
            plannedByMonth={plannedByMonth}
            leftOut={leftOut}
            entry={entry}
            onEntryChange={setEntry}
            copiedNote={copiedNote}
            onCopied={setCopiedNote}
          />
        )}

        {values.method !== 'none' && (
          <p
            className={balance.kind === 'tooMuch' ? styles.balanceProblem : styles.balance}
            aria-live="polite"
          >
            {balanceText(balance, currency)}
          </p>
        )}
        {values.method !== 'none' && budget === null && (
          <p className={styles.note}>Set a budget before phasing it.</p>
        )}
      </div>
    </FormDialog>
  );
}
