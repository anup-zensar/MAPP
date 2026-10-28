import { useCurrencies, useFieldHelp } from '@/features/settings';
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

/** Budget: plan currency and budget (read-only when allocated), finance defaults and BCPIR. */
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
  const { help } = useFieldHelp();
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
              tip={help('plan.currency')}
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
          tip={help('plan.budget')}
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
              tip={help('plan.budget')}
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
            tip={help('plan.defaultFinanceBudget')}
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
            tip={help('plan.defaultSpread')}
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
            tip={help('plan.bcpir')}
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
  tip?: string;
  lock?: string;
}

/** The active currencies and the plan's own; locked with the API's reason once amounts use it. */
function CurrencyField(props: CurrencyFieldProps) {
  const { name, value, initial, hint, tip, lock, error, onChange } = props;
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
      tip={tip}
      options={options}
      value={value}
      onChange={onChange}
      disabledReason={lock}
      note="The plan budget and every amount in this plan are kept in this currency."
      error={error}
    />
  );
}

/** The budget in the reader's currency, or that clearing a phased budget removes its phasing. */
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
