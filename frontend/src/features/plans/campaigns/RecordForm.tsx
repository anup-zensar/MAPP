import { useId } from 'react';
import { useWatch } from 'react-hook-form';
import type { Control } from 'react-hook-form';
import { DateRangeField } from '@/ui/DateRangeField/DateRangeField';
import { FormField } from '@/ui/FormField/FormField';
import { FormSection } from '@/ui/FormSection/FormSection';
import { MoneyField } from '@/ui/MoneyField/MoneyField';
import { ShellAttributeFields } from '@/ui/ShellAttributeFields/ShellAttributeFields';
import { TextAreaField } from '@/ui/TextAreaField/TextAreaField';
import { TextField } from '@/ui/TextField/TextField';
import type { FieldProblem } from '@/ui/FormErrorSummary/FormErrorSummary';
import type { AttributeForm, AttributeValues } from '@/ui/ShellAttributeFields/rules';
import { budgetCheck, OverspendField } from '@/features/overspend';
import { PersonField } from '@/features/people';
import { useFormatters } from '@/format/useFormatters';
import { PHASING_REMOVED } from '../plan-text';
import { RemainingBudgetLine } from './BudgetCost';
import type { Money } from './api';
import { attributeErrors, outsideYear } from './form-values';
import styles from './RecordForm.module.css';
import type { RecordFormValues } from './schemas';
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

export interface FormDefaults {
  ownerId: string;
  startDate: string;
  endDate: string;
}

export interface ParentCampaign {
  name: string;
  startDate: string | null;
  endDate: string | null;
}

export interface RecordFormProps {
  level: RecordLevel;
  control: Control<RecordFormValues>;
  problems: FieldProblem[];
  attributeForm: AttributeForm;
  businessUnit: string[];
  previousAttributes?: AttributeValues;
  planId: string;
  planYear: number;
  campaign?: ParentCampaign;
  defaults?: FormDefaults;
  parentRemaining: Money | null;
  savedBudget: Money | null;
  currency: string;
  phased?: boolean;
  overspendLock?: string;
  planned?: Money | null;
}

export function RecordForm({
  level,
  control,
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
  overspendLock = '',
  planned = null,
}: RecordFormProps) {
  const { date } = useFormatters();
  const campaignFieldId = useId();
  const values = useWatch({ control, compute: (all: RecordFormValues) => all });
  const isCampaign = level === 'campaign';
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
        <FormField
          control={control}
          name="name"
          render={(field) => (
            <TextField
              label={NAME_LABEL[level]}
              marking="Required"
              maxLength={120}
              {...field}
              name={`${level}-name`}
            />
          )}
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
        <FormField
          control={control}
          name="ownerId"
          render={(field) => (
            <PersonField
              label={OWNER}
              marking="Needed to publish"
              hint={ownerDefaulted ? (isCampaign ? FROM_THE_PLAN : FROM_THE_CAMPAIGN) : undefined}
              {...field}
              name={`${level}-ownerId`}
            />
          )}
        />
        <FormField
          control={control}
          name="startDate"
          render={(start) => (
            <FormField
              control={control}
              name="endDate"
              render={(end) => (
                <DateRangeField
                  label={DATES}
                  idPrefix={level}
                  marking="Needed to publish"
                  hint={
                    datesDefaulted ? (isCampaign ? THE_PLANS_YEAR : THE_CAMPAIGNS_DATES) : undefined
                  }
                  start={start.value}
                  end={end.value}
                  onChange={(range) => {
                    start.onChange(range.start);
                    end.onChange(range.end);
                  }}
                  allowed={allowed}
                  allowedHint={
                    allowed
                      ? allowedRange(`${date(allowed.start)} – ${date(allowed.end)}`)
                      : undefined
                  }
                  errors={{ start: start.error, end: end.error }}
                  warning={
                    isCampaign && outsideYear(start.value, end.value, planYear)
                      ? outsidePlanYear(planYear)
                      : undefined
                  }
                />
              )}
            />
          )}
        />
        <FormField
          control={control}
          name="description"
          render={(field) => (
            <TextAreaField
              label={DESCRIPTION}
              maxLength={2000}
              {...field}
              name={`${level}-description`}
            />
          )}
        />
      </FormSection>
      <FormSection title={BUDGET}>
        <FormField
          control={control}
          name="budget"
          render={(field) => (
            <MoneyField
              label={BUDGET_LABEL[level]}
              currency={currency}
              note={
                phased && savedBudget && field.value.trim() === '' ? PHASING_REMOVED : undefined
              }
              {...field}
              name={`${level}-budget`}
            />
          )}
        />
        <RemainingBudgetLine
          label={isCampaign ? REMAINING_PLAN_BUDGET : REMAINING_CAMPAIGN_BUDGET}
          tip={isCampaign ? REMAINING_PLAN_BUDGET_TIP : REMAINING_CAMPAIGN_BUDGET_TIP}
          parentRemaining={parentRemaining}
          savedOwn={savedBudget}
          typed={values.budget}
        />
        <FormField
          control={control}
          name="overspend"
          render={(field) => (
            <OverspendField
              name={`${level}-overspend`}
              value={field.value}
              onChange={field.onChange}
              lockedReason={overspendLock}
              check={budgetCheck({
                mode: field.value,
                typed: values.budget,
                stored: savedBudget?.amount ?? null,
                planned,
              })}
            />
          )}
        />
      </FormSection>
      <FormField
        control={control}
        name="attributes"
        render={(field) => (
          <ShellAttributeFields
            form={attributeForm}
            values={field.value}
            onChange={field.onChange}
            businessUnit={businessUnit}
            previousValues={previousAttributes}
            errors={attributeErrors(problems)}
            planLink={`/plans/${planId}`}
            sectionHeading="h2"
          />
        )}
      />
    </>
  );
}
