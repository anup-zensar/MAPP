import { TriangleAlert } from 'lucide-react';
import type { UseFormReturn } from 'react-hook-form';
import { CheckboxField } from '@/ui/CheckboxField/CheckboxField';
import { DateRangeField } from '@/ui/DateRangeField/DateRangeField';
import { FormField } from '@/ui/FormField/FormField';
import { FormCard } from '@/ui/FormRail/FormRail';
import { MoneyField } from '@/ui/MoneyField/MoneyField';
import { SpreadEffectNote } from '@/features/line-items';
import { useFormatters } from '@/format/useFormatters';
import type { ActivityFormData } from './api';
import { fieldId } from './activity-form-state';
import type { FormState, StoredActivity } from './activity-form-state';
import styles from './ActivityForm.module.css';
import { ACTIVITY_SECTION } from './activity-form-progress';

const FROM_INITIATIVE = 'From the initiative';
const ONGOING_WARNING = "Ongoing can't be undone after saving.";
const ONGOING_LOCKED = "Ongoing can't be undone — to plan with an end date, create a new activity.";

export interface ActivityDatesCardProps {
  form: UseFormReturn<FormState>;
  data: ActivityFormData;
  value: FormState;
  stored?: StoredActivity;
  /** The plan currency the planned cost is typed in (016 FR-011). */
  planCurrency: string;
  /** Where a pre-filled planned cost came from, while it is still the activity type's default. */
  costNote?: string;
  /** The person typed a planned cost: it is no longer the type's default. */
  onCostTyped: () => void;
}

/** Dates & cost: the dates within the initiative's, what they do to the spreads, ongoing, cost. */
export function ActivityDatesCard(props: ActivityDatesCardProps) {
  const { form, data, value, stored, planCurrency } = props;
  const { date } = useFormatters();
  const { initiative } = data.context;
  const defaults = data.defaults;
  const datesFromInitiative = Boolean(
    defaults?.startDate &&
    value.startDate === defaults.startDate &&
    value.endDate === (defaults.endDate ?? '') &&
    !value.ongoing,
  );
  const allowed =
    initiative.startDate && initiative.endDate
      ? { start: initiative.startDate, end: initiative.endDate }
      : undefined;
  const ongoingSaved = Boolean(stored?.ongoing);

  return (
    <FormCard id={ACTIVITY_SECTION.dates} title="Dates & cost">
      <FormField
        control={form.control}
        name="startDate"
        render={(start) => (
          <FormField
            control={form.control}
            name="endDate"
            render={(end) => (
              <DateRangeField
                label="Start and end dates"
                idPrefix="field"
                marking="Needed to submit"
                hint={datesFromInitiative ? FROM_INITIATIVE : undefined}
                start={start.value}
                end={end.value}
                onChange={(range) => {
                  start.onChange(range.start);
                  end.onChange(range.end);
                }}
                allowed={allowed}
                allowedHint={
                  allowed
                    ? `Allowed: ${date(allowed.start)} – ${date(allowed.end)} (the initiative's dates)`
                    : 'The initiative has no dates yet.'
                }
                maxEnd={data.context.latestEndDate}
                errors={{ start: start.error, end: end.error }}
                endDisabledReason={value.ongoing ? 'Ongoing (no end date)' : undefined}
              />
            )}
          />
        )}
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
        <FormField
          control={form.control}
          name="ongoing"
          render={(field) => (
            <CheckboxField
              label="Ongoing (no end date)"
              name={fieldId('ongoing')}
              checked={field.value}
              onChange={(ongoing) => {
                if (ongoing) {
                  form.setValue('endDate', '', { shouldDirty: true });
                }
                field.onChange(ongoing);
              }}
              disabledReason={ongoingSaved ? ONGOING_LOCKED : undefined}
              error={field.error}
            />
          )}
        />
        {value.ongoing && !ongoingSaved && (
          <p role="status" className={styles.warning}>
            <TriangleAlert aria-hidden="true" />
            {ONGOING_WARNING}
          </p>
        )}
      </div>
      <FormField
        control={form.control}
        name="plannedCost"
        render={(field) => (
          <MoneyField
            label="Planned cost"
            marking="Needed to submit"
            hint={
              planCurrency === 'USD'
                ? '0.00 or more'
                : `0.00 or more, in ${planCurrency}, the plan's currency`
            }
            currency={planCurrency}
            {...field}
            name={fieldId('plannedCost')}
            onChange={(plannedCost) => {
              props.onCostTyped();
              field.onChange(plannedCost);
            }}
            note={props.costNote}
          />
        )}
      />
    </FormCard>
  );
}
