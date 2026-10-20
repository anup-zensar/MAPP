import { ChoiceField, MoneyField } from '@/components';
import type { ChoiceOption } from '@/components';
import type { ConditionDraft } from './levelDrafts';
import type { ApprovalRuleOptions } from './workflowsApi';
import styles from './RuleConditionFields.module.css';

export interface RuleConditionFieldsProps {
  value: ConditionDraft;
  onChange: (value: ConditionDraft) => void;
  /** The business units and activity types to choose from; undefined while they load. */
  options: ApprovalRuleOptions | undefined;
  /** Why the lists can't be used yet ("Loading…", "couldn't be loaded"); empty when they can. */
  optionsReason?: string;
  /** Makes the fields' ids: "<idPrefix>.plannedCostAtLeast", … */
  idPrefix: string;
  /** Names the group for screen readers: "Level 1: only needed when". */
  label: string;
}

/**
 * The four parts of a level condition (017 FR-005, data-model §2): planned cost at least and below
 * (in US dollars, whatever the plan's currency — the activity's planned cost is converted at
 * submission), business unit is one of and activity type is one of. Every part that is set must
 * match. Inactive business units and types stay chosen and read "(inactive)". The API's one
 * problem for the condition is shown at the check box that reveals these fields.
 */
export function RuleConditionFields({
  value,
  onChange,
  options,
  optionsReason,
  idPrefix,
  label,
}: RuleConditionFieldsProps) {
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
          label="Planned cost at least (USD)"
          name={`${idPrefix}.plannedCostAtLeast`}
          currency="USD"
          value={value.plannedCostAtLeast}
          onChange={(plannedCostAtLeast) => onChange({ ...value, plannedCostAtLeast })}
        />
        <MoneyField
          label="Planned cost below (USD)"
          name={`${idPrefix}.plannedCostBelow`}
          currency="USD"
          value={value.plannedCostBelow}
          onChange={(plannedCostBelow) => onChange({ ...value, plannedCostBelow })}
        />
      </div>
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
    </div>
  );
}
