import { ChoiceField, InfoTip, MoneyField, problemOf } from '@/components';
import type { FieldProblem } from '@/components';
import { formatRate } from '@/lib/decimal';
import styles from './LineItemForm.module.css';
import type { LineItemFormState } from './lineItemFormState';
import { centsInput, parseAmount, toUsd } from './lineItemMath';
import type { LineItemCurrency } from './lineItemsApi';
import {
  CALCULATED,
  conversionText,
  conversionTip,
  currencyLabel,
  inactiveCurrencyNote,
  LABEL,
} from './lineItemText';

export interface CostFieldsProps {
  state: LineItemFormState;
  /** The active currencies and, on an edit form, the line item's inactive one (FR-011). */
  currencies: LineItemCurrency[];
  problems: FieldProblem[];
  onCurrencyChange: (code: string) => void;
  onPlannedCostChange: (text: string) => void;
}

/**
 * Currency and planned cost (FR-011, FR-012): the currency choice with an inactive currency kept
 * and explained, the amount in that currency, and for another currency than USD the calculated
 * "= 10,869.57 USD at 0.92 EUR per USD" under it.
 */
export function CostFields({
  state,
  currencies,
  problems,
  onCurrencyChange,
  onPlannedCostChange,
}: CostFieldsProps) {
  const currency = currencies.find((candidate) => candidate.code === state.currency);
  const cents = parseAmount(state.plannedCost);
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
        {currency && currency.code !== 'USD' && cents !== null && (
          <p className={styles.conversion}>
            <span className={styles.calculated}>{CALCULATED}</span>
            <span>
              {conversionText(
                centsInput(toUsd(cents, currency.ratePerUsd)),
                formatRate(currency.ratePerUsd),
                currency.code,
              )}
            </span>
            <InfoTip label="the calculated USD amount" text={conversionTip(currency.code)} />
          </p>
        )}
      </div>
    </>
  );
}
