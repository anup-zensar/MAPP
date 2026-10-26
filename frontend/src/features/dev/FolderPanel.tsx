import { zodResolver } from '@hookform/resolvers/zod';
import { useQueryClient } from '@tanstack/react-query';
import { useRef, useState } from 'react';
import { useForm } from 'react-hook-form';
import { ApiError, conflictMessage } from '@/api/errors';
import { FormField } from '@/ui/FormField/FormField';
import { FormPanel } from '@/ui/FormPanel/FormPanel';
import { TextField } from '@/ui/TextField/TextField';
import { useToast } from '@/ui/Toast/useToast';
import { CHECK_ON_SAVE, useFormProblems } from '@/ui/useFormProblems';
import type { SaveFailure } from '@/ui/FormPanel/problem-of';
import { useFormatters } from '@/format/useFormatters';
import { fetchLatestFolder, useCreateFolder, useRenameFolder } from './api';
import type { SampleFolder } from './api';
import { folderSchema } from './schemas';
import type { FolderValues } from './schemas';

const CONFLICT_NEXT_STEP =
  'Your changes are still here. Save again to replace their changes, or Cancel to see theirs.';

const FIELDS = ['name'] as const;

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
  const form = useForm<FolderValues>({
    resolver: zodResolver(folderSchema),
    defaultValues: { name: folder?.name ?? '' },
    ...CHECK_ON_SAVE,
  });
  const version = useRef(folder?.version ?? 0);
  const { problems, onInvalid, showServerProblems, clearProblems } = useFormProblems(form, FIELDS);
  const [failure, setFailure] = useState<SaveFailure | null>(null);

  async function saveFolder({ name }: FolderValues) {
    try {
      if (folder) {
        const saved = await renameFolder.mutateAsync({
          id: folder.id,
          name,
          version: version.current,
        });
        show({ message: `Sample folder '${saved.name}' saved` });
      } else {
        const created = await createFolder.mutateAsync(name);
        show({ message: `Sample folder '${created.name}' created` });
      }
      onClose();
    } catch (error) {
      await showFailure(error);
    }
  }

  async function showFailure(error: unknown) {
    const apiError = error instanceof ApiError ? error : new ApiError('other');
    if (apiError.kind === 'validation') {
      showServerProblems(apiError.fields);
      setFailure(null);
      return;
    }
    clearProblems();
    if (apiError.kind === 'conflict' && folder) {
      // The typed name stays; the form takes the version that includes the other person's change,
      // so Save again replaces it on purpose (research R5). When the folders cannot be read now,
      // the form keeps its version and the next Save meets the conflict again.
      const latest = await fetchLatestFolder(queryClient, folder.id).catch(() => undefined);
      if (latest) {
        version.current = latest.version;
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
      isDirty={form.formState.isDirty}
      saving={form.formState.isSubmitting}
      problems={problems}
      failure={failure}
      onSave={() => void form.handleSubmit(saveFolder, onInvalid)()}
      onClose={onClose}
    >
      <FormField
        control={form.control}
        name="name"
        render={(field) => <TextField label="Name" marking="Required" {...field} />}
      />
    </FormPanel>
  );
}
