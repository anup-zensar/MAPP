import type { Control } from 'react-hook-form';
import { ChoiceField } from '@/ui/ChoiceField/ChoiceField';
import { FormField } from '@/ui/FormField/FormField';
import { FormCard } from '@/ui/FormRail/FormRail';
import { LockedNote } from '@/ui/LockedNote/LockedNote';
import { TextField } from '@/ui/TextField/TextField';
import { PersonField } from '@/features/people';
import { formatMoneyText } from '@/format/decimal';
import type { ActivityFormData, ActivityTypeOption } from './api';
import { fieldId } from './activity-form-state';
import type { FormState, StoredActivity } from './activity-form-state';
import styles from './ActivityForm.module.css';
import { ACTIVITY_SECTION } from './activity-form-progress';
import { NoTypesText } from './NoTypesNotice';

const FROM_INITIATIVE = 'From the initiative';
const WORKFLOW_LOCKED = "It can't change while the activity is submitted or approved.";
/** While a round is open the person can recall the submission first (008 FR-013). */
const WORKFLOW_RECALL = 'Recall the activity to choose another workflow.';
const NO_WORKFLOW = 'No approval workflow is active — an administrator sets them up in Settings.';

export interface ActivityBasicsCardProps {
  control: Control<FormState>;
  form: ActivityFormData;
  value: FormState;
  stored?: StoredActivity;
  onChooseType: (typeId: string) => void;
}

/** Basics: names, the campaign and initiative it lives in, type, owner and approval workflow. */
export function ActivityBasicsCard({
  control,
  form,
  value,
  stored,
  onChooseType,
}: ActivityBasicsCardProps) {
  const { context, defaults } = form;
  const ownerFromInitiative = Boolean(defaults?.owner && value.ownerId === defaults.owner.id);
  const onlyWorkflow = Boolean(
    defaults?.approvalWorkflowId && value.approvalWorkflowId === defaults.approvalWorkflowId,
  );
  const noTypes = form.activityTypes.length === 0;

  return (
    <FormCard id={ACTIVITY_SECTION.basics} title="Basics">
      <FormField
        control={control}
        name="name"
        render={(field) => (
          <TextField label="Activity name" marking="Required" {...field} name={fieldId('name')} />
        )}
      />
      <FormField
        control={control}
        name="externalName"
        render={(field) => (
          <TextField
            label="External name"
            hint="Leave empty to use the activity name"
            {...field}
            name={fieldId('externalName')}
          />
        )}
      />
      <div className={styles.readOnly}>
        <dl className={styles.pairs}>
          <div className={styles.pair}>
            <dt>Campaign</dt>
            <dd>{context.campaign.name}</dd>
          </div>
          <div className={styles.pair}>
            <dt>Initiative</dt>
            <dd>{context.initiative.name}</dd>
          </div>
        </dl>
        <LockedNote>Change it later with Move</LockedNote>
      </div>
      <FormField
        control={control}
        name="activityTypeId"
        render={(field) => (
          <ChoiceField
            label="Activity type"
            marking="Required"
            options={form.activityTypes.map((type) => ({
              id: type.id,
              label: typeLabel(type),
              active: type.active,
            }))}
            {...field}
            name={fieldId('activityTypeId')}
            onChange={onChooseType}
            note={noTypes ? <NoTypesText model={context.model.name} /> : undefined}
          />
        )}
      />
      <FormField
        control={control}
        name="ownerId"
        render={(field) => (
          <PersonField
            label="Owner"
            marking="Needed to submit"
            hint={ownerFromInitiative ? FROM_INITIATIVE : undefined}
            {...field}
            name={fieldId('ownerId')}
          />
        )}
      />
      <FormField
        control={control}
        name="approvalWorkflowId"
        render={(field) => (
          <ChoiceField
            label="Approval workflow"
            marking="Needed to submit"
            hint={onlyWorkflow ? 'The only active approval workflow' : undefined}
            options={form.approvalWorkflows.map((workflow) => ({
              id: workflow.id,
              label: workflow.name,
              active: workflow.active,
            }))}
            {...field}
            name={fieldId('approvalWorkflowId')}
            disabledReason={workflowLock(stored)}
            note={form.approvalWorkflows.length === 0 ? NO_WORKFLOW : undefined}
          />
        )}
      />
    </FormCard>
  );
}

function workflowLock(stored: StoredActivity | undefined): string | undefined {
  if (stored?.status === 'submitted') {
    return WORKFLOW_RECALL;
  }
  return stored?.status === 'approved' ? WORKFLOW_LOCKED : undefined;
}

/** "Event · default 25,000.00 USD"; "Event — not allowed by the plan's model". */
function typeLabel(type: ActivityTypeOption): string {
  const withDefault = type.defaultPlannedCost
    ? `${type.name} · default ${formatMoneyText(type.defaultPlannedCost)}`
    : type.name;
  return type.allowed ? withDefault : `${withDefault} — not allowed by the plan's model`;
}
