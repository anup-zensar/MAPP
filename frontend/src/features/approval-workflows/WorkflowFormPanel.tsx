import { zodResolver } from '@hookform/resolvers/zod';
import { useQueryClient } from '@tanstack/react-query';
import { Info } from 'lucide-react';
import { useState } from 'react';
import { useForm, useWatch } from 'react-hook-form';
import type { Control } from 'react-hook-form';
import { useNavigate } from 'react-router';
import { conflictMessage } from '@/api/errors';
import { CheckboxField } from '@/ui/CheckboxField/CheckboxField';
import { DecimalField } from '@/ui/DecimalField/DecimalField';
import { FormField } from '@/ui/FormField/FormField';
import { FormPanel } from '@/ui/FormPanel/FormPanel';
import { problemOf } from '@/ui/FormPanel/problem-of';
import { FormSection } from '@/ui/FormSection/FormSection';
import { TextAreaField } from '@/ui/TextAreaField/TextAreaField';
import { TextField } from '@/ui/TextField/TextField';
import { useToast } from '@/ui/Toast/useToast';
import { CHECK_ON_SAVE, useFormProblems } from '@/ui/useFormProblems';
import type { SaveFailure } from '@/ui/FormPanel/problem-of';
import { asApiError, isSessionLost } from '@/api/errors';
import { useFormatters } from '@/format/useFormatters';
import { LevelsEditor } from './LevelsEditor';
import {
  isWorkflowChanged,
  newWorkflow,
  workflowChanges,
  workflowSchema,
  workflowValuesOf,
} from './schemas';
import type { WorkflowValues } from './schemas';
import { openRoundsNote } from './workflow-texts';
import { fetchLatestWorkflow, useCreateWorkflow, useUpdateWorkflow } from './api';
import type { ApprovalWorkflow } from './api';
import styles from './WorkflowFormPanel.module.css';

const CONFLICT_NEXT_STEP =
  'Your changes are still here. Save again to replace their changes, or Cancel to see theirs.';
const LEVELS_HINT =
  'Levels are checked in order. A level can be needed only sometimes, approved automatically, or need every approver. Nobody approves an activity they own or submitted.';
const FIELDS = ['name', 'description', 'reminderDays', 'escalationDays'] as const;

export interface WorkflowFormPanelProps {
  /** The workflow to edit; without one, the panel creates a new workflow. */
  workflow?: ApprovalWorkflow;
  onClose: () => void;
}

/**
 * "New approval workflow" / "Edit approval workflow" (US4, FR-002, FR-003, FR-006): name,
 * description and the ordered levels. The web app checks nothing it cannot show at once: Save sends
 * the form and every problem the API finds is shown together, in the summary and at its field.
 * Editing a workflow with open rounds says that those rounds keep their levels.
 */
export function WorkflowFormPanel({ workflow, onClose }: WorkflowFormPanelProps) {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const { show } = useToast();
  const { dateFormat } = useFormatters();
  const createWorkflow = useCreateWorkflow();
  const updateWorkflow = useUpdateWorkflow();
  // The edit-form rule: the workflow, its version included, is copied into the form once.
  const [start] = useState(() => workflowValuesOf(workflow));
  const form = useForm<WorkflowValues>({
    resolver: zodResolver(workflowSchema),
    defaultValues: start,
    ...CHECK_ON_SAVE,
  });
  const { problems, onInvalid, showServerProblems, clearProblems } = useFormProblems(form, FIELDS);
  const isDirty = useWatch({
    control: form.control,
    compute: (values) => isWorkflowChanged(values, start),
  });
  const [version, setVersion] = useState(workflow?.version ?? 0);
  const [saving, setSaving] = useState(false);
  const [failure, setFailure] = useState<SaveFailure | null>(null);

  async function save(values: WorkflowValues) {
    setSaving(true);
    try {
      if (workflow) {
        const changes = workflowChanges(values, start, version);
        const saved = await updateWorkflow.mutateAsync({ id: workflow.id, changes });
        show({ message: `Approval workflow '${saved.name}' saved` });
      } else {
        const created = await createWorkflow.mutateAsync(newWorkflow(values));
        show({
          message: `Approval workflow '${created.name}' created`,
          action: {
            label: 'View',
            altText: 'Find it in the list of approval workflows',
            onClick: () => void navigate(`/settings/approval-workflows/${created.id}`),
          },
        });
      }
      onClose();
    } catch (error) {
      await showFailure(error);
    } finally {
      setSaving(false);
    }
  }

  async function showFailure(error: unknown) {
    const apiError = asApiError(error);
    if (apiError.kind === 'validation') {
      showServerProblems(apiError.fields);
      setFailure(null);
      return;
    }
    clearProblems();
    if (isSessionLost(apiError)) {
      setFailure(null);
      return;
    }
    if (apiError.kind === 'conflict' && workflow) {
      // The typed changes stay; the form takes the latest version, so Save again replaces the
      // other person's change on purpose.
      const latest = await fetchLatestWorkflow(queryClient, workflow.id).catch(() => undefined);
      if (latest) {
        setVersion(latest.version);
      }
      setFailure({
        message: conflictMessage(apiError, 'approval workflow', dateFormat),
        nextStep: CONFLICT_NEXT_STEP,
      });
      return;
    }
    setFailure({ message: apiError.message });
  }

  function showTypedProblems() {
    onInvalid();
    setFailure(null);
  }

  return (
    <FormPanel
      title={workflow ? 'Edit approval workflow' : 'New approval workflow'}
      isDirty={isDirty}
      saving={saving}
      problems={problems}
      failure={failure}
      onSave={() => void form.handleSubmit(save, showTypedProblems)()}
      onClose={onClose}
    >
      {workflow && workflow.waitingCount > 0 && (
        <p className={styles.note}>
          <Info aria-hidden="true" />
          <span>{openRoundsNote(workflow.waitingCount)}</span>
        </p>
      )}
      <FormField
        control={form.control}
        name="name"
        render={(field) => <TextField label="Name" marking="Required" maxLength={120} {...field} />}
      />
      <FormField
        control={form.control}
        name="description"
        render={(field) => (
          <TextAreaField label="Description" maxLength={2000} rows={3} {...field} />
        )}
      />
      <FormSection title="Levels" description={LEVELS_HINT}>
        {problemOf(problems, 'levels') && (
          <p id="levels" className={styles.error}>
            {problemOf(problems, 'levels')}
          </p>
        )}
        <FormField
          control={form.control}
          name="levels"
          render={(field) => (
            <LevelsEditor levels={field.value} onChange={field.onChange} problems={problems} />
          )}
        />
      </FormSection>
      <FormSection title="Reminders and escalation" description="Days are calendar days.">
        <TimingFields
          control={form.control}
          checkLabel="Send a reminder"
          checkName="sendReminder"
          daysLabel="Remind after"
          daysName="reminderDays"
        />
        <TimingFields
          control={form.control}
          checkLabel="Escalate to the backup approver"
          checkName="escalate"
          daysLabel="Escalate after"
          daysName="escalationDays"
        />
      </FormSection>
    </FormPanel>
  );
}

interface TimingFieldsProps {
  control: Control<WorkflowValues>;
  checkLabel: string;
  checkName: 'sendReminder' | 'escalate';
  daysLabel: string;
  /** The API's key of the days, also the input's id. */
  daysName: 'reminderDays' | 'escalationDays';
}

/** A check box and, while ticked, its days ("Remind after 3 days waiting"). */
function TimingFields({ control, checkLabel, checkName, daysLabel, daysName }: TimingFieldsProps) {
  const on = useWatch({ control, name: checkName });
  return (
    <div className={styles.timing}>
      <FormField
        control={control}
        name={checkName}
        render={({ value, ...field }) => (
          <CheckboxField label={checkLabel} checked={value} {...field} />
        )}
      />
      {on && (
        <div className={styles.days}>
          <FormField
            control={control}
            name={daysName}
            render={(field) => (
              <DecimalField
                label={daysLabel}
                decimals={0}
                suffix="days waiting"
                compact
                {...field}
              />
            )}
          />
        </div>
      )}
    </div>
  );
}
