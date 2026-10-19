import { useQueryClient } from '@tanstack/react-query';
import { CircleAlert } from 'lucide-react';
import { useId, useState } from 'react';
import type { FormEvent, ReactNode } from 'react';
import { ApiError, conflictMessage } from '@/api/errors';
import { Button, FormErrorSummary, SidePanel, TextField, useToast } from '@/components';
import type { FieldProblem } from '@/components';
import { useUnsavedChanges } from '@/lib/useUnsavedChanges';
import { fetchLatestFolder, useAddNote, useCreateFolder, useRenameFolder } from './sampleApi';
import type { SampleFolder } from './sampleApi';
import styles from './SamplePanels.module.css';

/** A failed save that is not about one field, shown above the form (FR-020). */
interface SaveFailure {
  message: string;
  /** What the person can do now, after a conflict. */
  nextStep?: string;
}

const CONFLICT_NEXT_STEP =
  'Your changes are still here. Save again to replace their changes, or Cancel to see theirs.';

export interface FolderPanelProps {
  /** The folder to rename; without one, the panel creates a new folder. */
  folder?: SampleFolder;
  onClose: () => void;
}

/**
 * "New sample folder" or "Edit sample folder" (plan "Developer pages"): the name, every problem at
 * once, the conflict message when someone else saved first (FR-029), and unsaved changes protected.
 */
export function FolderPanel({ folder, onClose }: FolderPanelProps) {
  const queryClient = useQueryClient();
  const { show } = useToast();
  const createFolder = useCreateFolder();
  const renameFolder = useRenameFolder();
  // The edit-form rule (plan "Shared hooks"): the folder, its version included, is copied into the
  // form once, when the panel opens. Later refreshes of the list never change the form.
  const startName = folder?.name ?? '';
  const [name, setName] = useState(startName);
  const [version, setVersion] = useState(folder?.version ?? 0);
  const [saving, setSaving] = useState(false);
  const [problems, setProblems] = useState<FieldProblem[]>([]);
  const [failure, setFailure] = useState<SaveFailure | null>(null);

  async function save() {
    setSaving(true);
    try {
      if (folder) {
        const saved = await renameFolder.mutateAsync({ id: folder.id, name, version });
        show({ message: `Sample folder '${saved.name}' saved` });
      } else {
        const created = await createFolder.mutateAsync(name);
        show({ message: `Sample folder '${created.name}' created` });
      }
      onClose();
    } catch (error) {
      await showFailure(error);
    } finally {
      setSaving(false);
    }
  }

  async function showFailure(error: unknown) {
    const apiError = error instanceof ApiError ? error : new ApiError('other');
    if (apiError.kind === 'validation') {
      setProblems(apiError.fields);
      setFailure(null);
      return;
    }
    setProblems([]);
    if (apiError.kind === 'conflict' && folder) {
      // The typed name stays; the form takes the version that includes the other person's change,
      // so Save again replaces it on purpose (research R5). When the folders cannot be read now,
      // the form keeps its version and the next Save meets the conflict again.
      const latest = await fetchLatestFolder(queryClient, folder.id).catch(() => undefined);
      if (latest) {
        setVersion(latest.version);
      }
      setFailure({
        message: conflictMessage(apiError, 'sample folder'),
        nextStep: CONFLICT_NEXT_STEP,
      });
      return;
    }
    setFailure({ message: apiError.message });
  }

  return (
    <FormPanel
      title={folder ? 'Edit sample folder' : 'New sample folder'}
      isDirty={name !== startName}
      saving={saving}
      problems={problems}
      failure={failure}
      onSave={save}
      onClose={onClose}
    >
      <TextField
        label="Name"
        name="name"
        marking="Required"
        value={name}
        onChange={setName}
        error={problemOf(problems, 'name')}
      />
    </FormPanel>
  );
}

export interface NotePanelProps {
  /** The folder the note goes in. */
  folder: SampleFolder;
  onClose: () => void;
}

/** "Add note" (plan "Developer pages"): the title of a new note in the folder. */
export function NotePanel({ folder, onClose }: NotePanelProps) {
  const { show } = useToast();
  const addNote = useAddNote();
  const [title, setTitle] = useState('');
  const [saving, setSaving] = useState(false);
  const [problems, setProblems] = useState<FieldProblem[]>([]);
  const [failure, setFailure] = useState<SaveFailure | null>(null);

  async function save() {
    setSaving(true);
    try {
      const note = await addNote.mutateAsync({ folderId: folder.id, title });
      show({ message: `Sample note '${note.title}' added` });
      onClose();
    } catch (error) {
      const apiError = error instanceof ApiError ? error : new ApiError('other');
      const isFieldProblem = apiError.kind === 'validation';
      setProblems(isFieldProblem ? apiError.fields : []);
      setFailure(isFieldProblem ? null : { message: apiError.message });
    } finally {
      setSaving(false);
    }
  }

  return (
    <FormPanel
      title="Add note"
      isDirty={title !== ''}
      saving={saving}
      problems={problems}
      failure={failure}
      onSave={save}
      onClose={onClose}
    >
      <TextField
        label="Title"
        name="title"
        marking="Required"
        value={title}
        onChange={setTitle}
        error={problemOf(problems, 'title')}
      />
    </FormPanel>
  );
}

interface FormPanelProps {
  title: string;
  /** Something was typed that is not saved yet. */
  isDirty: boolean;
  /** A save is on its way; another Save is ignored meanwhile. */
  saving: boolean;
  /** The field problems of the last save, all at once. */
  problems: FieldProblem[];
  failure: SaveFailure | null;
  onSave: () => void;
  onClose: () => void;
  /** The form's fields. */
  children: ReactNode;
}

/**
 * The frame both panels share (design system §6.2): a side panel with the form, Save and Cancel
 * at the bottom, a failure and the error summary above the fields, and the "Discard changes?"
 * question before typed changes are lost — on Cancel, Close, Escape, a click beside the panel,
 * or when leaving the page.
 */
function FormPanel({
  title,
  isDirty,
  saving,
  problems,
  failure,
  onSave,
  onClose,
  children,
}: FormPanelProps) {
  const { dialog, confirmDiscard } = useUnsavedChanges(isDirty);
  // Save sits in the panel's footer, outside the form; the form attribute connects them.
  const formId = useId();

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    // A second Save while saving is ignored; nothing is greyed out.
    if (!saving) {
      onSave();
    }
  }

  return (
    <>
      <SidePanel
        open
        onOpenChange={(open) => {
          if (!open) {
            confirmDiscard(onClose);
          }
        }}
        title={title}
        footer={
          <>
            <Button type="submit" form={formId} variant="primary">
              Save
            </Button>
            <Button onClick={() => confirmDiscard(onClose)}>Cancel</Button>
          </>
        }
      >
        <form id={formId} noValidate aria-busy={saving} className={styles.form} onSubmit={submit}>
          {failure && <FailureMessage failure={failure} />}
          <FormErrorSummary errors={problems} />
          {children}
        </form>
      </SidePanel>
      {dialog}
    </>
  );
}

/** A failure that is not about one field, such as a conflict; screen readers hear it at once. */
function FailureMessage({ failure }: { failure: SaveFailure }) {
  return (
    <div role="alert" className={styles.failure}>
      <CircleAlert />
      <div>
        <p className={styles.failureMessage}>{failure.message}</p>
        {failure.nextStep && <p>{failure.nextStep}</p>}
      </div>
    </div>
  );
}

/** The problem with one field, shown under it. */
function problemOf(problems: FieldProblem[], field: string): string | undefined {
  return problems.find((problem) => problem.field === field)?.message;
}
