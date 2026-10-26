import { TriangleAlert } from 'lucide-react';
import { useState } from 'react';
import { Link } from 'react-router';
import type { components } from '@/api/generated/schema';
import { CheckboxField } from '@/ui/CheckboxField/CheckboxField';
import { ChoiceField } from '@/ui/ChoiceField/ChoiceField';
import { ConfirmDialog } from '@/ui/ConfirmDialog/ConfirmDialog';
import { DateRangeField } from '@/ui/DateRangeField/DateRangeField';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { problemOf } from '@/ui/FormPanel/problem-of';
import { LockedNote } from '@/ui/LockedNote/LockedNote';
import { MoneyField } from '@/ui/MoneyField/MoneyField';
import { ShellAttributeFields } from '@/ui/ShellAttributeFields/ShellAttributeFields';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { TextAreaField } from '@/ui/TextAreaField/TextAreaField';
import { TextField } from '@/ui/TextField/TextField';
import type { FieldProblem } from '@/ui/FormErrorSummary/FormErrorSummary';
import type { AttributeForm, AttributeValues } from '@/ui/ShellAttributeFields/rules';
import { PersonField } from '@/features/people';
import { SpreadEffectNote } from '@/features/line-items';
import { formatAmount, formatMoneyText } from '@/format/decimal';
import { recordHref } from '@/ui/record-links';
import { useFormatters } from '@/format/useFormatters';
import type { ActivityFormData, ActivityTypeOption, Money } from './api';
import { fieldId, usdOf } from './activity-form-state';
import type { FormState } from './activity-form-state';
import styles from './ActivityForm.module.css';
import { ACTIVITY_SECTION } from './activity-form-progress';
import { FormCard, FormCardFrame } from '@/ui/FormRail/FormRail';
import { GoalsFields } from './GoalsFields';

type ActivityStatus = components['schemas']['ActivityStatus'];

/**
 * The edited activity as it is stored: its status and saved ongoing decide the locks; its Shell
 * attribute values show which dependent values saving would clear.
 */
export interface StoredActivity {
  id: string;
  status: ActivityStatus;
  ongoing: boolean;
  attributes: AttributeValues;
}

export interface ActivityFormProps {
  form: ActivityFormData;
  value: FormState;
  onChange: (value: FormState) => void;
  /** The problems of the last Save, all at once. */
  problems: FieldProblem[];
  /** The edited activity; none for a new one, whose defaults are labelled while unchanged. */
  stored?: StoredActivity;
  /** The activity level's Shell attribute form (feature 002), as it loads. */
  attributeForm: AttributeFormLoad;
  /** The plan currency the planned cost is typed in (016 FR-011); USD before 016. */
  planCurrency?: string;
}

/** The Shell attribute form while it loads: its data, or the failure and how to try again. */
export interface AttributeFormLoad {
  data: AttributeForm | undefined;
  error: unknown;
  refetch: () => void;
}

const FROM_INITIATIVE = 'From the initiative';
const ONGOING_WARNING = "Ongoing can't be undone after saving.";
const ONGOING_LOCKED = "Ongoing can't be undone — to plan with an end date, create a new activity.";
const WORKFLOW_LOCKED = "It can't change while the activity is submitted or approved.";
/** While a round is open the person can recall the submission first (008 FR-013). */
const WORKFLOW_RECALL = 'Recall the activity to choose another workflow.';
const NO_WORKFLOW = 'No approval workflow is active — an administrator sets them up in Settings.';

/** A type change that would replace a planned cost already there (FR-007). */
interface TypeQuestion {
  type: ActivityTypeOption;
  amount: Money;
}

/**
 * The activity form's sections (006 plan, "Activity form page"; design system §7): Basics, Dates
 * & cost, Shell attributes, Goals and More details, each field with its marking as a word, the
 * defaults labelled with where they came from, and the locks of a submitted, approved or ongoing
 * activity with their reasons.
 */
export function ActivityForm({
  form,
  value,
  onChange,
  problems,
  stored,
  attributeForm,
  planCurrency = 'USD',
}: ActivityFormProps) {
  const { date } = useFormatters();
  // Whether the planned cost is still the default the chosen type filled in.
  const [costFromType, setCostFromType] = useState(false);
  const [question, setQuestion] = useState<TypeQuestion | null>(null);
  const { context, defaults } = form;
  const { initiative } = context;
  const error = (field: string) => problemOf(problems, field);
  const set = (change: Partial<FormState>) => onChange({ ...value, ...change });

  const ownerFromInitiative = Boolean(defaults?.owner && value.ownerId === defaults.owner.id);
  const datesFromInitiative = Boolean(
    defaults?.startDate &&
    value.startDate === defaults.startDate &&
    value.endDate === (defaults.endDate ?? '') &&
    !value.ongoing,
  );
  const onlyWorkflow = Boolean(
    defaults?.approvalWorkflowId && value.approvalWorkflowId === defaults.approvalWorkflowId,
  );
  const underApproval = stored?.status === 'submitted' || stored?.status === 'approved';
  const ongoingSaved = Boolean(stored?.ongoing);
  const hasInitiativeDates = Boolean(initiative.startDate && initiative.endDate);
  const noTypes = form.activityTypes.length === 0;

  function chooseType(typeId: string) {
    const type = form.activityTypes.find((option) => option.id === typeId);
    const amount = type?.defaultPlannedCost;
    const next = { ...value, activityTypeId: typeId };
    if (!type || !amount) {
      setCostFromType(false);
      onChange(next);
      return;
    }
    if (value.plannedCost.trim() === '' || costFromType) {
      setCostFromType(true);
      onChange({ ...next, plannedCost: formatAmount(amount.amount) });
      return;
    }
    onChange(next);
    if (usdOf(value.plannedCost)?.amount === amount.amount) {
      setCostFromType(true);
    } else {
      setQuestion({ type, amount });
    }
  }

  function applyDefaultCost() {
    if (question) {
      setCostFromType(true);
      set({ plannedCost: formatAmount(question.amount.amount) });
    }
    setQuestion(null);
  }

  const currentCost = usdOf(value.plannedCost);

  return (
    <div className={styles.form}>
      {noTypes && <NoTypesNotice model={context.model.name} />}

      <FormCard id={ACTIVITY_SECTION.basics} title="Basics">
        <TextField
          label="Activity name"
          name={fieldId('name')}
          marking="Required"
          value={value.name}
          onChange={(name) => set({ name })}
          error={error('name')}
        />
        <TextField
          label="External name"
          name={fieldId('externalName')}
          hint="Leave empty to use the activity name"
          value={value.externalName}
          onChange={(externalName) => set({ externalName })}
          error={error('externalName')}
        />
        <div className={styles.readOnly}>
          <dl className={styles.pairs}>
            <div className={styles.pair}>
              <dt>Campaign</dt>
              <dd>{context.campaign.name}</dd>
            </div>
            <div className={styles.pair}>
              <dt>Initiative</dt>
              <dd>{initiative.name}</dd>
            </div>
          </dl>
          <LockedNote>Change it later with Move</LockedNote>
        </div>
        <ChoiceField
          label="Activity type"
          name={fieldId('activityTypeId')}
          marking="Required"
          options={form.activityTypes.map((type) => ({
            id: type.id,
            label: typeLabel(type),
            active: type.active,
          }))}
          value={value.activityTypeId}
          onChange={chooseType}
          error={error('activityTypeId')}
          note={noTypes ? <NoTypesText model={context.model.name} /> : undefined}
        />
        <PersonField
          label="Owner"
          name={fieldId('ownerId')}
          marking="Needed to submit"
          hint={ownerFromInitiative ? FROM_INITIATIVE : undefined}
          value={value.ownerId}
          onChange={(ownerId) => set({ ownerId })}
          error={error('ownerId')}
        />
        <ChoiceField
          label="Approval workflow"
          name={fieldId('approvalWorkflowId')}
          marking="Needed to submit"
          hint={onlyWorkflow ? 'The only active approval workflow' : undefined}
          options={form.approvalWorkflows.map((workflow) => ({
            id: workflow.id,
            label: workflow.name,
            active: workflow.active,
          }))}
          value={value.approvalWorkflowId}
          onChange={(approvalWorkflowId) => set({ approvalWorkflowId })}
          disabledReason={
            stored?.status === 'submitted'
              ? WORKFLOW_RECALL
              : underApproval
                ? WORKFLOW_LOCKED
                : undefined
          }
          note={form.approvalWorkflows.length === 0 ? NO_WORKFLOW : undefined}
          error={error('approvalWorkflowId')}
        />
      </FormCard>

      <FormCard id={ACTIVITY_SECTION.dates} title="Dates & cost">
        <DateRangeField
          label="Start and end dates"
          idPrefix="field"
          marking="Needed to submit"
          hint={datesFromInitiative ? FROM_INITIATIVE : undefined}
          start={value.startDate}
          end={value.endDate}
          onChange={({ start, end }) => set({ startDate: start, endDate: end })}
          allowed={
            hasInitiativeDates
              ? { start: initiative.startDate!, end: initiative.endDate! }
              : undefined
          }
          allowedHint={
            hasInitiativeDates
              ? `Allowed: ${date(initiative.startDate!)} – ${date(initiative.endDate!)} (the initiative's dates)`
              : 'The initiative has no dates yet.'
          }
          maxEnd={context.latestEndDate}
          errors={{ start: error('startDate'), end: error('endDate') }}
          endDisabledReason={value.ongoing ? 'Ongoing (no end date)' : undefined}
        />
        {stored && (
          <SpreadEffectNote
            activityId={stored.id}
            startDate={value.startDate}
            endDate={value.ongoing ? '' : value.endDate}
            ongoing={value.ongoing}
          />
        )}
        <div>
          <CheckboxField
            label="Ongoing (no end date)"
            name={fieldId('ongoing')}
            checked={value.ongoing}
            onChange={(ongoing) => set(ongoing ? { ongoing, endDate: '' } : { ongoing })}
            disabledReason={ongoingSaved ? ONGOING_LOCKED : undefined}
            error={error('ongoing')}
          />
          {value.ongoing && !ongoingSaved && (
            <p role="status" className={styles.warning}>
              <TriangleAlert aria-hidden="true" />
              {ONGOING_WARNING}
            </p>
          )}
        </div>
        <MoneyField
          label="Planned cost"
          name={fieldId('plannedCost')}
          marking="Needed to submit"
          hint={
            planCurrency === 'USD'
              ? '0.00 or more'
              : `0.00 or more, in ${planCurrency}, the plan's currency`
          }
          currency={planCurrency}
          value={value.plannedCost}
          onChange={(plannedCost) => {
            setCostFromType(false);
            set({ plannedCost });
          }}
          note={costFromType ? typeDefaultNote(form, value.activityTypeId) : undefined}
          error={error('plannedCost')}
        />
      </FormCard>

      {/* The attribute groups' own names are the card's headings ("Shell attributes",
          "Lubricants attributes"), so no heading repeats its parent (UI audit AF-02, X-09). */}
      <FormCardFrame
        id={ACTIVITY_SECTION.attributes}
        label="Shell attributes"
        className={styles.attributes}
      >
        {attributeForm.data ? (
          <ShellAttributeFields
            sectionHeading="h2"
            form={attributeForm.data}
            values={value.attributes}
            onChange={(attributes) => set({ attributes })}
            businessUnit={context.businessUnits.map((unit) => unit.id)}
            previousValues={stored?.attributes ?? {}}
            errors={attributeErrors(problems)}
            planLink={recordHref('plan', context.plan.id)}
          />
        ) : attributeForm.error ? (
          <ErrorState error={attributeForm.error} onRetry={attributeForm.refetch} />
        ) : (
          <>
            <h2>Shell attributes</h2>
            <Skeleton lines={4} height="36px" />
          </>
        )}
      </FormCardFrame>

      <FormCardFrame id={ACTIVITY_SECTION.goals}>
        <GoalsFields
          value={value.goals}
          onChange={(goals) => set({ goals })}
          model={context.model}
          errors={{
            inq: error('goals.inq'),
            mql: error('goals.mql'),
            revenue: error('goals.revenue'),
          }}
        />
      </FormCardFrame>

      <FormCard id={ACTIVITY_SECTION.more} title="More details">
        <TextAreaField
          label="Description"
          name={fieldId('description')}
          maxLength={2000}
          value={value.description}
          onChange={(description) => set({ description })}
          error={error('description')}
        />
        {!stored && <p className={styles.muted}>Save the activity to add its line items</p>}
      </FormCard>

      <ConfirmDialog
        open={question !== null}
        onOpenChange={(open) => !open && setQuestion(null)}
        title={
          question
            ? `Use the default planned cost of ${formatMoneyText(question.amount)} for '${question.type.name}'?`
            : ''
        }
        confirmLabel="Use default"
        cancelLabel={`Keep ${currentCost ? formatMoneyText(currentCost) : value.plannedCost}`}
        onConfirm={applyDefaultCost}
      >
        {null}
      </ConfirmDialog>
    </div>
  );
}

/** The problems of Shell attributes by field name, as ShellAttributeFields takes them. */
function attributeErrors(problems: FieldProblem[]): Record<string, string> {
  const errors: Record<string, string> = {};
  for (const problem of problems) {
    if (problem.field.startsWith('attributes.') && !(problem.field in errors)) {
      errors[problem.field] = problem.message;
    }
  }
  return errors;
}

/**
 * Where a pre-filled planned cost came from: "From the activity type", or with a plan in another
 * currency "Default 5,000.00 USD = 3,950.00 GBP at today's rate" (016 FR-016).
 */
function typeDefaultNote(form: ActivityFormData, typeId: string): string {
  const type = form.activityTypes.find((option) => option.id === typeId);
  const usd = type?.defaultPlannedCostUsd;
  const converted = type?.defaultPlannedCost;
  if (!usd || !converted || usd.currency === converted.currency) {
    return 'From the activity type';
  }
  return `Default ${formatMoneyText(usd)} = ${formatMoneyText(converted)} at today's rate`;
}

/** "Event · default 25,000.00 USD"; "Event — not allowed by the plan's model". */
function typeLabel(type: ActivityTypeOption): string {
  const withDefault = type.defaultPlannedCost
    ? `${type.name} · default ${formatMoneyText(type.defaultPlannedCost)}`
    : type.name;
  return type.allowed ? withDefault : `${withDefault} — not allowed by the plan's model`;
}

function NoTypesText({ model }: { model: string }) {
  return (
    <>
      The model &apos;{model}&apos; has no activity types yet — an administrator adds them in
      Settings. <Link to="/settings/activity-types">Open Settings</Link>
    </>
  );
}

/** The edge case of a model without activity types, said at the top of the form too. */
function NoTypesNotice({ model }: { model: string }) {
  return (
    <p role="note" className={styles.notice}>
      <TriangleAlert aria-hidden="true" />
      <span>
        <NoTypesText model={model} />
      </span>
    </p>
  );
}
