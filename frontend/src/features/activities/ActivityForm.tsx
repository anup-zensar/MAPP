import { useState } from 'react';
import { useFormState, useWatch } from 'react-hook-form';
import type { UseFormReturn } from 'react-hook-form';
import { ConfirmDialog } from '@/ui/ConfirmDialog/ConfirmDialog';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { FormField } from '@/ui/FormField/FormField';
import { FormCard, FormCardFrame } from '@/ui/FormRail/FormRail';
import { ShellAttributeFields } from '@/ui/ShellAttributeFields/ShellAttributeFields';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { TextAreaField } from '@/ui/TextAreaField/TextAreaField';
import type { FieldProblem } from '@/ui/FormErrorSummary/FormErrorSummary';
import type { AttributeForm } from '@/ui/ShellAttributeFields/rules';
import { formatAmount, formatMoneyText } from '@/format/decimal';
import { recordHref } from '@/ui/record-links';
import type { ActivityFormData, ActivityTypeOption, Money } from './api';
import { ActivityBasicsCard } from './ActivityBasicsCard';
import { ActivityDatesCard } from './ActivityDatesCard';
import { fieldId, usdOf } from './activity-form-state';
import type { FormState, StoredActivity } from './activity-form-state';
import styles from './ActivityForm.module.css';
import { ACTIVITY_SECTION } from './activity-form-progress';
import { GoalsFields } from './GoalsFields';
import { NoTypesNotice } from './NoTypesNotice';

export interface ActivityFormProps {
  form: UseFormReturn<FormState>;
  data: ActivityFormData;
  /** The problems of the last Save, all at once: the Shell attributes read theirs here. */
  problems: FieldProblem[];
  /** The edited activity; none for a new one, whose defaults are labelled while unchanged. */
  stored?: StoredActivity;
  /** The activity level's Shell attribute form (feature 002), as it loads. */
  attributeForm: AttributeFormLoad;
  planCurrency?: string;
}

export interface AttributeFormLoad {
  data: AttributeForm | undefined;
  error: unknown;
  refetch: () => void;
}

interface TypeQuestion {
  type: ActivityTypeOption;
  amount: Money;
}

/**
 * The activity form's sections (design system §7): Basics, Dates & cost, Shell attributes, Goals and
 * More details, with the defaults' sources and the locks of a submitted, approved or ongoing activity.
 */
export function ActivityForm({
  form,
  data,
  problems,
  stored,
  attributeForm,
  planCurrency = 'USD',
}: ActivityFormProps) {
  const value = useWatch({ control: form.control, compute: (values) => values });
  const { errors } = useFormState({ control: form.control });
  // Whether the planned cost is still the default the chosen type filled in.
  const [costFromType, setCostFromType] = useState(false);
  const [question, setQuestion] = useState<TypeQuestion | null>(null);
  const { context } = data;

  function setPlannedCost(amount: Money) {
    setCostFromType(true);
    form.setValue('plannedCost', formatAmount(amount.amount), { shouldDirty: true });
  }

  function chooseType(typeId: string) {
    const type = data.activityTypes.find((option) => option.id === typeId);
    const amount = type?.defaultPlannedCost;
    form.setValue('activityTypeId', typeId, { shouldDirty: true });
    if (!type || !amount) {
      setCostFromType(false);
    } else if (value.plannedCost.trim() === '' || costFromType) {
      setPlannedCost(amount);
    } else if (usdOf(value.plannedCost)?.amount === amount.amount) {
      setCostFromType(true);
    } else {
      setQuestion({ type, amount });
    }
  }

  const currentCost = usdOf(value.plannedCost);
  const attributes = attributeForm.data;

  return (
    <div className={styles.form}>
      {data.activityTypes.length === 0 && <NoTypesNotice model={context.model.name} />}
      <ActivityBasicsCard
        control={form.control}
        form={data}
        value={value}
        stored={stored}
        onChooseType={chooseType}
      />
      <ActivityDatesCard
        form={form}
        data={data}
        value={value}
        stored={stored}
        planCurrency={planCurrency}
        costNote={costFromType ? typeDefaultNote(data, value.activityTypeId) : undefined}
        onCostTyped={() => setCostFromType(false)}
      />

      {/* The attribute groups' own names are the card's headings ("Shell attributes",
          "Lubricants attributes"), so no heading repeats its parent (UI audit AF-02, X-09). */}
      <FormCardFrame
        id={ACTIVITY_SECTION.attributes}
        label="Shell attributes"
        className={styles.attributes}
      >
        {attributes ? (
          <FormField
            control={form.control}
            name="attributes"
            render={(field) => (
              <ShellAttributeFields
                sectionHeading="h2"
                form={attributes}
                values={field.value}
                onChange={field.onChange}
                businessUnit={context.businessUnits.map((unit) => unit.id)}
                previousValues={stored?.attributes ?? {}}
                errors={attributeErrors(problems)}
                planLink={recordHref('plan', context.plan.id)}
              />
            )}
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
        <FormField
          control={form.control}
          name="goals"
          render={(field) => (
            <GoalsFields
              value={field.value}
              onChange={field.onChange}
              model={context.model}
              errors={{
                inq: errors.goals?.inq?.message,
                mql: errors.goals?.mql?.message,
                revenue: errors.goals?.revenue?.message,
              }}
            />
          )}
        />
      </FormCardFrame>

      <FormCard id={ACTIVITY_SECTION.more} title="More details">
        <FormField
          control={form.control}
          name="description"
          render={(field) => (
            <TextAreaField
              label="Description"
              maxLength={2000}
              {...field}
              name={fieldId('description')}
            />
          )}
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
        onConfirm={() => {
          if (question) {
            setPlannedCost(question.amount);
          }
          setQuestion(null);
        }}
      >
        {null}
      </ConfirmDialog>
    </div>
  );
}

function attributeErrors(problems: FieldProblem[]): Record<string, string> {
  const errors: Record<string, string> = {};
  for (const problem of problems) {
    if (problem.field.startsWith('attributes.') && !(problem.field in errors)) {
      errors[problem.field] = problem.message;
    }
  }
  return errors;
}

/** "From the activity type", or "Default 5,000.00 USD = 3,950.00 GBP at today's rate" (016 FR-016). */
function typeDefaultNote(form: ActivityFormData, typeId: string): string {
  const type = form.activityTypes.find((option) => option.id === typeId);
  const usd = type?.defaultPlannedCostUsd;
  const converted = type?.defaultPlannedCost;
  if (!usd || !converted || usd.currency === converted.currency) {
    return 'From the activity type';
  }
  return `Default ${formatMoneyText(usd)} = ${formatMoneyText(converted)} at today's rate`;
}
