import { useQueryClient } from '@tanstack/react-query';
import { Info } from 'lucide-react';
import { useState } from 'react';
import { useNavigate } from 'react-router';
import { conflictMessage } from '@/api/errors';
import {
  FormPanel,
  FormSection,
  problemOf,
  TextAreaField,
  TextField,
  useToast,
} from '@/components';
import type { FieldProblem, SaveFailure } from '@/components';
import { asApiError, isSessionLost } from '@/features/approvals/messages';
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
  'Every level applies to every submission, in order. Any one approver of a level approves it.';

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
  }));
  const [name, setName] = useState(start.name);
  const [description, setDescription] = useState(start.description);
  const [levels, setLevels] = useState<LevelDraft[]>(start.levels);
  const [version, setVersion] = useState(workflow?.version ?? 0);
  const [saving, setSaving] = useState(false);
  const [problems, setProblems] = useState<FieldProblem[]>([]);
  const [failure, setFailure] = useState<SaveFailure | null>(null);

  const levelsChanged = JSON.stringify(toInputs(levels)) !== JSON.stringify(toInputs(start.levels));
  const isDirty = name !== start.name || description !== start.description || levelsChanged;

  async function save() {
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
        const saved = await updateWorkflow.mutateAsync({ id: workflow.id, changes });
        show({ message: `Approval workflow '${saved.name}' saved` });
      } else {
        const created = await createWorkflow.mutateAsync({
          name,
          description,
          levels: toInputs(levels),
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
    </FormPanel>
  );
}
