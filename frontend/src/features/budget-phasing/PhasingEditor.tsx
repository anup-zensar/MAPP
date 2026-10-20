import { useState } from 'react';
import { ApiError, conflictMessage } from '@/api/errors';
import {
  Button,
  FailureMessage,
  FormDialog,
  MoneyField,
  SegmentedFilter,
  useToast,
} from '@/components';
import type { SaveFailure } from '@/components';
import { formatCents, formatMoneyText, toCents } from '@/lib/decimal';
import { useFormatters } from '@/lib/useFormatters';
import { useSetPhasing } from './phasingApi';
import type { BudgetPhasing, PhasingLevel, PhasingMethod } from './phasingApi';
import styles from './BudgetByMonth.module.css';
import {
  amountText,
  balanceOf,
  balanceText,
  monthKeys,
  monthLabel,
  quarterLabels,
  quarterTotals,
  readAmount,
  splitEvenly,
  withQuarter,
} from './phasingMath';

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
  const [method, setMethod] = useState<PhasingMethod>(phasing.method);
  const [entry, setEntry] = useState<'monthly' | 'quarterly'>('monthly');
  const [texts, setTexts] = useState<string[]>(() =>
    keys.map((_, index) => {
      const amount = phasing.monthly[index]?.amount;
      return amount ? amountText(toCents(amount)) : '';
    }),
  );
  const [quarterTexts, setQuarterTexts] = useState<string[] | null>(null);
  const [copiedNote, setCopiedNote] = useState<string | null>(null);
  const [failure, setFailure] = useState<SaveFailure | null>(null);

  const months = texts.map((text) => readAmount(text));
  const readable = months.every((month) => month !== null);
  const cents = months.map((month) => month ?? 0n);
  const evenly = budget === null ? [] : splitEvenly(budget, 12);

  function fill(next: bigint[]) {
    setTexts(next.map((amount) => amountText(amount)));
    setQuarterTexts(null);
  }

  function typeQuarter(quarter: number, text: string) {
    const quarters = quarterTexts ?? quarterTotals(cents).map((amount) => amountText(amount));
    const nextQuarters = quarters.map((value, index) => (index === quarter ? text : value));
    setQuarterTexts(nextQuarters);
    const amount = readAmount(text);
    if (amount !== null) {
      setTexts(withQuarter(cents, quarter, amount).map((value) => amountText(value)));
    }
  }

  function copyPlannedCost() {
    fill(plannedByMonth.map((money) => (money ? toCents(money.amount) : 0n)));
    const parts = [
      leftOut.otherYears && `${formatMoneyText(leftOut.otherYears)} in other years`,
      leftOut.noDates && `${formatMoneyText(leftOut.noDates)} without dates`,
    ].filter(Boolean);
    setCopiedNote(parts.length > 0 ? `Left out: ${parts.join(' and ')}.` : null);
  }

  async function submit() {
    if (method === 'custom' && !readable) {
      setFailure({ message: 'Type each month as an amount of 0.00 or more.' });
      return;
    }
    try {
      await save.mutateAsync({
        method,
        months:
          method === 'custom'
            ? keys
                .map((month, index) => ({ month, amount: formatCents(cents[index]) }))
                .filter((part) => cents[keys.indexOf(part.month)] !== 0n)
            : [],
        version: record.version,
      });
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
            ? apiError.fields.map((field) => field.message).join(' ')
            : apiError.message,
      });
    }
  }

  const balance = balanceOf(method === 'evenly' ? evenly : cents, budget);

  return (
    <FormDialog
      open={open}
      onOpenChange={onOpenChange}
      title={`Budget by month — ${record.name}`}
      submitLabel="Save budget by month"
      onSubmit={() => void submit()}
      busy={save.isPending}
    >
      <div className={styles.editor}>
        {failure && <FailureMessage failure={failure} />}
        <fieldset className={styles.methods}>
          <legend className={styles.legend}>How to phase the budget</legend>
          {METHODS.map((choice) => (
            <label key={choice.value} className={styles.method}>
              <input
                type="radio"
                name="method"
                value={choice.value}
                checked={method === choice.value}
                onChange={() => setMethod(choice.value)}
              />
              <span>
                <span className={styles.methodLabel}>{choice.label}</span>
                <span className={styles.methodText}>{choice.text}</span>
              </span>
            </label>
          ))}
        </fieldset>

        {method === 'evenly' && budget !== null && (
          <p className={styles.note}>
            Each month gets {amountText(evenly[0])} {currency}
            {evenly[11] !== evenly[0] && `; December gets ${amountText(evenly[11])} ${currency}`}.
          </p>
        )}

        {method === 'custom' && (
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
                  setEntry(next);
                  setQuarterTexts(null);
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
                    <MoneyField
                      key={key}
                      label={monthLabel(key)}
                      name={`month-${key}`}
                      currency={currency}
                      value={texts[index]}
                      onChange={(text) =>
                        setTexts((current) =>
                          current.map((value, at) => (at === index ? text : value)),
                        )
                      }
                      error={months[index] === null ? 'Type an amount of 0.00 or more.' : undefined}
                    />
                  ))
                : quarterLabels(year).map((label, quarter) => (
                    <MoneyField
                      key={label}
                      label={label}
                      name={`quarter-${quarter + 1}`}
                      currency={currency}
                      value={quarterTexts?.[quarter] ?? amountText(quarterTotals(cents)[quarter])}
                      onChange={(text) => typeQuarter(quarter, text)}
                      note="Spread evenly over its three months"
                    />
                  ))}
            </div>
          </>
        )}

        {method !== 'none' && (
          <p
            className={balance.kind === 'tooMuch' ? styles.balanceProblem : styles.balance}
            aria-live="polite"
          >
            {balanceText(balance, currency)}
          </p>
        )}
        {method !== 'none' && budget === null && (
          <p className={styles.note}>Set a budget before phasing it.</p>
        )}
      </div>
    </FormDialog>
  );
}
