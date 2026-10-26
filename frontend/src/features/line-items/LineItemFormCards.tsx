import type { UseFormReturn } from 'react-hook-form';
import type { components } from '@/api/generated/schema';
import { CheckboxField } from '@/ui/CheckboxField/CheckboxField';
import { FormField } from '@/ui/FormField/FormField';
import { FormCard, FormCardFrame } from '@/ui/FormRail/FormRail';
import { MoneyField } from '@/ui/MoneyField/MoneyField';
import { ShellAttributeFields } from '@/ui/ShellAttributeFields/ShellAttributeFields';
import { TextAreaField } from '@/ui/TextAreaField/TextAreaField';
import { TextField } from '@/ui/TextField/TextField';
import type { FieldProblem } from '@/ui/FormErrorSummary/FormErrorSummary';
import type { AttributeForm } from '@/ui/ShellAttributeFields/rules';
import { PersonField } from '@/features/people';
import { recordHref } from '@/ui/record-links';
import { CostFields } from './CostFields';
import { FinanceBudgetFields } from './FinanceBudgetFields';
import styles from './LineItemForm.module.css';
import {
  addCustomMonth,
  changeBudgetYear,
  changeCurrency,
  changeCustomMonth,
  changePlannedCost,
  changeSpreadMethod,
  withField,
} from './line-item-form-state';
import type { LineItemFormState } from './line-item-form-state';
import { LINE_ITEM_SECTION } from './line-item-form-progress';
import type { FinanceBudgetItemRef, LineItemCurrency, LineItemFormOptions } from './api';
import {
  LABEL,
  SECTION_BASICS,
  SECTION_BUDGET,
  SECTION_COST,
  SECTION_FINANCE_CODING,
  SECTION_GSAP,
  SECTION_SPREAD,
  SWBS_HELP,
  SWBS_LATER,
} from './line-item-text';
import { SpreadEditor } from './SpreadEditor';

type AttributeValues = components['schemas']['AttributeValues'];

export interface LineItemFormCardsProps {
  form: UseFormReturn<LineItemFormState>;
  state: LineItemFormState;
  /** Applies one of the form's rules (line-item-form-state) to the values. */
  update: (edit: (current: LineItemFormState) => LineItemFormState) => void;
  /** Every problem of the last Save: the field groups, the months and the attributes read theirs. */
  problems: FieldProblem[];
  options: LineItemFormOptions;
  currencies: LineItemCurrency[];
  budgetItems: FinanceBudgetItemRef[];
  attributeForm: AttributeForm;
  businessUnit: string[];
  storedAttributes: AttributeValues;
}

/** The line-item form's six sections, in order (FR-007). */
export function LineItemFormCards(props: LineItemFormCardsProps) {
  const { form, state, update, problems, options, currencies, budgetItems } = props;
  const { activity } = options;
  const planCode = options.balance.assigned.currency;
  const dates = {
    startDate: activity.startDate,
    endDate: activity.endDate,
    ongoing: activity.ongoing,
  };
  const change = <K extends keyof LineItemFormState>(field: K, value: LineItemFormState[K]) =>
    update((current) => withField(current, field, value));

  return (
    <>
      <FormCard id={LINE_ITEM_SECTION.basics} title={SECTION_BASICS}>
        <FormField
          control={form.control}
          name="name"
          render={(field) => (
            <TextField
              label={LABEL.name}
              marking="Required"
              maxLength={120}
              {...field}
              onChange={(value) => change('name', value)}
              hint={state.sources.name}
            />
          )}
        />
        <FormField
          control={form.control}
          name="ownerId"
          render={(field) => (
            <PersonField
              label={LABEL.owner}
              marking="Needed to submit"
              {...field}
              onChange={(value) => change('ownerId', value)}
              hint={state.sources.ownerId}
            />
          )}
        />
        <FormField
          control={form.control}
          name="description"
          render={(field) => (
            <TextAreaField
              label={LABEL.description}
              maxLength={2000}
              {...field}
              onChange={(value) => change('description', value)}
            />
          )}
        />
      </FormCard>

      <FormCard id={LINE_ITEM_SECTION.cost} title={SECTION_COST}>
        <CostFields
          state={state}
          currencies={currencies}
          problems={problems}
          planCode={planCode}
          onCurrencyChange={(code) => update((s) => changeCurrency(s, code, currencies, planCode))}
          onPlannedCostChange={(text) => update((s) => changePlannedCost(s, text))}
        />
      </FormCard>

      <FormCard id={LINE_ITEM_SECTION.spread} title={SECTION_SPREAD}>
        <SpreadEditor
          method={state.spreadMethod}
          plannedCost={state.plannedCost}
          currency={state.currency}
          dates={dates}
          customMonths={state.customMonths}
          customTyped={state.customTyped}
          source={state.sources.spread}
          problems={problems}
          onMethodChange={(method) =>
            update((s) => changeSpreadMethod(s, method, dates, activity.planYear))
          }
          onMonthChange={(month, amount) => update((s) => changeCustomMonth(s, month, amount))}
          onAddMonth={(month) => update((s) => addCustomMonth(s, month))}
          onUseTotal={(amount) => update((s) => changePlannedCost(s, amount))}
        />
      </FormCard>

      {/* The attribute group's own name is the card's heading (UI audit LI-03, X-09). */}
      <FormCardFrame
        id={LINE_ITEM_SECTION.coding}
        label={SECTION_FINANCE_CODING}
        className={styles.codingCard}
      >
        <ShellAttributeFields
          sectionHeading="h2"
          form={props.attributeForm}
          values={state.attributes}
          onChange={(values) => change('attributes', values)}
          businessUnit={props.businessUnit}
          previousValues={props.storedAttributes}
          errors={attributeErrors(problems)}
          planLink={recordHref('plan', activity.path[0]?.id ?? '')}
        />
        <FormField
          control={form.control}
          name="contractedAmount"
          render={(field) => (
            <MoneyField
              label={LABEL.contractedAmount}
              currency={state.currency}
              {...field}
              onChange={(value) => change('contractedAmount', value)}
            />
          )}
        />
      </FormCardFrame>

      <FormCard id={LINE_ITEM_SECTION.budget} title={SECTION_BUDGET}>
        <FinanceBudgetFields
          state={state}
          activity={activity}
          years={options.financeBudgetYears}
          items={budgetItems}
          problems={problems}
          onYearChange={(year) => update((s) => changeBudgetYear(s, year, budgetItems))}
          onItemChange={(id) => change('financeBudgetItemId', id)}
        />
      </FormCard>

      <FormCard id={LINE_ITEM_SECTION.gsap} title={SECTION_GSAP}>
        <FormField
          control={form.control}
          name="requestSwbs"
          render={(field) => (
            <CheckboxField
              label={LABEL.requestSwbs}
              name={field.name}
              checked={field.value}
              onChange={(checked) => change('requestSwbs', checked)}
              hint={[SWBS_HELP, state.sources.requestSwbs].filter(Boolean).join(' ')}
              error={field.error}
            />
          )}
        />
        <div className={styles.readOnlyField}>
          <span className={styles.readOnlyLabel}>{LABEL.swbsCode}</span>
          <span className={styles.note}>{SWBS_LATER}</span>
        </div>
      </FormCard>
    </>
  );
}

/** The problems of Shell attributes by field name, as ShellAttributeFields takes them. */
function attributeErrors(problems: FieldProblem[]): Record<string, string> {
  return Object.fromEntries(
    problems
      .filter((problem) => problem.field.startsWith('attributes.'))
      .map((problem) => [problem.field, problem.message]),
  );
}
