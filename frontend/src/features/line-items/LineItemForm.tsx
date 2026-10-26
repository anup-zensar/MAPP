import { useEffect, useState } from 'react';
import type { FormEvent } from 'react';
import { ApiError, conflictMessage } from '@/api/errors';
import type { components } from '@/api/generated/schema';
import { Button, LinkButton } from '@/ui/Button/Button';
import { CheckboxField } from '@/ui/CheckboxField/CheckboxField';
import { FormErrorSummary } from '@/ui/FormErrorSummary/FormErrorSummary';
import { FormFooter } from '@/ui/FormFooter/FormFooter';
import { FailureMessage } from '@/ui/FormPanel/FormPanel';
import { problemOf } from '@/ui/FormPanel/problemOf';
import { MoneyField } from '@/ui/MoneyField/MoneyField';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { ShellAttributeFields } from '@/ui/ShellAttributeFields/ShellAttributeFields';
import { valuesToClear } from '@/ui/ShellAttributeFields/rules';
import { TextAreaField } from '@/ui/TextAreaField/TextAreaField';
import { TextField } from '@/ui/TextField/TextField';
import type { FieldProblem } from '@/ui/FormErrorSummary/FormErrorSummary';
import type { SaveFailure } from '@/ui/FormPanel/problemOf';
import type { AttributeForm } from '@/ui/ShellAttributeFields/rules';
import { FormCard, FormCardFrame, FormLayout } from '@/ui/FormRail/FormRail';
import { PersonField } from '@/features/people';
import { toCents } from '@/format/decimal';
import { recordHref } from '@/ui/record-links';
import { useFormatters } from '@/format/useFormatters';
import { useUnsavedChanges } from '@/ui/useUnsavedChanges';
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
  formProblems,
  isRestartRefusal,
  needsRestartConfirmation,
  rateOf,
  restartReasons,
  toCreateBody,
  withField,
} from './lineItemFormState';
import type { LineItemFormState } from './lineItemFormState';
import { LINE_ITEM_SECTION, lineItemRailSections } from './lineItemFormProgress';
import { convert, parseAmount } from './lineItemMath';
import { activityBreadcrumb } from './lineItemLinks';
import type { LineItem, LineItemCreate, LineItemFormOptions } from './lineItemsApi';
import {
  ACTIVITY_GONE,
  CANCEL,
  GO_TO_BIN,
  GO_TO_PLANS,
  LABEL,
  SAVE,
  SAVE_AND_ADD,
  SAVING,
  SECTION_BASICS,
  SECTION_BUDGET,
  SECTION_COST,
  SECTION_FINANCE_CODING,
  SECTION_GSAP,
  SECTION_SPREAD,
  SWBS_HELP,
  SWBS_LATER,
} from './lineItemText';
import { LineItemSummaryBar } from './LineItemSummaryBar';
import { RestartApprovalDialog } from './RestartApprovalDialog';
import { SpreadEditor } from './SpreadEditor';

type AttributeValues = components['schemas']['AttributeValues'];
type ClearedValue = components['schemas']['ClearedValue'];

const CONFLICT_NEXT_STEP =
  'Your changes are still here. Save again to replace their changes, or Cancel to see theirs.';

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
  /** The page's <h1>. */
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

/** A save waiting for "Save and restart approval". */
interface PendingSave {
  body: LineItemCreate;
  addAnother: boolean;
  reasons: string[];
}

/**
 * The line-item form of the create and edit pages (FR-006, FR-007, FR-027): breadcrumb, header,
 * the activity's summary bar, the seven sections in order, Save never disabled, every problem at
 * once in the summary and at the fields, the restart dialog before a change that sends the
 * activity back for approval (FR-031), a conflict with the values kept, unsaved changes protected.
 */
export function LineItemForm({
  mode,
  options,
  attributeForm,
  businessUnit,
  initial,
  stored,
  title,
  breadcrumbEnd,
  focusField,
  focusName,
  onSave,
  onConflict,
  onCancel,
}: LineItemFormProps) {
  const { dateFormat } = useFormatters();
  const { activity } = options;
  // The edit-form rule: the starting values are copied once; later refreshes never change them.
  const [start] = useState(initial);
  const [state, setState] = useState(initial);
  const [problems, setProblems] = useState<FieldProblem[]>([]);
  const [failure, setFailure] = useState<(SaveFailure & { gone?: 'activity' | 'lineItem' }) | null>(
    null,
  );
  const [saving, setSaving] = useState(false);
  const [pending, setPending] = useState<PendingSave | null>(null);
  // What to do after a successful save; set, it lifts the unsaved-changes guard first.
  const [next, setNext] = useState<(() => void) | null>(null);

  const dirty = JSON.stringify(toCreateBody(state)) !== JSON.stringify(toCreateBody(start));
  const { dialog } = useUnsavedChanges(dirty && next === null);

  useEffect(() => {
    next?.();
  }, [next]);

  useEffect(() => {
    const id = focusField ?? (focusName ? 'name' : null);
    const field = id ? document.getElementById(id) : null;
    field?.scrollIntoView({ block: 'center' });
    field?.focus();
  }, [focusField, focusName]);

  const currencies = withStored(options.currencies, stored?.currency, (c) => c.code);
  const budgetItems = withStored(
    options.financeBudgetItems,
    stored?.financeBudgetItem ?? undefined,
    (item) => item.id,
  );
  const storedState = mode === 'edit' ? start : null;
  const dates = {
    startDate: activity.startDate,
    endDate: activity.endDate,
    ongoing: activity.ongoing,
  };

  // The plan currency the unassigned planned cost is worked out in (016 FR-015); USD before 016.
  const planCode = options.balance.assigned.currency;
  const planRate = rateOf(currencies, planCode) ?? '1';
  // Saving changes the assigned amount by the new planned cost in the plan currency, less the saved
  // one (edit).
  const rate = rateOf(currencies, state.currency) ?? '1';
  const newPlan = convert(parseAmount(state.plannedCost) ?? 0n, rate, planRate);
  const change = newPlan - (stored ? storedPlanCents(stored, planCode, planRate) : 0n);

  function update(edit: (current: LineItemFormState) => LineItemFormState) {
    setState((current) => edit(current));
  }

  function submit(addAnother: boolean) {
    if (saving) {
      return;
    }
    const cleared = valuesToClear(attributeForm, state.attributes, stored?.attributes ?? {});
    const body = toCreateBody(state, withoutCleared(state.attributes, cleared));
    if (needsRestartConfirmation(activity, storedState, state)) {
      setPending({ body, addAnother, reasons: restartReasons(storedState, state) });
      return;
    }
    void send(body, addAnother);
  }

  async function send(body: LineItemCreate, addAnother: boolean) {
    setSaving(true);
    try {
      const after = await onSave(body, addAnother);
      setPending(null);
      setNext(() => after);
    } catch (error) {
      await showFailure(error, body, addAnother);
    } finally {
      setSaving(false);
    }
  }

  async function showFailure(error: unknown, body: LineItemCreate, addAnother: boolean) {
    const apiError = error instanceof ApiError ? error : new ApiError('other');
    if (isRestartRefusal(apiError)) {
      // Someone submitted the activity meanwhile: the person confirms first (D-146).
      setPending({ body, addAnother, reasons: restartReasons(storedState, state) });
      setProblems(formProblems(apiError, body));
      setFailure(null);
      return;
    }
    setPending(null);
    if (apiError.kind === 'validation') {
      setProblems(formProblems(apiError, body));
      setFailure(null);
      return;
    }
    setProblems([]);
    if (apiError.kind === 'conflict') {
      await onConflict?.().catch(() => undefined);
      setFailure({
        message: conflictMessage(apiError, 'line item', dateFormat),
        nextStep: CONFLICT_NEXT_STEP,
      });
      return;
    }
    if (apiError.kind === 'notFound') {
      setFailure({
        message: apiError.message,
        gone: apiError.message === ACTIVITY_GONE ? 'activity' : 'lineItem',
      });
      return;
    }
    setFailure({ message: apiError.message });
  }

  function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    submit(false);
  }

  const attributeErrors = Object.fromEntries(
    problems
      .filter((problem) => problem.field.startsWith('attributes.'))
      .map((problem) => [problem.field, problem.message]),
  );

  const railSections = lineItemRailSections(state, attributeForm, businessUnit);

  return (
    <div className={styles.page}>
      {/* The breadcrumb names the activity, so no eyebrow or description repeats it (X-03). */}
      <PageHeader
        breadcrumb={[...activityBreadcrumb(activity), { label: breadcrumbEnd }]}
        title={title}
      />
      <form noValidate aria-busy={saving} className={styles.form} onSubmit={onSubmit}>
        <FormLayout
          sections={railSections}
          railExtra={<LineItemSummaryBar balance={options.balance} change={change} layout="rail" />}
          narrowExtra={<LineItemSummaryBar balance={options.balance} change={change} />}
        >
          {(failure || problems.length > 0) && (
            <div className={styles.stack}>
              {failure && <FailureMessage failure={failure} />}
              {failure?.gone && (
                <div>
                  {failure.gone === 'activity' ? (
                    <LinkButton to="/recycle-bin">{GO_TO_BIN}</LinkButton>
                  ) : (
                    <LinkButton to="/plans">{GO_TO_PLANS}</LinkButton>
                  )}
                </div>
              )}
              <FormErrorSummary errors={problems} />
            </div>
          )}
          <FormCard id={LINE_ITEM_SECTION.basics} title={SECTION_BASICS}>
            <TextField
              label={LABEL.name}
              name="name"
              marking="Required"
              maxLength={120}
              value={state.name}
              onChange={(value) => update((s) => withField(s, 'name', value))}
              hint={state.sources.name}
              error={problemOf(problems, 'name')}
            />
            <PersonField
              label={LABEL.owner}
              name="ownerId"
              marking="Needed to submit"
              value={state.ownerId}
              onChange={(value) => update((s) => withField(s, 'ownerId', value))}
              hint={state.sources.ownerId}
              error={problemOf(problems, 'ownerId')}
            />
            <TextAreaField
              label={LABEL.description}
              name="description"
              maxLength={2000}
              value={state.description}
              onChange={(value) => update((s) => withField(s, 'description', value))}
              error={problemOf(problems, 'description')}
            />
          </FormCard>

          <FormCard id={LINE_ITEM_SECTION.cost} title={SECTION_COST}>
            <CostFields
              state={state}
              currencies={currencies}
              problems={problems}
              planCode={planCode}
              onCurrencyChange={(code) =>
                update((s) => changeCurrency(s, code, currencies, planCode))
              }
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
              form={attributeForm}
              values={state.attributes}
              onChange={(values) => update((s) => withField(s, 'attributes', values))}
              businessUnit={businessUnit}
              previousValues={stored?.attributes ?? {}}
              errors={attributeErrors}
              planLink={recordHref('plan', activity.path[0]?.id ?? '')}
            />
            <MoneyField
              label={LABEL.contractedAmount}
              name="contractedAmount"
              currency={state.currency}
              value={state.contractedAmount}
              onChange={(value) => update((s) => withField(s, 'contractedAmount', value))}
              error={problemOf(problems, 'contractedAmount')}
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
              onItemChange={(id) => update((s) => withField(s, 'financeBudgetItemId', id))}
            />
          </FormCard>

          <FormCard id={LINE_ITEM_SECTION.gsap} title={SECTION_GSAP}>
            <CheckboxField
              label={LABEL.requestSwbs}
              name="requestSwbs"
              checked={state.requestSwbs}
              onChange={(checked) => update((s) => withField(s, 'requestSwbs', checked))}
              hint={[SWBS_HELP, state.sources.requestSwbs].filter(Boolean).join(' ')}
              error={problemOf(problems, 'requestSwbs')}
            />
            <div className={styles.readOnlyField}>
              <span className={styles.readOnlyLabel}>{LABEL.swbsCode}</span>
              <span className={styles.note}>{SWBS_LATER}</span>
            </div>
          </FormCard>
        </FormLayout>

        <FormFooter>
          <Button type="submit" variant="primary" aria-busy={saving || undefined}>
            {saving ? SAVING : SAVE}
          </Button>
          {mode === 'create' && <Button onClick={() => submit(true)}>{SAVE_AND_ADD}</Button>}
          {/* Leaving the page asks "Discard changes?" when something was typed (useUnsavedChanges). */}
          <Button onClick={onCancel}>{CANCEL}</Button>
        </FormFooter>
      </form>

      <RestartApprovalDialog
        open={pending !== null}
        onOpenChange={(open) => {
          if (!open) {
            setPending(null);
          }
        }}
        activityName={activity.name}
        activityStatus={activity.status}
        reasons={pending?.reasons ?? []}
        busy={saving}
        onConfirm={() => {
          if (pending) {
            void send({ ...pending.body, restartApproval: true }, pending.addAnother);
          }
        }}
      />
      {dialog}
    </div>
  );
}

/** The values to send: those the form will clear (D-151) are removed. */
function withoutCleared(values: AttributeValues, cleared: ClearedValue[]): AttributeValues {
  const result = { ...values };
  for (const { attributeId, optionIds } of cleared) {
    const value = result[attributeId];
    if (Array.isArray(value)) {
      const rest = value.filter(
        (item) => !optionIds.includes(typeof item === 'string' ? item : item.optionId),
      );
      result[attributeId] = rest.length > 0 ? (rest as AttributeValues[string]) : null;
    } else {
      result[attributeId] = null;
    }
  }
  return result;
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
