import { CheckboxField } from '@/ui/CheckboxField/CheckboxField';
import { ChoiceField } from '@/ui/ChoiceField/ChoiceField';
import { FormField } from '@/ui/FormField/FormField';
import { TextAreaField } from '@/ui/TextAreaField/TextAreaField';
import { TextField } from '@/ui/TextField/TextField';
import type { components } from '@/api/generated/schema';
import { PersonField, TeamField } from '@/features/people';
import { useFieldHelp } from '@/features/settings';
import { formatAmount, formatPercent } from '@/format/decimal';
import type { Plan, PlanDefaults } from './api';
import { PLAN_TYPE_LABELS, PLAN_TYPES } from './plan-currency';
import type { SectionProps } from './plan-form-sections';
import { YEAR_LOCKED } from './plan-text';
import { PlanFormCard } from './PlanFormCard';
import { ReadOnlyField } from './ReadOnlyField';

type Model = components['schemas']['Model'];

export interface PlanBasicsSectionProps extends SectionProps {
  /** Create: the defaults, for the years to choose from and the "Default: …" hints (FR-006). */
  defaults?: PlanDefaults;
  models: Model[];
  chosenModel: Plan['model'] | undefined;
  /** The chosen owner's name, for the note when they are in none of the chosen teams. */
  ownerName: string;
  onYearChange: (year: string) => void;
}

/** Basics: name, year, plan type (016), model, teams, owner, AOP and description. */
export function PlanBasicsSection({
  control,
  values,
  mode,
  plan,
  with016,
  hint,
  defaults,
  models,
  chosenModel,
  ownerName,
  onYearChange,
}: PlanBasicsSectionProps) {
  const { help } = useFieldHelp();
  const modelOptions = models.map((model) => ({
    id: model.id,
    label: model.name,
    active: model.active,
  }));
  if (plan && !models.some((model) => model.id === plan.model.id)) {
    modelOptions.push({ id: plan.model.id, label: plan.model.name, active: plan.model.active });
  }

  return (
    <PlanFormCard id="plan-basics" title="Basics">
      <FormField
        control={control}
        name="name"
        render={(field) => (
          <TextField
            label="Plan name"
            marking="Required"
            tip={help('plan.name')}
            maxLength={120}
            {...field}
          />
        )}
      />
      {mode === 'create' ? (
        <FormField
          control={control}
          name="year"
          render={(field) => (
            <ChoiceField
              label="Year"
              marking="Required"
              tip={help('plan.year')}
              hint={hint('year', 'Default: the current year')}
              options={(defaults?.years ?? [Number(field.value)]).map((year) => ({
                id: String(year),
                label: String(year),
                active: true,
              }))}
              {...field}
              onChange={(year) => {
                onYearChange(year);
                field.onChange(year);
              }}
            />
          )}
        />
      ) : (
        <ReadOnlyField label="Year" marking="Required" tip={help('plan.year')} value={values.year}>
          {YEAR_LOCKED}
        </ReadOnlyField>
      )}
      {with016 && (
        <FormField
          control={control}
          name="planType"
          render={(field) => (
            <ChoiceField
              label="Plan type"
              marking="Required"
              tip={help('plan.planType')}
              hint={hint('planType', 'Default: Operational')}
              options={PLAN_TYPES.map((type) => ({
                id: type,
                label: PLAN_TYPE_LABELS[type],
                active: true,
              }))}
              {...field}
              onChange={(type) =>
                field.onChange(PLAN_TYPES.find((known) => known === type) ?? field.value)
              }
            />
          )}
        />
      )}
      <FormField
        control={control}
        name="modelId"
        render={(field) => (
          <ChoiceField
            label="Model"
            marking="Required"
            tip={help('plan.model')}
            hint={hint('modelId', 'Default: the only active model', !!defaults?.modelId)}
            options={modelOptions}
            note={
              chosenModel
                ? `INQ→MQL ${formatPercent(chosenModel.inqToMqlRate)} · MQL→deal ${formatPercent(chosenModel.mqlToDealRate)} · average deal size ${formatAmount(chosenModel.averageDealSize.amount)} ${chosenModel.averageDealSize.currency}`
                : undefined
            }
            {...field}
          />
        )}
      />
      <FormField
        control={control}
        name="teamIds"
        render={(field) => (
          <TeamField
            label="Teams"
            marking="Needed to publish"
            tip={help('plan.teams')}
            hint={hint('teamIds', 'Default: your teams', !!defaults?.teamIds.length)}
            {...field}
          />
        )}
      />
      <FormField
        control={control}
        name="ownerId"
        render={(field) => (
          <PersonField
            label="Owner"
            marking="Needed to publish"
            tip={help('plan.owner')}
            hint={hint('ownerId', 'Default: you', !!defaults?.ownerId)}
            teamIds={values.teamIds}
            notMemberText={`${ownerName} isn't in any of the chosen teams — choose another owner or clear the field`}
            {...field}
          />
        )}
      />
      <FormField
        control={control}
        name="aop"
        render={({ name, value, onChange }) => (
          <CheckboxField
            label="Strategic / AOP plan"
            name={name}
            tip={help('plan.aop')}
            hint={hint('aop', 'Default: No')}
            checked={value}
            onChange={onChange}
          />
        )}
      />
      <FormField
        control={control}
        name="description"
        render={(field) => (
          <TextAreaField
            label="Description"
            tip={help('plan.description')}
            maxLength={2000}
            {...field}
          />
        )}
      />
    </PlanFormCard>
  );
}
