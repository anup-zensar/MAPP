import { zodResolver } from '@hookform/resolvers/zod';
import { useId } from 'react';
import { useForm } from 'react-hook-form';
import { FormDialog } from '@/ui/FormDialog/FormDialog';
import { FormErrorSummary } from '@/ui/FormErrorSummary/FormErrorSummary';
import { FormField } from '@/ui/FormField/FormField';
import { TextField } from '@/ui/TextField/TextField';
import { useToast } from '@/ui/Toast/useToast';
import { CHECK_ON_SAVE, useFormProblems } from '@/ui/useFormProblems';
import { useUpdateView } from './api';
import type { SavedView } from './all-work-types';
import { renameViewSchema } from './schemas';
import type { RenameViewValues } from './schemas';
import { useViewRefusal } from './useViewRefusal';

const FIELDS = ['name'] as const;

interface RenameViewDialogProps {
  view: SavedView;
  views: SavedView[];
  onClose: () => void;
}

/** Rename view: the same name rules as saving; another view's name is refused (FR-029). */
export function RenameViewDialog({ view, views, onClose }: RenameViewDialogProps) {
  const id = useId();
  const nameField = `${id}-name`;
  const form = useForm<RenameViewValues>({
    resolver: zodResolver(renameViewSchema(views, view.id)),
    defaultValues: { name: view.name },
    ...CHECK_ON_SAVE,
  });
  const { problems, onInvalid, showServerProblems } = useFormProblems(form, FIELDS);
  const update = useUpdateView();
  const refused = useViewRefusal();
  const { show } = useToast();

  const submit = form.handleSubmit(({ name }) => {
    update.mutate(
      { viewId: view.id, body: { name: name.trim() } },
      {
        onSuccess: (saved) => {
          show({ message: `View '${saved.name}' renamed` });
          onClose();
        },
        onError: (error) => {
          const apiProblems = refused(error, 'name');
          if (apiProblems.length > 0) {
            showServerProblems(apiProblems);
          } else {
            onClose();
          }
        },
      },
    );
  }, onInvalid);

  return (
    <FormDialog
      open
      onOpenChange={(isOpen) => !isOpen && onClose()}
      title="Rename view"
      submitLabel="Rename view"
      onSubmit={() => void submit()}
      busy={update.isPending}
    >
      <FormErrorSummary
        errors={problems}
        fieldId={(field) => (field === 'name' ? nameField : field)}
      />
      <FormField
        control={form.control}
        name="name"
        render={(field) => (
          <TextField label="Name" marking="Required" {...field} name={nameField} />
        )}
      />
    </FormDialog>
  );
}
