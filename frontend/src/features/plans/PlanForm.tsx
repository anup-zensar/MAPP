import { useEffect, useRef, useState } from 'react';
import type { FormEvent } from 'react';
import { useNavigate } from 'react-router';
import { useAttributeForm, useFinanceBudgets, useModels } from '@/api/settingsQueries';
import {
  Button,
  CheckboxField,
  ChoiceField,
  ClearValuesDialog,
  ErrorState,
  FailureMessage,
  FormErrorSummary,
  FormFooter,
  FormSection,
  LockedNote,
  MoneyField,
  problemOf,
  ShellAttributeFields,
  Skeleton,
  TextAreaField,
  TextField,
  valuesToClear,
} from '@/components';
import type { FieldProblem, SaveFailure } from '@/components';
import { PersonField, TeamField, usePeople } from '@/features/people';
import { formatAmount, formatPercent } from '@/lib/decimal';
import { useUnsavedChanges } from '@/lib/useUnsavedChanges';
import { GoalFields } from './GoalFields';
import { sameValues } from './planFormValues';
import type { PlanFormValues } from './planFormValues';
import styles from './PlanForm.module.css';
import type { Plan, PlanDefaults } from './plansApi';
import { APPLIES_FROM_NOW, PUBLISHED_BANNER, SPREAD_LABELS, YEAR_LOCKED } from './planText';
import type { SpreadMethod } from './planText';

/** The choice that stands for "no default finance budget" (ChoiceField's empty entry is "Choose…"). */
const NO_DEFAULT = 'none';

export interface PlanFormProps {
  mode: 'create' | 'edit';
  /** The values the form starts with; copied once (edit-form rule). */
  initial: PlanFormValues;
  /** Create: the defaults, for the years to choose from and the "Default: …" hints (FR-006). */
  defaults?: PlanDefaults;
  /** Edit: the plan as it was when the form opened. */
  plan?: Plan;
  /** The field problems of the last save, all at once. */
  problems: FieldProblem[];
  /** A failure that is not about one field (a conflict, the server unreachable). */
  failure: SaveFailure | null;
  /** Saves; answers where to go once saved, or null when nothing was saved. */
  onSave: (values: PlanFormValues) => Promise<string | null>;
  /** Where Cancel leads: the list or the plan page. */
  cancelTo: string;
  /** The field to focus once the form is drawn (`?focus=` from completeness links). */
  focusField?: string | null;
}

/**
 * The plan form of the create and edit pages (plan "Plan form", FR-005 – FR-010, FR-018 – FR-023):
 * one column with Basics, Budget, Shell attributes and Goals in the order fields depend on each
 * other, defaults labelled, markings as words, every problem at once, a sticky Save and Cancel,
 * and "Discard changes?" before typed values are lost.
 */
export function PlanForm({
  mode,
  initial,
  defaults,
  plan,
  problems,
  failure,
  onSave,
  cancelTo,
  focusField,
}: PlanFormProps) {
  const navigate = useNavigate();
  const models = useModels();
  const attributeForm = useAttributeForm('plan');
  const people = usePeople();
  const [values, setValues] = useState(initial);
  const [saving, setSaving] = useState(false);
  const [leaveTo, setLeaveTo] = useState<string | null>(null);
  const [budgetNote, setBudgetNote] = useState<string | null>(null);
  const [confirmClear, setConfirmClear] = useState(false);
  const financeBudgets = useFinanceBudgets(Number(values.year) || undefined);
  const { dialog, confirmDiscard } = useUnsavedChanges(!leaveTo && !sameValues(values, initial));
  const focused = useRef(false);
  const ready = models.isSuccess && attributeForm.isSuccess;

  // Once saved, the guard is off (the line above) before the page is left.
  useEffect(() => {
    if (leaveTo) {
      void navigate(leaveTo);
    }
  }, [leaveTo, navigate]);

  // ?focus=<field>: once the form is drawn, the field gets the focus and comes into view.
  useEffect(() => {
    if (!ready || !focusField || focused.current) {
      return;
    }
    focused.current = true;
    const target = document.getElementById(focusField);
    const control =
      target instanceof HTMLFieldSetElement
        ? target.querySelector<HTMLElement>('input, select, textarea')
        : target;
    control?.scrollIntoView?.({ block: 'center' });
    control?.focus();
  }, [ready, focusField]);

  if (models.isError || attributeForm.isError) {
    return (
      <ErrorState
        error={models.error ?? attributeForm.error}
        onRetry={() => {
          void models.refetch();
          void attributeForm.refetch();
        }}
      />
    );
  }
  if (!ready) {
    return <PlanFormSkeleton />;
  }
  const shellForm = attributeForm.data;

  function set<K extends keyof PlanFormValues>(key: K, value: PlanFormValues[K]) {
    setValues((current) => ({ ...current, [key]: value }));
  }

  /** "Default: …" while the field still holds the value the defaults gave it (FR-006). */
  function defaultHint<K extends keyof PlanFormValues>(key: K, text: string, given = true) {
    return mode === 'create' && given && sameValues({ ...values, [key]: initial[key] }, values)
      ? text
      : undefined;
  }

  function changeYear(year: string) {
    const chosen = financeBudgets.data?.items.find(
      (budget) => budget.id === values.defaultFinanceBudgetId,
    );
    if (chosen && String(chosen.year) !== year) {
      setBudgetNote(`Default finance budget cleared — '${chosen.name}' belongs to ${chosen.year}.`);
      setValues((current) => ({ ...current, year, defaultFinanceBudgetId: '' }));
      return;
    }
    set('year', year);
  }

  async function save() {
    setConfirmClear(false);
    setSaving(true);
    try {
      const destination = await onSave(values);
      if (destination) {
        setLeaveTo(destination);
      }
    } finally {
      setSaving(false);
    }
  }

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    // A second Save while saving is ignored; Save is never greyed out.
    if (saving) {
      return;
    }
    if (valuesToClear(shellForm, values.attributes, plan?.attributes ?? {}).length > 0) {
      setConfirmClear(true);
      return;
    }
    void save();
  }

  const error = (field: string) => problemOf(problems, field);
  const attributeErrors = Object.fromEntries(
    problems
      .filter((problem) => problem.field.startsWith('attributes.'))
      .map((problem) => [problem.field, problem.message]),
  );

  const modelItems = models.data.items;
  const chosenModel =
    modelItems.find((model) => model.id === values.modelId) ??
    (plan && plan.model.id === values.modelId ? plan.model : undefined);
  const modelOptions = modelItems.map((model) => ({
    id: model.id,
    label: model.name,
    active: model.active,
  }));
  if (plan && !modelItems.some((model) => model.id === plan.model.id)) {
    modelOptions.push({ id: plan.model.id, label: plan.model.name, active: plan.model.active });
  }

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

  const ownerName =
    people.data?.items.find((person) => person.id === values.ownerId)?.name ??
    (plan?.owner?.id === values.ownerId ? plan.owner.name : 'The owner');

  return (
    <>
      <form noValidate aria-busy={saving} className={styles.form} onSubmit={submit}>
        {mode === 'edit' && plan?.status === 'published' && (
          <p role="note" className={styles.banner}>
            {PUBLISHED_BANNER}
          </p>
        )}
        {failure && <FailureMessage failure={failure} />}
        <FormErrorSummary errors={problems} />

        <FormSection title="Basics">
          <TextField
            label="Plan name"
            name="name"
            marking="Required"
            maxLength={120}
            value={values.name}
            onChange={(name) => set('name', name)}
            error={error('name')}
          />
          {mode === 'create' ? (
            <ChoiceField
              label="Year"
              name="year"
              marking="Required"
              hint={defaultHint('year', 'Default: the current year')}
              options={(defaults?.years ?? [Number(values.year)]).map((year) => ({
                id: String(year),
                label: String(year),
                active: true,
              }))}
              value={values.year}
              onChange={changeYear}
              error={error('year')}
            />
          ) : (
            <div className={styles.readOnly}>
              <p className={styles.readOnlyLabel}>
                Year <span className={styles.marking}>Required</span>
              </p>
              <p className={styles.readOnlyValue}>{values.year}</p>
              <LockedNote>{YEAR_LOCKED}</LockedNote>
            </div>
          )}
          <ChoiceField
            label="Model"
            name="modelId"
            marking="Required"
            hint={defaultHint('modelId', 'Default: the only active model', !!defaults?.modelId)}
            options={modelOptions}
            value={values.modelId}
            onChange={(modelId) => set('modelId', modelId)}
            note={
              chosenModel
                ? `INQ→MQL ${formatPercent(chosenModel.inqToMqlRate)} · MQL→deal ${formatPercent(chosenModel.mqlToDealRate)} · average deal size ${formatAmount(chosenModel.averageDealSize.amount)} ${chosenModel.averageDealSize.currency}`
                : undefined
            }
            error={error('modelId')}
          />
          <TeamField
            label="Teams"
            name="teamIds"
            marking="Needed to publish"
            hint={defaultHint('teamIds', 'Default: your teams', !!defaults?.teamIds.length)}
            value={values.teamIds}
            onChange={(teamIds) => set('teamIds', teamIds)}
            error={error('teamIds')}
          />
          <PersonField
            label="Owner"
            name="ownerId"
            marking="Needed to publish"
            hint={defaultHint('ownerId', 'Default: you', !!defaults?.ownerId)}
            teamIds={values.teamIds}
            notMemberText={`${ownerName} isn't in any of the chosen teams — choose another owner or clear the field`}
            value={values.ownerId}
            onChange={(ownerId) => set('ownerId', ownerId)}
            error={error('ownerId')}
          />
          <CheckboxField
            label="Strategic / AOP plan"
            name="aop"
            hint={defaultHint('aop', 'Default: No')}
            checked={values.aop}
            onChange={(aop) => set('aop', aop)}
          />
          <TextAreaField
            label="Description"
            name="description"
            maxLength={2000}
            value={values.description}
            onChange={(description) => set('description', description)}
            error={error('description')}
          />
        </FormSection>

        <FormSection title="Budget">
          <MoneyField
            label="Plan budget"
            name="budget"
            currency="USD"
            value={values.budget}
            onChange={(budget) => set('budget', budget)}
            error={error('budget')}
          />
          <ChoiceField
            label="Default finance budget"
            name="defaultFinanceBudgetId"
            options={budgetOptions}
            value={values.defaultFinanceBudgetId || NO_DEFAULT}
            onChange={(id) => {
              setBudgetNote(null);
              set('defaultFinanceBudgetId', id === NO_DEFAULT ? '' : id);
            }}
            note={
              [
                budgetNote,
                noBudgetsYet,
                mode === 'edit' && values.defaultFinanceBudgetId !== initial.defaultFinanceBudgetId
                  ? APPLIES_FROM_NOW
                  : null,
              ]
                .filter(Boolean)
                .join(' ') || undefined
            }
            error={error('defaultFinanceBudgetId')}
          />
          <ChoiceField
            label="Default line-item spread"
            name="defaultSpread"
            hint={defaultHint('defaultSpread', 'Default: Evenly')}
            options={(Object.keys(SPREAD_LABELS) as SpreadMethod[]).map((id) => ({
              id,
              label: SPREAD_LABELS[id],
              active: true,
            }))}
            value={values.defaultSpread}
            onChange={(spread) =>
              set('defaultSpread', (spread || values.defaultSpread) as SpreadMethod)
            }
            note={
              mode === 'edit' && values.defaultSpread !== initial.defaultSpread
                ? APPLIES_FROM_NOW
                : undefined
            }
            error={error('defaultSpread')}
          />
          <CheckboxField
            label="BCPIR plan"
            name="bcpir"
            hint={defaultHint('bcpir', 'Default: No')}
            checked={values.bcpir}
            onChange={(bcpir) => set('bcpir', bcpir)}
          />
        </FormSection>

        <FormSection title="Shell attributes">
          <ShellAttributeFields
            form={shellForm}
            values={values.attributes}
            onChange={(attributes) => set('attributes', attributes)}
            businessUnit={null}
            previousValues={plan?.attributes}
            errors={attributeErrors}
          />
        </FormSection>

        <FormSection title="Goals">
          <GoalFields
            stage={values.goalStage}
            amount={values.goalAmount}
            onStageChange={(stage) => set('goalStage', stage)}
            onAmountChange={(amount) => set('goalAmount', amount)}
            model={chosenModel ?? null}
            stageHint={defaultHint('goalStage', 'Default: INQ')}
            stageError={error('goalStage')}
            amountError={error('goalAmount')}
          />
        </FormSection>

        <FormFooter>
          <Button type="submit" variant="primary">
            Save
          </Button>
          <Button onClick={() => confirmDiscard(() => void navigate(cancelTo))}>Cancel</Button>
        </FormFooter>
      </form>
      <ClearValuesDialog
        open={confirmClear}
        onOpenChange={setConfirmClear}
        cleared={valuesToClear(shellForm, values.attributes, plan?.attributes ?? {})}
        onConfirm={() => void save()}
        busy={saving}
      />
      {dialog}
    </>
  );
}

/** The form's shape while its data loads. */
export function PlanFormSkeleton() {
  return (
    <div className={styles.form} aria-busy="true" aria-label="Loading the form">
      {[0, 1, 2, 3].map((section) => (
        <div key={section} className={styles.skeletonSection}>
          <Skeleton width="160px" height="18px" />
          <Skeleton height="38px" />
          <Skeleton height="38px" />
        </div>
      ))}
    </div>
  );
}
