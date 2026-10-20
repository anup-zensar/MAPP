import { ChoiceField, InfoTip, MoneyField, problemOf } from '@/components';
import type { FieldProblem } from '@/components';
import { formatRate } from '@/lib/decimal';
import { useFormatters } from '@/lib/useFormatters';
import styles from './LineItemForm.module.css';
import { rateOf } from './lineItemFormState';
import type { LineItemFormState } from './lineItemFormState';
import { centsInput, convert, parseAmount } from './lineItemMath';
import type { LineItemCurrency } from './lineItemsApi';
import {
  CALCULATED,
  conversionText,
  conversionTip,
  currencyLabel,
  inactiveCurrencyNote,
  LABEL,
  rateDatesText,
} from './lineItemText';
import { useCurrentRates } from './rates';

export interface CostFieldsProps {
  state: LineItemFormState;
  /** The active currencies and, on an edit form, the line item's inactive one (FR-011). */
  currencies: LineItemCurrency[];
  problems: FieldProblem[];
  /** The plan currency the planned cost is converted into (016 FR-015); USD before 016. */
  planCode?: string;
  onCurrencyChange: (code: string) => void;
  onPlannedCostChange: (text: string) => void;
}

/**
 * Currency and planned cost (FR-011, FR-012): the currency choice with an inactive currency kept
 * and explained, the amount in that currency, and for another currency than the plan's the
 * calculated amount in the plan currency under it with the rates and their date: "= 19,750.00 GBP
 * at 0.92 EUR and 0.79 GBP per USD (rates of 19 Oct 2026)" (016 FR-009, FR-015), or "= 10,869.57 USD
 * at 0.92 EUR per USD" in a USD plan.
 */
export function CostFields({
  state,
  currencies,
  problems,
  planCode = 'USD',
  onCurrencyChange,
  onPlannedCostChange,
}: CostFieldsProps) {
  const currency = currencies.find((candidate) => candidate.code === state.currency);
  const cents = parseAmount(state.plannedCost);
  const planRate = rateOf(currencies, planCode);
  const converts = Boolean(currency && currency.code !== planCode && cents !== null && planRate);
  const costNote =
    state.costNotes.length > 0 ? state.costNotes.join(' ') : state.sources.plannedCost;

  return (
    <>
      <ChoiceField
        label={LABEL.currency}
        name="currency"
        marking="Required"
        options={currencies.map((option) => ({
          id: option.code,
          label: currencyLabel(option.code, option.name, true),
          active: option.active,
        }))}
        value={state.currency}
        onChange={onCurrencyChange}
        error={problemOf(problems, 'currency')}
        note={
          currency && !currency.active
            ? inactiveCurrencyNote(currency.code, formatRate(currency.ratePerUsd))
            : state.sources.currency
        }
      />
      <div>
        <MoneyField
          label={LABEL.plannedCost}
          name="plannedCost"
          marking="Required"
          currency={state.currency}
          value={state.plannedCost}
          onChange={onPlannedCostChange}
          error={problemOf(problems, 'plannedCost')}
          note={costNote}
        />
        {converts && currency && cents !== null && planRate && (
          <Conversion
            cents={convert(cents, currency.ratePerUsd, planRate)}
            from={currency}
            planCode={planCode}
            planRate={planRate}
          />
        )}
      </div>
    </>
  );
}

/**
 * "Calculated = 19,750.00 GBP at 0.92 EUR and 0.79 GBP per USD (rates of 19 Oct 2026)": the rates
 * of the currencies other than USD, and their date once the current rates have loaded (FR-009).
 */
function Conversion({
  cents,
  from,
  planCode,
  planRate,
}: {
  cents: bigint;
  from: LineItemCurrency;
  planCode: string;
  planRate: string;
}) {
  const { date } = useFormatters();
  const rates = useCurrentRates();
  const parts = [
    { code: from.code, rate: from.ratePerUsd },
    { code: planCode, rate: planRate },
  ].filter((part) => part.code !== 'USD');
  const dates = parts.map(
    (part) => rates.data?.items.find((item) => item.currency === part.code)?.rateDate ?? null,
  );
  const known = dates.every((day) => day !== null)
    ? rateDatesText(dates.map((day) => date(day as string)))
    : null;
  return (
    <p className={styles.conversion}>
      <span className={styles.calculated}>{CALCULATED}</span>
      <span>
        {conversionText(
          centsInput(cents),
          planCode,
          parts.map((part) => ({ rate: formatRate(part.rate), code: part.code })),
          known,
        )}
      </span>
      <InfoTip
        label={`the calculated ${planCode} amount`}
        text={conversionTip(from.code, planCode)}
      />
    </p>
  );
}
