import { useQueryClient } from '@tanstack/react-query';
import { Info } from 'lucide-react';
import { useState } from 'react';
import { useNavigate } from 'react-router';
import { conflictMessage } from '@/api/errors';
import {
  CheckboxField,
  DecimalField,
  FormPanel,
  FormSection,
  problemOf,
  TextAreaField,
  TextField,
  useToast,
} from '@/components';
import type { FieldProblem, SaveFailure } from '@/components';
import { asApiError, isSessionLost } from '@/features/approvals/messages';
import { parseWholeInput } from '@/lib/decimal';
import { useFormatters } from '@/lib/useFormatters';
import { toDrafts, toInputs } from './levelDrafts';
import type { LevelDraft } from './levelDrafts';
import { LevelsEditor } from './LevelsEditor';
import { openRoundsNote } from './workflowTexts';
import { fetchLatestWorkflow, useCreateWorkflow, useUpdateWorkflow } from './workflowsApi';
import type { ApprovalWorkflow, WorkflowUpdate } from './workflowsApi';
import styles from './WorkflowFormPanel.module.css';

const CONFLICT_NEXT_STEP =
  'Your changes are still here. Save again to replace their changes, or Cancel to see theirs.';
const LEVELS_HINT =
  'Levels are checked in order. A level can be needed only sometimes, approved automatically, or need every approver. Nobody approves an activity they own or submitted.';

/** New workflows start with a reminder after 3 days and escalation after 5 (017 FR-007). */
const NEW_REMINDER_DAYS = 3;
const NEW_ESCALATION_DAYS = 5;

/** The days of a reminder or an escalation as the form holds them: ticked, and the days as typed. */
interface TimingDraft {
  on: boolean;
  days: string;
}

function timingDraft(days: number | null | undefined, fallback: number): TimingDraft {
  return days ? { on: true, days: String(days) } : { on: false, days: String(fallback) };
}

/** The days for the API: null when unticked; undefined when the typed text is no whole number. */
function timingValue(timing: TimingDraft): number | null | undefined {
  if (!timing.on) {
    return null;
  }
  const days = parseWholeInput(timing.days);
  return days === null ? undefined : Number(days);
}

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
  const [start] = useState(() => ({
    name: workflow?.name ?? '',
    description: workflow?.description ?? '',
    levels: toDrafts(workflow),
    reminder: workflow
      ? timingDraft(workflow.reminderDays, NEW_REMINDER_DAYS)
      : { on: true, days: String(NEW_REMINDER_DAYS) },
    escalation: workflow
      ? timingDraft(workflow.escalationDays, NEW_ESCALATION_DAYS)
      : { on: true, days: String(NEW_ESCALATION_DAYS) },
  }));
  const [name, setName] = useState(start.name);
  const [description, setDescription] = useState(start.description);
  const [levels, setLevels] = useState<LevelDraft[]>(start.levels);
  const [reminder, setReminder] = useState<TimingDraft>(start.reminder);
  const [escalation, setEscalation] = useState<TimingDraft>(start.escalation);
  const [version, setVersion] = useState(workflow?.version ?? 0);
  const [saving, setSaving] = useState(false);
  const [problems, setProblems] = useState<FieldProblem[]>([]);
  const [failure, setFailure] = useState<SaveFailure | null>(null);

  const levelsChanged = JSON.stringify(toInputs(levels)) !== JSON.stringify(toInputs(start.levels));
  const reminderDays = timingValue(reminder);
  const escalationDays = timingValue(escalation);
  const isDirty =
    name !== start.name ||
    description !== start.description ||
    levelsChanged ||
    reminder.on !== start.reminder.on ||
    reminder.days !== start.reminder.days ||
    escalation.on !== start.escalation.on ||
    escalation.days !== start.escalation.days;

  /** Days that are no whole number can't be sent: said at once, in the API's words. */
  function timingProblems(): FieldProblem[] {
    const found: FieldProblem[] = [];
    if (reminderDays === undefined) {
      found.push({
        field: 'reminderDays',
        message: 'Remind after: enter a whole number of days from 1 to 60.',
      });
    }
    if (escalationDays === undefined) {
      found.push({
        field: 'escalationDays',
        message: 'Escalate after: enter a whole number of days from 1 to 90.',
      });
    }
    return found;
  }

  async function save() {
    const typed = timingProblems();
    if (typed.length > 0) {
      setProblems(typed);
      setFailure(null);
      return;
    }
    setSaving(true);
    try {
      if (workflow) {
        const changes: WorkflowUpdate = { version };
        if (name !== start.name) {
          changes.name = name;
        }
        if (description !== start.description) {
          changes.description = description;
        }
        if (levelsChanged) {
          changes.levels = toInputs(levels);
        }
        if (reminderDays !== timingValue(start.reminder)) {
          changes.reminderDays = reminderDays ?? null;
        }
        if (escalationDays !== timingValue(start.escalation)) {
          changes.escalationDays = escalationDays ?? null;
        }
        const saved = await updateWorkflow.mutateAsync({ id: workflow.id, changes });
        show({ message: `Approval workflow '${saved.name}' saved` });
      } else {
        const created = await createWorkflow.mutateAsync({
          name,
          description,
          levels: toInputs(levels),
          reminderDays: reminderDays ?? null,
          escalationDays: escalationDays ?? null,
        });
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
      setProblems(apiError.fields);
      setFailure(null);
      return;
    }
    setProblems([]);
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

  return (
    <FormPanel
      title={workflow ? 'Edit approval workflow' : 'New approval workflow'}
      isDirty={isDirty}
      saving={saving}
      problems={problems}
      failure={failure}
      onSave={() => void save()}
      onClose={onClose}
    >
      {workflow && workflow.waitingCount > 0 && (
        <p className={styles.note}>
          <Info aria-hidden="true" />
          <span>{openRoundsNote(workflow.waitingCount)}</span>
        </p>
      )}
      <TextField
        label="Name"
        name="name"
        marking="Required"
        maxLength={120}
        value={name}
        onChange={setName}
        error={problemOf(problems, 'name')}
      />
      <TextAreaField
        label="Description"
        name="description"
        maxLength={2000}
        rows={3}
        value={description}
        onChange={setDescription}
        error={problemOf(problems, 'description')}
      />
      <FormSection title="Levels" description={LEVELS_HINT}>
        {problemOf(problems, 'levels') && (
          <p id="levels" className={styles.error}>
            {problemOf(problems, 'levels')}
          </p>
        )}
        <LevelsEditor levels={levels} onChange={setLevels} problems={problems} />
      </FormSection>
      <FormSection title="Reminders and escalation" description="Days are calendar days.">
        <TimingFields
          checkLabel="Send a reminder"
          checkName="sendReminder"
          daysLabel="Remind after"
          daysName="reminderDays"
          value={reminder}
          onChange={setReminder}
          error={problemOf(problems, 'reminderDays')}
        />
        <TimingFields
          checkLabel="Escalate to the backup approver"
          checkName="escalate"
          daysLabel="Escalate after"
          daysName="escalationDays"
          value={escalation}
          onChange={setEscalation}
          error={problemOf(problems, 'escalationDays')}
        />
      </FormSection>
    </FormPanel>
  );
}

interface TimingFieldsProps {
  checkLabel: string;
  checkName: string;
  daysLabel: string;
  /** The API's key of the days ("reminderDays"), also the input's id. */
  daysName: string;
  value: TimingDraft;
  onChange: (value: TimingDraft) => void;
  error?: string;
}

/** A check box and, while ticked, its days ("Remind after 3 days waiting"). */
function TimingFields({
  checkLabel,
  checkName,
  daysLabel,
  daysName,
  value,
  onChange,
  error,
}: TimingFieldsProps) {
  return (
    <div className={styles.timing}>
      <CheckboxField
        label={checkLabel}
        name={checkName}
        checked={value.on}
        onChange={(on) => onChange({ ...value, on })}
      />
      {value.on && (
        <div className={styles.days}>
          <DecimalField
            label={daysLabel}
            name={daysName}
            decimals={0}
            suffix="days waiting"
            compact
            value={value.days}
            onChange={(days) => onChange({ ...value, days })}
            error={error}
          />
        </div>
      )}
    </div>
  );
}
