import { useCurrencies } from '@/features/settings';
import type { useFinanceBudgets } from '@/features/settings';
import { CheckboxField } from '@/ui/CheckboxField/CheckboxField';
import { ChoiceField } from '@/ui/ChoiceField/ChoiceField';
import { FormField } from '@/ui/FormField/FormField';
import type { FieldBinding } from '@/ui/FormField/FormField';
import { MoneyField } from '@/ui/MoneyField/MoneyField';
import { formatMoneyText } from '@/format/decimal';
import type { Plan } from './api';
import type { SectionProps } from './plan-form-sections';
import {
  allocationLockText,
  APPLIES_FROM_NOW,
  PHASING_REMOVED,
  SPREAD_LABELS,
  SPREAD_METHODS,
} from './plan-text';
import { PlanFormCard } from './PlanFormCard';
import { ReadOnlyField } from './ReadOnlyField';
import type { PlanFormValues } from './schemas';

/** The choice that stands for "no default finance budget" (ChoiceField's empty entry is "Choose…"). */
const NO_DEFAULT = 'none';

export interface PlanBudgetSectionProps extends SectionProps {
  initial: PlanFormValues;
  financeBudgets: ReturnType<typeof useFinanceBudgets>;
  /** Why the default finance budget was cleared, after another year was chosen. */
  budgetNote: string | null;
  onFinanceBudgetChange: () => void;
}

/**
 * Budget: the plan currency and budget (016), the default finance budget and spread, and BCPIR.
 * With 014 an allocated plan's budget is read-only with its reason.
 */
export function PlanBudgetSection({
  control,
  values,
  mode,
  plan,
  with016,
  hint,
  initial,
  financeBudgets,
  budgetNote,
  onFinanceBudgetChange,
}: PlanBudgetSectionProps) {
  const budgets = financeBudgets.data?.items ?? [];
  const budgetOptions = [
    { id: NO_DEFAULT, label: 'No default', active: true },
    ...budgets.map((budget) => ({ id: budget.id, label: budget.name, active: true })),
  ];
  const storedBudget = plan?.defaultFinanceBudget;
  if (storedBudget && !budgets.some((budget) => budget.id === storedBudget.id)) {
    budgetOptions.push({ id: storedBudget.id, label: storedBudget.name, active: true });
  }
  const noBudgetsYet =
    financeBudgets.isSuccess && budgets.length === 0
      ? `No finance budgets for ${values.year} yet — an administrator adds them in Settings.`
      : null;
  // 014: an allocated plan's budget is set by its global budget and can't be typed here.
  const allocation = plan?.allocation ?? null;
  const allocationLock = allocation
    ? allocationLockText(allocation, plan?.access?.editBudget.reason)
    : null;

  return (
    <PlanFormCard id="plan-budget" title="Budget">
      {with016 && (
        <FormField
          control={control}
          name="currency"
          render={(field) => (
            <CurrencyField
              initial={initial.currency}
              hint={hint('currency', 'Default: USD')}
              lock={plan?.currencyLock ?? undefined}
              {...field}
              onChange={(currency) => field.onChange(currency || field.value)}
            />
          )}
        />
      )}
      {allocationLock ? (
        <ReadOnlyField
          label="Plan budget"
          value={values.budget ? `${values.budget} ${values.currency}` : '—'}
        >
          {allocationLock}
        </ReadOnlyField>
      ) : (
        <FormField
          control={control}
          name="budget"
          render={(field) => (
            <MoneyField
              label="Plan budget"
              currency={values.currency}
              note={budgetFieldNote(values, initial, plan) ?? undefined}
              {...field}
            />
          )}
        />
      )}
      <FormField
        control={control}
        name="defaultFinanceBudgetId"
        render={(field) => (
          <ChoiceField
            label="Default finance budget"
            options={budgetOptions}
            note={
              [
                budgetNote,
                noBudgetsYet,
                mode === 'edit' && field.value !== initial.defaultFinanceBudgetId
                  ? APPLIES_FROM_NOW
                  : null,
              ]
                .filter(Boolean)
                .join(' ') || undefined
            }
            {...field}
            value={field.value || NO_DEFAULT}
            onChange={(id) => {
              onFinanceBudgetChange();
              field.onChange(id === NO_DEFAULT ? '' : id);
            }}
          />
        )}
      />
      <FormField
        control={control}
        name="defaultSpread"
        render={(field) => (
          <ChoiceField
            label="Default line-item spread"
            hint={hint('defaultSpread', 'Default: Evenly')}
            options={SPREAD_METHODS.map((id) => ({ id, label: SPREAD_LABELS[id], active: true }))}
            note={
              mode === 'edit' && field.value !== initial.defaultSpread
                ? APPLIES_FROM_NOW
                : undefined
            }
            {...field}
            onChange={(spread) =>
              field.onChange(SPREAD_METHODS.find((method) => method === spread) ?? field.value)
            }
          />
        )}
      />
      <FormField
        control={control}
        name="bcpir"
        render={({ name, value, onChange }) => (
          <CheckboxField
            label="BCPIR plan"
            name={name}
            hint={hint('bcpir', 'Default: No')}
            checked={value}
            onChange={onChange}
          />
        )}
      />
    </PlanFormCard>
  );
}

interface CurrencyFieldProps extends FieldBinding<string> {
  initial: string;
  hint?: string;
  lock?: string;
}

/**
 * The plan currency (016 FR-011, FR-012): the active currencies, the plan's own kept even when it
 * was set inactive since; locked with the API's reason once amounts are kept in it.
 */
function CurrencyField({ name, value, initial, hint, lock, error, onChange }: CurrencyFieldProps) {
  const currencies = useCurrencies();
  const options = (currencies.data?.items ?? [])
    .filter((currency) => currency.active || currency.code === initial)
    .map((currency) => ({
      id: currency.code,
      label: `${currency.code} – ${currency.name}`,
      active: currency.active,
    }));
  if (!options.some((option) => option.id === value)) {
    options.unshift({ id: value, label: value, active: true });
  }
  return (
    <ChoiceField
      label="Plan currency"
      name={name}
      marking="Required"
      hint={hint}
      options={options}
      value={value}
      onChange={onChange}
      disabledReason={lock}
      note="The plan budget and every amount in this plan are kept in this currency."
      error={error}
    />
  );
}

/**
 * The note under the budget (016): what the saved budget is in the reader's display currency, and
 * that clearing a phased budget removes its Budget by month (FR-022).
 */
function budgetFieldNote(
  values: PlanFormValues,
  initial: PlanFormValues,
  plan: Plan | undefined,
): string | null {
  if (
    plan &&
    initial.budget !== '' &&
    values.budget.trim() === '' &&
    plan.budgetPhasing &&
    plan.budgetPhasing.method !== 'none'
  ) {
    return PHASING_REMOVED;
  }
  const display = plan?.figures.budget;
  if (
    plan &&
    display &&
    values.budget === initial.budget &&
    values.currency === initial.currency &&
    display.currency !== values.currency
  ) {
    return `≈ ${formatMoneyText(display)} at today's rate`;
  }
  return null;
}
