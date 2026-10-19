import { useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { ApiError, conflictMessage } from '@/api/errors';
import { FormPanel, problemOf, TextField, useToast } from '@/components';
import type { FieldProblem, SaveFailure } from '@/components';
import { useFormatters } from '@/lib/useFormatters';
import { fetchLatestFolder, useAddNote, useCreateFolder, useRenameFolder } from './sampleApi';
import type { SampleFolder } from './sampleApi';

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
  const { dateFormat } = useFormatters();
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
        message: conflictMessage(apiError, 'sample folder', dateFormat),
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
