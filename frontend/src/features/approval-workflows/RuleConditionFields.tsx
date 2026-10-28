import { ChoiceField } from '@/ui/ChoiceField/ChoiceField';
import { MoneyField } from '@/ui/MoneyField/MoneyField';
import type { ChoiceOption } from '@/ui/ChoiceField/ChoiceField';
import type { ConditionDraft } from './level-drafts';
import type { ApprovalRuleOptions } from './api';
import type { ObjectType } from './schemas';
import styles from './RuleConditionFields.module.css';

export interface RuleConditionFieldsProps {
  value: ConditionDraft;
  onChange: (value: ConditionDraft) => void;
  options: ApprovalRuleOptions | undefined;
  optionsReason?: string;
  idPrefix: string;
  label: string;
  objectType: ObjectType;
}

export function RuleConditionFields({
  value,
  onChange,
  options,
  optionsReason,
  idPrefix,
  label,
  objectType,
}: RuleConditionFieldsProps) {
  const activity = objectType === 'activity';
  const amount = activity ? 'Planned cost' : 'Amount';
  const businessUnits: ChoiceOption[] = (options?.businessUnits ?? []).map((unit) => ({
    id: unit.id,
    label: unit.label,
    active: unit.active,
  }));
  const activityTypes: ChoiceOption[] = (options?.activityTypes ?? []).map((type) => ({
    id: type.id,
    label: type.name,
    active: type.active,
  }));

  return (
    <div role="group" aria-label={label} className={styles.condition}>
      <div className={styles.amounts}>
        <MoneyField
          label={`${amount} at least (USD)`}
          name={`${idPrefix}.plannedCostAtLeast`}
          currency="USD"
          value={value.plannedCostAtLeast}
          onChange={(plannedCostAtLeast) => onChange({ ...value, plannedCostAtLeast })}
        />
        <MoneyField
          label={`${amount} below (USD)`}
          name={`${idPrefix}.plannedCostBelow`}
          currency="USD"
          value={value.plannedCostBelow}
          onChange={(plannedCostBelow) => onChange({ ...value, plannedCostBelow })}
        />
      </div>
      {activity && (
        <ActivityConditions
          value={value}
          onChange={onChange}
          businessUnits={businessUnits}
          activityTypes={activityTypes}
          idPrefix={idPrefix}
          optionsReason={optionsReason}
        />
      )}
    </div>
  );
}

interface ActivityConditionsProps {
  value: ConditionDraft;
  onChange: (value: ConditionDraft) => void;
  businessUnits: ChoiceOption[];
  activityTypes: ChoiceOption[];
  idPrefix: string;
  optionsReason?: string;
}

/** Business units and activity types are fields of activities only. */
function ActivityConditions({
  value,
  onChange,
  businessUnits,
  activityTypes,
  idPrefix,
  optionsReason,
}: ActivityConditionsProps) {
  return (
    <>
      <ChoiceField
        multiple
        label="Business unit is one of"
        name={`${idPrefix}.businessUnitIds`}
        options={businessUnits}
        value={value.businessUnitIds}
        onChange={(businessUnitIds) => onChange({ ...value, businessUnitIds })}
        disabledReason={optionsReason || undefined}
      />
      <ChoiceField
        multiple
        label="Activity type is one of"
        name={`${idPrefix}.activityTypeIds`}
        options={activityTypes}
        value={value.activityTypeIds}
        onChange={(activityTypeIds) => onChange({ ...value, activityTypeIds })}
        disabledReason={optionsReason || undefined}
      />
    </>
  );
}
