import { zodResolver } from '@hookform/resolvers/zod';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { ApiError } from '@/api/errors';
import { FormField } from '@/ui/FormField/FormField';
import { FormPanel } from '@/ui/FormPanel/FormPanel';
import { TextField } from '@/ui/TextField/TextField';
import { useToast } from '@/ui/Toast/useToast';
import { CHECK_ON_SAVE, useFormProblems } from '@/ui/useFormProblems';
import type { SaveFailure } from '@/ui/FormPanel/problem-of';
import { useAddNote } from './api';
import type { SampleFolder } from './api';
import { noteSchema } from './schemas';
import type { NoteValues } from './schemas';

const FIELDS = ['title'] as const;

export interface NotePanelProps {
  /** The folder the note goes in. */
  folder: SampleFolder;
  onClose: () => void;
}

/** "Add note" (plan "Developer pages"): the title of a new note in the folder. */
export function NotePanel({ folder, onClose }: NotePanelProps) {
  const { show } = useToast();
  const addNote = useAddNote();
  const form = useForm<NoteValues>({
    resolver: zodResolver(noteSchema),
    defaultValues: { title: '' },
    ...CHECK_ON_SAVE,
  });
  const { problems, onInvalid, showServerProblems } = useFormProblems(form, FIELDS);
  const [failure, setFailure] = useState<SaveFailure | null>(null);

  const save = form.handleSubmit(async ({ title }) => {
    try {
      const note = await addNote.mutateAsync({ folderId: folder.id, title });
      show({ message: `Sample note '${note.title}' added` });
      onClose();
    } catch (error) {
      const apiError = error instanceof ApiError ? error : new ApiError('other');
      const isFieldProblem = apiError.kind === 'validation';
      showServerProblems(isFieldProblem ? apiError.fields : []);
      setFailure(isFieldProblem ? null : { message: apiError.message });
    }
  }, onInvalid);

  return (
    <FormPanel
      title="Add note"
      isDirty={form.formState.isDirty}
      saving={form.formState.isSubmitting}
      problems={problems}
      failure={failure}
      onSave={() => void save()}
      onClose={onClose}
    >
      <FormField
        control={form.control}
        name="title"
        render={(field) => <TextField label="Title" marking="Required" {...field} />}
      />
    </FormPanel>
  );
}
