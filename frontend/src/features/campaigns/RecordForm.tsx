import { useId } from 'react';
import {
  DateRangeField,
  FormSection,
  MoneyField,
  problemOf,
  ShellAttributeFields,
  TextAreaField,
  TextField,
} from '@/components';
import type { AttributeForm, AttributeValues, FieldProblem } from '@/components';
import { PersonField } from '@/features/people';
import { useFormatters } from '@/lib/useFormatters';
import { PHASING_REMOVED } from '@/features/plans/planText';
import { RemainingBudgetLine } from './BudgetCost';
import type { Money } from './campaignApi';
import { attributeErrors, outsideYear } from './formValues';
import type { RecordFormValues } from './formValues';
import styles from './RecordForm.module.css';
import {
  allowedRange,
  BASICS,
  BUDGET,
  BUDGET_LABEL,
  CAMPAIGN,
  CHANGE_IT_WITH_MOVE,
  DATES,
  DESCRIPTION,
  FROM_THE_CAMPAIGN,
  FROM_THE_PLAN,
  NAME_LABEL,
  outsidePlanYear,
  OWNER,
  REMAINING_CAMPAIGN_BUDGET,
  REMAINING_CAMPAIGN_BUDGET_TIP,
  REMAINING_PLAN_BUDGET,
  REMAINING_PLAN_BUDGET_TIP,
  THE_CAMPAIGNS_DATES,
  THE_PLANS_YEAR,
} from './texts';
import type { RecordLevel } from './texts';

/** The values a create form starts with; their hints show while they are unchanged (FR-003). */
export interface FormDefaults {
  ownerId: string;
  startDate: string;
  endDate: string;
}

/** The campaign an initiative belongs to: shown read-only, its dates are the allowed range (R3). */
export interface ParentCampaign {
  name: string;
  startDate: string | null;
  endDate: string | null;
}

export interface RecordFormProps {
  level: RecordLevel;
  values: RecordFormValues;
  onChange: (values: RecordFormValues) => void;
  /** The field problems of the last save, all at once. */
  problems: FieldProblem[];
  /** The level's Shell attributes (GET /api/v1/shell-attribute-forms/{level}). */
  attributeForm: AttributeForm;
  /** The plan's Business unit option ids: they decide the conditional attributes. */
  businessUnit: string[];
  /** The stored attribute values (edit forms). */
  previousAttributes?: AttributeValues;
  planId: string;
  /** The plan's year, for the R4 warning (campaigns). */
  planYear: number;
  /** The initiative's campaign. */
  campaign?: ParentCampaign;
  /** Create forms: the pre-filled values and their hints. */
  defaults?: FormDefaults;
  /** The parent's remaining budget as saved, and this record's saved budget (FR-010). */
  parentRemaining: Money | null;
  savedBudget: Money | null;
  /** The plan currency: the budget is typed in it (016 FR-011). */
  currency: string;
  /** Edit forms: the saved budget has a Budget by month, which clearing the budget removes. */
  phased?: boolean;
}

/**
 * The campaign or initiative form (plan "Forms", FR-002): Basics, Budget and the level's Shell
 * attributes from Settings, one column, markings as words, every problem under its field. Field
 * ids are "<level>-<field>" (attributes: "attributes.<id>"), as the error summary links them.
 */
export function RecordForm({
  level,
  values,
  onChange,
  problems,
  attributeForm,
  businessUnit,
  previousAttributes,
  planId,
  planYear,
  campaign,
  defaults,
  parentRemaining,
  savedBudget,
  currency,
  phased = false,
}: RecordFormProps) {
  const { date } = useFormatters();
  const campaignFieldId = useId();
  const isCampaign = level === 'campaign';

  function set<K extends keyof RecordFormValues>(field: K, value: RecordFormValues[K]) {
    onChange({ ...values, [field]: value });
  }

  const ownerDefaulted = defaults && defaults.ownerId !== '' && values.ownerId === defaults.ownerId;
  const datesDefaulted =
    defaults &&
    defaults.startDate !== '' &&
    values.startDate === defaults.startDate &&
    values.endDate === defaults.endDate;
  const allowed =
    campaign?.startDate && campaign.endDate
      ? { start: campaign.startDate, end: campaign.endDate }
      : null;

  return (
    <>
      <FormSection title={BASICS}>
        <TextField
          label={NAME_LABEL[level]}
          name={`${level}-name`}
          marking="Required"
          maxLength={120}
          value={values.name}
          onChange={(name) => set('name', name)}
          error={problemOf(problems, 'name')}
        />
        {campaign && (
          <div className={styles.readOnly}>
            <label htmlFor={campaignFieldId} className={styles.readOnlyLabel}>
              {CAMPAIGN}
            </label>
            <input
              id={campaignFieldId}
              type="text"
              readOnly
              value={campaign.name}
              aria-describedby={`${campaignFieldId}-hint`}
              className={styles.readOnlyInput}
            />
            <p id={`${campaignFieldId}-hint`} className={styles.hint}>
              {CHANGE_IT_WITH_MOVE}
            </p>
          </div>
        )}
        <PersonField
          label={OWNER}
          name={`${level}-ownerId`}
          marking="Needed to publish"
          hint={ownerDefaulted ? (isCampaign ? FROM_THE_PLAN : FROM_THE_CAMPAIGN) : undefined}
          value={values.ownerId}
          onChange={(ownerId) => set('ownerId', ownerId)}
          error={problemOf(problems, 'ownerId')}
        />
        <DateRangeField
          label={DATES}
          idPrefix={level}
          marking="Needed to publish"
          hint={datesDefaulted ? (isCampaign ? THE_PLANS_YEAR : THE_CAMPAIGNS_DATES) : undefined}
          start={values.startDate}
          end={values.endDate}
          onChange={({ start, end }) => onChange({ ...values, startDate: start, endDate: end })}
          allowed={allowed}
          allowedHint={
            allowed ? allowedRange(`${date(allowed.start)} – ${date(allowed.end)}`) : undefined
          }
          errors={{
            start: problemOf(problems, 'startDate'),
            end: problemOf(problems, 'endDate'),
          }}
          warning={
            isCampaign && outsideYear(values.startDate, values.endDate, planYear)
              ? outsidePlanYear(planYear)
              : undefined
          }
        />
        <TextAreaField
          label={DESCRIPTION}
          name={`${level}-description`}
          maxLength={2000}
          value={values.description}
          onChange={(description) => set('description', description)}
          error={problemOf(problems, 'description')}
        />
      </FormSection>
      <FormSection title={BUDGET}>
        <MoneyField
          label={BUDGET_LABEL[level]}
          name={`${level}-budget`}
          currency={currency}
          value={values.budget}
          onChange={(budget) => set('budget', budget)}
          note={phased && savedBudget && values.budget.trim() === '' ? PHASING_REMOVED : undefined}
          error={problemOf(problems, 'budget')}
        />
        <RemainingBudgetLine
          label={isCampaign ? REMAINING_PLAN_BUDGET : REMAINING_CAMPAIGN_BUDGET}
          tip={isCampaign ? REMAINING_PLAN_BUDGET_TIP : REMAINING_CAMPAIGN_BUDGET_TIP}
          parentRemaining={parentRemaining}
          savedOwn={savedBudget}
          typed={values.budget}
        />
      </FormSection>
      <ShellAttributeFields
        form={attributeForm}
        values={values.attributes}
        onChange={(attributes) => set('attributes', attributes)}
        businessUnit={businessUnit}
        previousValues={previousAttributes}
        errors={attributeErrors(problems)}
        planLink={`/plans/${planId}`}
        sectionHeading="h2"
      />
    </>
  );
}
