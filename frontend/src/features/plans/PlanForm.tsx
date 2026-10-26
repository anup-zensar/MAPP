import { useEffect, useRef, useState } from 'react';
import type { FormEvent, MouseEvent, ReactNode } from 'react';
import { useNavigate } from 'react-router';
import { useAttributeForm, useCurrencies, useFinanceBudgets, useModels } from '@/features/settings';
import { Button } from '@/ui/Button/Button';
import { CheckboxField } from '@/ui/CheckboxField/CheckboxField';
import { ChoiceField } from '@/ui/ChoiceField/ChoiceField';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { FormErrorSummary } from '@/ui/FormErrorSummary/FormErrorSummary';
import { FormFooter } from '@/ui/FormFooter/FormFooter';
import { FailureMessage } from '@/ui/FormPanel/FormPanel';
import { problemOf } from '@/ui/FormPanel/problemOf';
import { FormSection } from '@/ui/FormSection/FormSection';
import { LockedNote } from '@/ui/LockedNote/LockedNote';
import { MoneyField } from '@/ui/MoneyField/MoneyField';
import { ClearValuesDialog } from '@/ui/ShellAttributeFields/ClearValuesDialog';
import { ShellAttributeFields } from '@/ui/ShellAttributeFields/ShellAttributeFields';
import {
  businessUnitOf,
  missingAttributes,
  valuesToClear,
  visibleAttributes,
} from '@/ui/ShellAttributeFields/rules';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { TextAreaField } from '@/ui/TextAreaField/TextAreaField';
import { TextField } from '@/ui/TextField/TextField';
import type { FieldProblem } from '@/ui/FormErrorSummary/FormErrorSummary';
import type { SaveFailure } from '@/ui/FormPanel/problemOf';
import type { AttributeForm } from '@/ui/ShellAttributeFields/rules';
import { PersonField, TeamField, usePeople } from '@/features/people';
import { formatAmount, formatMoneyText, formatPercent } from '@/format/decimal';
import { useUnsavedChanges } from '@/ui/useUnsavedChanges';
import { GoalFields } from './GoalFields';
import { PLAN_TYPE_LABELS, PLAN_TYPES } from './planCurrency';
import type { PlanType } from './planCurrency';
import { answers016, sameValues } from './planFormValues';
import type { PlanFormValues } from './planFormValues';
import styles from './PlanForm.module.css';
import type { Plan, PlanDefaults } from './plansApi';
import {
  allocationLockText,
  APPLIES_FROM_NOW,
  PHASING_REMOVED,
  PUBLISHED_BANNER,
  SPREAD_LABELS,
  YEAR_LOCKED,
} from './planText';
import type { SpreadMethod } from './planText';

/** The choice that stands for "no default finance budget" (ChoiceField's empty entry is "Choose…"). */
const NO_DEFAULT = 'none';

/** The form's sections, in the order of design system §7, with the ids the rail links to. */
const SECTIONS = [
  { id: 'plan-basics', title: 'Basics' },
  { id: 'plan-budget', title: 'Budget' },
  { id: 'plan-attributes', title: 'Shell attributes' },
  { id: 'plan-goals', title: 'Goals' },
] as const;
type SectionId = (typeof SECTIONS)[number]['id'];

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
  const with016 = answers016(mode === 'create' ? defaults : plan);
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

  // 014: an allocated plan's budget is set by its global budget and can't be typed here.
  const allocation = plan?.allocation ?? null;
  const allocationLock = allocation
    ? allocationLockText(allocation, plan?.access?.editBudget.reason)
    : null;

  const ownerName =
    people.data?.items.find((person) => person.id === values.ownerId)?.name ??
    (plan?.owner?.id === values.ownerId ? plan.owner.name : 'The owner');

  const needed = neededFields(values, shellForm, with016);
  // One attribute section named like the card itself shows no second heading (UI audit F-03).
  const attributeHeading =
    shellForm.sections.length === 1 && shellForm.sections[0].name === 'Shell attributes'
      ? 'none'
      : 'h3';

  return (
    <>
      <form noValidate aria-busy={saving} className={styles.layout} onSubmit={submit}>
        <div className={styles.main}>
          <p className={styles.progressLine}>{progressText(needed)}</p>
          {mode === 'edit' && plan?.status === 'published' && (
            <p role="note" className={styles.banner}>
              {PUBLISHED_BANNER}
            </p>
          )}
          {failure && <FailureMessage failure={failure} />}
          <FormErrorSummary errors={problems} />

          <Card id="plan-basics" title="Basics">
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
              <ReadOnlyField label="Year" marking="Required" value={values.year}>
                {YEAR_LOCKED}
              </ReadOnlyField>
            )}
            {with016 && (
              <ChoiceField
                label="Plan type"
                name="planType"
                marking="Required"
                hint={defaultHint('planType', 'Default: Operational')}
                options={PLAN_TYPES.map((type) => ({
                  id: type,
                  label: PLAN_TYPE_LABELS[type],
                  active: true,
                }))}
                value={values.planType}
                onChange={(type) => set('planType', (type || values.planType) as PlanType)}
                error={error('planType')}
              />
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
          </Card>

          <Card id="plan-budget" title="Budget">
            {with016 && (
              <CurrencyField
                value={values.currency}
                initial={initial.currency}
                hint={defaultHint('currency', 'Default: USD')}
                lock={plan?.currencyLock ?? undefined}
                error={error('currency')}
                onChange={(currency) => set('currency', currency || values.currency)}
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
              <MoneyField
                label="Plan budget"
                name="budget"
                currency={values.currency}
                value={values.budget}
                onChange={(budget) => set('budget', budget)}
                note={budgetFieldNote(values, initial, plan) ?? undefined}
                error={error('budget')}
              />
            )}
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
                  mode === 'edit' &&
                  values.defaultFinanceBudgetId !== initial.defaultFinanceBudgetId
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
          </Card>

          <Card id="plan-attributes" title="Shell attributes">
            <ShellAttributeFields
              form={shellForm}
              values={values.attributes}
              onChange={(attributes) => set('attributes', attributes)}
              businessUnit={null}
              previousValues={plan?.attributes}
              errors={attributeErrors}
              sectionHeading={attributeHeading}
            />
          </Card>

          <Card id="plan-goals" title="Goals">
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
          </Card>
        </div>

        <nav className={styles.rail} aria-label="On this page">
          <p className={styles.railTitle}>On this page</p>
          <ul className={styles.railList}>
            {SECTIONS.map((section) => {
              const left = needed.bySection[section.id].missing;
              return (
                <li key={section.id}>
                  <a href={`#${section.id}`} className={styles.railLink} onClick={jumpTo}>
                    <span>{section.title}</span>
                    <span className={left > 0 ? styles.railMissing : styles.railDone}>
                      {left > 0 ? `${left} needed` : 'Done'}
                    </span>
                  </a>
                </li>
              );
            })}
          </ul>
          <p className={styles.railCount}>{progressText(needed)}</p>
        </nav>

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

/**
 * The plan currency (016 FR-011, FR-012): the active currencies, the plan's own kept even when it
 * was set inactive since; locked with the API's reason once amounts are kept in it.
 */
function CurrencyField({
  value,
  initial,
  hint,
  lock,
  error,
  onChange,
}: {
  value: string;
  initial: string;
  hint?: string;
  lock?: string;
  error?: string;
  onChange: (currency: string) => void;
}) {
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
      name="currency"
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

/** One section of the form as a white card with an anchor for the rail (design system §5.5). */
function Card({ id, title, children }: { id: SectionId; title: string; children: ReactNode }) {
  return (
    <div id={id} className={styles.card}>
      <FormSection title={title}>{children}</FormSection>
    </div>
  );
}

/** A field shown but not editable: its label, the value and the lock with the reason (§6.6). */
function ReadOnlyField({
  label,
  marking,
  value,
  children,
}: {
  label: string;
  marking?: string;
  value: string;
  children: ReactNode;
}) {
  return (
    <div className={styles.readOnly}>
      <p className={styles.readOnlyLabel}>
        {label} {marking && <span className={styles.marking}>{marking}</span>}
      </p>
      <p className={styles.readOnlyValue}>{value}</p>
      <LockedNote>{children}</LockedNote>
    </div>
  );
}

/** The rail's link: moves the focus to the section's first field as well as scrolling to it. */
function jumpTo(event: MouseEvent<HTMLAnchorElement>) {
  const id = event.currentTarget.getAttribute('href')?.slice(1);
  const section = id ? document.getElementById(id) : null;
  if (!section) {
    return;
  }
  event.preventDefault();
  section.scrollIntoView?.({ block: 'start' });
  section.querySelector<HTMLElement>('input, select, textarea, button')?.focus({
    preventScroll: true,
  });
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

interface NeededCount {
  missing: number;
  total: number;
}

/** The needed fields of each section and how many are still empty (design system §2.4 C, §7). */
function neededFields(
  values: PlanFormValues,
  form: AttributeForm,
  with016: boolean,
): { filled: number; total: number; bySection: Record<SectionId, NeededCount> } {
  const count = (checks: boolean[]): NeededCount => ({
    missing: checks.filter((filled) => !filled).length,
    total: checks.length,
  });
  const units = businessUnitOf(form, values.attributes, null);
  const attributesTotal = visibleAttributes(form, units).filter(
    (attribute) =>
      attribute.active && attribute.marking !== 'optional' && attribute.type !== 'yesNo',
  ).length;
  const attributesMissing = missingAttributes(form, values.attributes, units).length;
  const bySection: Record<SectionId, NeededCount> = {
    'plan-basics': count([
      values.name.trim() !== '',
      values.year !== '',
      values.modelId !== '',
      values.teamIds.length > 0,
      values.ownerId !== '',
    ]),
    'plan-budget': count(with016 ? [values.currency !== ''] : []),
    'plan-attributes': {
      missing: Math.min(attributesMissing, attributesTotal),
      total: attributesTotal,
    },
    'plan-goals': count([values.goalStage !== undefined]),
  };
  const sections = Object.values(bySection);
  const total = sections.reduce((sum, section) => sum + section.total, 0);
  const missing = sections.reduce((sum, section) => sum + section.missing, 0);
  return { filled: total - missing, total, bySection };
}

/** "9 of 15 needed fields filled". */
function progressText({ filled, total }: { filled: number; total: number }): string {
  return `${filled} of ${total} needed fields filled`;
}

/** The form's shape while its data loads. */
export function PlanFormSkeleton() {
  return (
    <div className={styles.main} aria-busy="true" aria-label="Loading the form">
      {[0, 1, 2, 3].map((section) => (
        <div key={section} className={styles.card}>
          <div className={styles.skeletonSection}>
            <Skeleton width="160px" height="18px" />
            <Skeleton height="38px" />
            <Skeleton height="38px" />
          </div>
        </div>
      ))}
    </div>
  );
}
