import { zodResolver } from '@hookform/resolvers/zod';
import { useEffect, useState } from 'react';
import { useForm, useWatch } from 'react-hook-form';
import { Button, LinkButton } from '@/ui/Button/Button';
import { FormErrorSummary } from '@/ui/FormErrorSummary/FormErrorSummary';
import { FormFooter } from '@/ui/FormFooter/FormFooter';
import { FailureMessage } from '@/ui/FormPanel/FormPanel';
import { FormLayout } from '@/ui/FormRail/FormRail';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import type { AttributeForm } from '@/ui/ShellAttributeFields/rules';
import { CHECK_ON_SAVE, useFormProblems } from '@/ui/useFormProblems';
import { useUnsavedChanges } from '@/ui/useUnsavedChanges';
import { toCents } from '@/format/decimal';
import styles from './LineItemForm.module.css';
import { rateOf } from './line-item-form-state';
import type { LineItemFormState } from './line-item-form-state';
import { lineItemRailSections } from './line-item-form-progress';
import { activityBreadcrumb } from './line-item-links';
import { convert, parseAmount } from './line-item-math';
import { toCreateBody } from './line-item-requests';
import type { LineItem, LineItemCreate, LineItemFormOptions } from './api';
import { CANCEL, GO_TO_BIN, GO_TO_PLANS, SAVE, SAVE_AND_ADD, SAVING } from './line-item-text';
import { LineItemFormCards } from './LineItemFormCards';
import { LineItemSummaryBar } from './LineItemSummaryBar';
import { RestartApprovalDialog } from './RestartApprovalDialog';
import { LINE_ITEM_FIELDS, lineItemFormSchema } from './schemas';
import { useLineItemSave } from './useLineItemSave';

/** The fields the API names in its problems, in the form's order. */
const FIELDS = [
  'name',
  'ownerId',
  'description',
  'currency',
  'plannedCost',
  'contractedAmount',
  'financeBudgetYear',
  'financeBudgetItemId',
  'requestSwbs',
] as const;

export interface LineItemFormProps {
  mode: 'create' | 'edit';
  options: LineItemFormOptions;
  attributeForm: AttributeForm;
  /** The plan's Business unit option ids, which decide the conditional attributes. */
  businessUnit: string[];
  /** The form's starting values: the defaults, or the stored line item's values. */
  initial: LineItemFormState;
  /** The stored line item (edit form). */
  stored?: LineItem;
  title: string;
  /** The last breadcrumb step: "New line item" or the line item's name. */
  breadcrumbEnd: string;
  /** A field to focus when the page opens (?field=, from completeness links). */
  focusField?: string | null;
  /** Focuses the name when the form opens (after Save and add another). */
  focusName?: boolean;
  /**
   * Sends the request; resolves with what to do next (leave the page, or open a new form), which
   * runs once the unsaved-changes guard is lifted. Throws an ApiError when it was refused.
   */
  onSave: (body: LineItemCreate, addAnother: boolean) => Promise<() => void>;
  /** After a conflict: take the latest version, so the next Save replaces the other change. */
  onConflict?: () => Promise<void>;
  onCancel: () => void;
}

/**
 * The line-item form of the create and edit pages (FR-006, FR-007, FR-027): breadcrumb, header,
 * the activity's summary bar, the sections in order, Save never disabled, every problem at once in
 * the summary and at the fields, the restart dialog before a change that sends the activity back
 * for approval (FR-031), a conflict with the values kept, unsaved changes protected.
 */
export function LineItemForm(props: LineItemFormProps) {
  const { mode, options, attributeForm, stored, focusField, focusName } = props;
  const { activity } = options;
  // The edit-form rule: the starting values are copied once; later refreshes never change them.
  const [start] = useState(props.initial);
  const form = useForm<LineItemFormState>({
    resolver: zodResolver(lineItemFormSchema),
    defaultValues: start,
    ...CHECK_ON_SAVE,
  });
  const formProblems = useFormProblems(form, FIELDS);
  const { problems } = formProblems;
  const state = useWatch({ control: form.control, compute: (values) => values });
  const save = useLineItemSave({
    activity,
    attributeForm,
    stored: stored ? { state: start, attributes: stored.attributes } : null,
    problems: formProblems,
    onSave: props.onSave,
    onConflict: props.onConflict,
  });
  // Only what would be sent counts: the notes and a custom grid kept for later are not changes.
  const isDirty = JSON.stringify(toCreateBody(state)) !== JSON.stringify(toCreateBody(start));
  const { afterSave } = save;
  const { dialog } = useUnsavedChanges(isDirty && afterSave === null);

  // Declared after the guard: by the time it runs, the guard has let go of the page.
  useEffect(() => {
    afterSave?.();
  }, [afterSave]);

  useEffect(() => {
    const id = focusField ?? (focusName ? 'name' : null);
    const field = id ? document.getElementById(id) : null;
    field?.scrollIntoView({ block: 'center' });
    field?.focus();
  }, [focusField, focusName]);

  /** Applies one of the form's rules to the values, field by field. */
  function update(edit: (current: LineItemFormState) => LineItemFormState) {
    const current = form.getValues();
    const next = edit(current);
    for (const key of LINE_ITEM_FIELDS) {
      if (next[key] !== current[key]) {
        form.setValue(key, next[key]);
      }
    }
  }

  const submit = (addAnother: boolean) =>
    form.handleSubmit((values) => {
      formProblems.clearProblems();
      save.save(values, addAnother);
    }, formProblems.onInvalid);

  const currencies = withStored(options.currencies, stored?.currency, (c) => c.code);
  const budgetItems = withStored(
    options.financeBudgetItems,
    stored?.financeBudgetItem ?? undefined,
    (item) => item.id,
  );
  const change = assignedChange(state, options, currencies, stored);

  return (
    <div className={styles.page}>
      {/* The breadcrumb names the activity, so no eyebrow or description repeats it (X-03). */}
      <PageHeader
        breadcrumb={[...activityBreadcrumb(activity), { label: props.breadcrumbEnd }]}
        title={props.title}
      />
      <form
        noValidate
        aria-busy={save.saving}
        className={styles.form}
        onSubmit={(event) => void submit(false)(event)}
      >
        <FormLayout
          sections={lineItemRailSections(state, attributeForm, props.businessUnit)}
          railExtra={<LineItemSummaryBar balance={options.balance} change={change} layout="rail" />}
          narrowExtra={<LineItemSummaryBar balance={options.balance} change={change} />}
        >
          {(save.failure || problems.length > 0) && (
            <div className={styles.stack}>
              {save.failure && <FailureMessage failure={save.failure} />}
              {save.failure?.gone && (
                <div>
                  {save.failure.gone === 'activity' ? (
                    <LinkButton to="/recycle-bin">{GO_TO_BIN}</LinkButton>
                  ) : (
                    <LinkButton to="/plans">{GO_TO_PLANS}</LinkButton>
                  )}
                </div>
              )}
              <FormErrorSummary errors={problems} />
            </div>
          )}
          <LineItemFormCards
            form={form}
            state={state}
            update={update}
            problems={problems}
            options={options}
            currencies={currencies}
            budgetItems={budgetItems}
            attributeForm={attributeForm}
            businessUnit={props.businessUnit}
            storedAttributes={stored?.attributes ?? {}}
          />
        </FormLayout>

        <FormFooter>
          <Button type="submit" variant="primary" aria-busy={save.saving || undefined}>
            {save.saving ? SAVING : SAVE}
          </Button>
          {mode === 'create' && <Button onClick={() => void submit(true)()}>{SAVE_AND_ADD}</Button>}
          {/* Leaving the page asks "Discard changes?" when something was typed (useUnsavedChanges). */}
          <Button onClick={props.onCancel}>{CANCEL}</Button>
        </FormFooter>
      </form>

      <RestartApprovalDialog
        open={save.pending !== null}
        onOpenChange={(open) => {
          if (!open) {
            save.cancelRestart();
          }
        }}
        activityName={activity.name}
        activityStatus={activity.status}
        reasons={save.pending?.reasons ?? []}
        busy={save.saving}
        onConfirm={save.confirmRestart}
      />
      {dialog}
    </div>
  );
}

/**
 * What saving changes the assigned amount by, in plan-currency cents (016 FR-015): the new planned
 * cost converted into the plan currency, less the saved one on an edit.
 */
function assignedChange(
  state: LineItemFormState,
  options: LineItemFormOptions,
  currencies: LineItemFormOptions['currencies'],
  stored: LineItem | undefined,
): bigint {
  const planCode = options.balance.assigned.currency;
  const planRate = rateOf(currencies, planCode) ?? '1';
  const rate = rateOf(currencies, state.currency) ?? '1';
  const newPlan = convert(parseAmount(state.plannedCost) ?? 0n, rate, planRate);
  return newPlan - (stored ? storedPlanCents(stored, planCode, planRate) : 0n);
}

/** The list with the stored value added when it is not in it (an inactive one, kept). */
function withStored<T>(list: T[], stored: T | undefined, key: (value: T) => string): T[] {
  return stored && !list.some((value) => key(value) === key(stored)) ? [...list, stored] : list;
}

/** The saved planned cost in the plan currency, in cents: the API's, else converted from its own. */
function storedPlanCents(stored: LineItem, planCode: string, planRate: string): bigint {
  if (stored.plannedCostPlan?.currency === planCode) {
    return toCents(stored.plannedCostPlan.amount);
  }
  if (planCode === 'USD') {
    return toCents(stored.plannedCostUsd.amount);
  }
  return convert(toCents(stored.plannedCost.amount), stored.currency.ratePerUsd, planRate);
}
