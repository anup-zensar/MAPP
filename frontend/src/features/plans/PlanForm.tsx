import { zodResolver } from '@hookform/resolvers/zod';
import { useEffect, useRef, useState } from 'react';
import { useForm, useWatch } from 'react-hook-form';
import { useNavigate } from 'react-router';
import { useAttributeForm, useFinanceBudgets, useModels } from '@/features/settings';
import { Button } from '@/ui/Button/Button';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { FormErrorSummary } from '@/ui/FormErrorSummary/FormErrorSummary';
import { FormField } from '@/ui/FormField/FormField';
import { FormFooter } from '@/ui/FormFooter/FormFooter';
import { FailureMessage } from '@/ui/FormPanel/FormPanel';
import { ClearValuesDialog } from '@/ui/ShellAttributeFields/ClearValuesDialog';
import { ShellAttributeFields } from '@/ui/ShellAttributeFields/ShellAttributeFields';
import { valuesToClear } from '@/ui/ShellAttributeFields/rules';
import type { FieldProblem } from '@/ui/FormErrorSummary/FormErrorSummary';
import type { SaveFailure } from '@/ui/FormPanel/problem-of';
import { usePeople } from '@/features/people';
import { CHECK_ON_SAVE, useFormProblems } from '@/ui/useFormProblems';
import { useUnsavedChanges } from '@/ui/useUnsavedChanges';
import { PlanBasicsSection } from './PlanBasicsSection';
import { PlanBudgetSection } from './PlanBudgetSection';
import styles from './PlanForm.module.css';
import { PlanFormCard } from './PlanFormCard';
import { PlanFormRail } from './PlanFormRail';
import { PlanFormSkeleton } from './PlanFormSkeleton';
import { PlanGoalsSection } from './PlanGoalsSection';
import type { Plan, PlanDefaults } from './api';
import { neededFields, progressText } from './plan-form-sections';
import type { DefaultHint } from './plan-form-sections';
import { answers016, sameValues } from './plan-form-values';
import { PUBLISHED_BANNER } from './plan-text';
import { planFormSchema } from './schemas';
import type { PlanFormValues } from './schemas';

/** The fields that show the API's problems under them, in the form's order. */
const FIELDS = [
  'name',
  'year',
  'planType',
  'modelId',
  'teamIds',
  'ownerId',
  'description',
  'currency',
  'budget',
  'defaultFinanceBudgetId',
  'defaultSpread',
  'goalStage',
  'goalAmount',
] as const;

/** What a save answers: the page to open once saved, or the API's field problems; null for neither. */
export type PlanSaveAnswer = { to: string } | { problems: FieldProblem[] } | null;

export interface PlanFormProps {
  mode: 'create' | 'edit';
  /** The values the form starts with; copied once (edit-form rule). */
  initial: PlanFormValues;
  /** Create: the defaults, for the years to choose from and the "Default: …" hints (FR-006). */
  defaults?: PlanDefaults;
  /** Edit: the plan as it was when the form opened. */
  plan?: Plan;
  /** A failure that is not about one field (a conflict, the server unreachable). */
  failure: SaveFailure | null;
  onSave: (values: PlanFormValues) => Promise<PlanSaveAnswer>;
  /** Where Cancel leads: the list or the plan page. */
  cancelTo: string;
  /** The field to focus once the form is drawn (`?focus=` from completeness links). */
  focusField?: string | null;
}

/**
 * The plan form of the create and edit pages (plan "Plan form", FR-005 – FR-010, FR-018 – FR-023)
 * on the form template (design system §2.4 C): one white card per section — Basics, Budget, Shell
 * attributes, Goals — in a column of at most 720 px, and from 1,280 px a rail that links the
 * sections and counts the needed fields still empty. Defaults are labelled, markings are words,
 * every problem shows at once, Save and Cancel stay at the bottom of the window, and "Discard
 * changes?" comes before typed values are lost. With 016 the plan type and the plan currency
 * join Basics and Budget; with 014 an allocated plan's budget is read-only with its reason.
 */
export function PlanForm({
  mode,
  initial,
  defaults,
  plan,
  failure,
  onSave,
  cancelTo,
  focusField,
}: PlanFormProps) {
  const navigate = useNavigate();
  const models = useModels();
  const attributeForm = useAttributeForm('plan');
  const people = usePeople();
  const form = useForm<PlanFormValues>({
    resolver: zodResolver(planFormSchema),
    defaultValues: initial,
    ...CHECK_ON_SAVE,
  });
  const { problems, onInvalid, showServerProblems } = useFormProblems(form, FIELDS);
  const values = useWatch({ control: form.control, compute: (all: PlanFormValues) => all });
  const [saving, setSaving] = useState(false);
  const [leaveTo, setLeaveTo] = useState<string | null>(null);
  const [budgetNote, setBudgetNote] = useState<string | null>(null);
  const [confirmClear, setConfirmClear] = useState(false);
  const financeBudgets = useFinanceBudgets(Number(values.year) || undefined);
  const with016 = answers016(mode === 'create' ? defaults : plan);
  const { dialog, confirmDiscard } = useUnsavedChanges(!leaveTo && form.formState.isDirty);
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
  const previousAttributes = plan?.attributes ?? {};

  const hint: DefaultHint = (key, text, given = true) =>
    mode === 'create' && given && sameValues({ ...values, [key]: initial[key] }, values)
      ? text
      : undefined;

  function changeYear(year: string) {
    const chosen = financeBudgets.data?.items.find(
      (budget) => budget.id === values.defaultFinanceBudgetId,
    );
    if (chosen && String(chosen.year) !== year) {
      setBudgetNote(`Default finance budget cleared — '${chosen.name}' belongs to ${chosen.year}.`);
      form.setValue('defaultFinanceBudgetId', '', { shouldDirty: true });
    }
  }

  async function save(saved: PlanFormValues) {
    setConfirmClear(false);
    setSaving(true);
    try {
      const answer = await onSave(saved);
      if (answer && 'to' in answer) {
        setLeaveTo(answer.to);
      } else if (answer) {
        showServerProblems(answer.problems);
      }
    } finally {
      setSaving(false);
    }
  }

  const submit = form.handleSubmit((typed) => {
    if (valuesToClear(shellForm, typed.attributes, previousAttributes).length > 0) {
      setConfirmClear(true);
      return;
    }
    void save(typed);
  }, onInvalid);

  const attributeErrors = Object.fromEntries(
    problems
      .filter((problem) => problem.field.startsWith('attributes.'))
      .map((problem) => [problem.field, problem.message]),
  );
  const modelItems = models.data.items;
  const chosenModel =
    modelItems.find((model) => model.id === values.modelId) ??
    (plan && plan.model.id === values.modelId ? plan.model : undefined);
  const ownerName =
    people.data?.items.find((person) => person.id === values.ownerId)?.name ??
    (plan?.owner?.id === values.ownerId ? plan.owner.name : 'The owner');
  const needed = neededFields(values, shellForm, with016);
  // One attribute section named like the card itself shows no second heading (UI audit F-03).
  const attributeHeading =
    shellForm.sections.length === 1 && shellForm.sections[0]?.name === 'Shell attributes'
      ? 'none'
      : 'h3';
  const section = { control: form.control, values, mode, plan, with016, hint };

  return (
    <>
      <form
        noValidate
        aria-busy={saving}
        className={styles.layout}
        onSubmit={(event) => {
          event.preventDefault();
          // A second Save while saving is ignored; Save is never greyed out.
          if (!saving) {
            void submit(event);
          }
        }}
      >
        <div className={styles.main}>
          <p className={styles.progressLine}>{progressText(needed)}</p>
          {mode === 'edit' && plan?.status === 'published' && (
            <p role="note" className={styles.banner}>
              {PUBLISHED_BANNER}
            </p>
          )}
          {failure && <FailureMessage failure={failure} />}
          <FormErrorSummary errors={problems} />
          <PlanBasicsSection
            {...section}
            defaults={defaults}
            models={modelItems}
            chosenModel={chosenModel}
            ownerName={ownerName}
            onYearChange={changeYear}
          />
          <PlanBudgetSection
            {...section}
            initial={initial}
            financeBudgets={financeBudgets}
            budgetNote={budgetNote}
            onFinanceBudgetChange={() => setBudgetNote(null)}
          />
          <PlanFormCard id="plan-attributes" title="Shell attributes">
            <FormField
              control={form.control}
              name="attributes"
              render={(field) => (
                <ShellAttributeFields
                  form={shellForm}
                  values={field.value}
                  onChange={field.onChange}
                  businessUnit={null}
                  previousValues={plan?.attributes}
                  errors={attributeErrors}
                  sectionHeading={attributeHeading}
                />
              )}
            />
          </PlanFormCard>
          <PlanGoalsSection control={form.control} model={chosenModel ?? null} hint={hint} />
        </div>
        <PlanFormRail needed={needed} />
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
        cleared={valuesToClear(shellForm, values.attributes, previousAttributes)}
        onConfirm={() => void save(form.getValues())}
        busy={saving}
      />
      {dialog}
    </>
  );
}
