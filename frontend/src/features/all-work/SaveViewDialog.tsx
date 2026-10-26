import { zodResolver } from '@hookform/resolvers/zod';
import { useId, useState } from 'react';
import { useForm } from 'react-hook-form';
import { CheckboxField } from '@/ui/CheckboxField/CheckboxField';
import { ConfirmDialog } from '@/ui/ConfirmDialog/ConfirmDialog';
import { FormDialog } from '@/ui/FormDialog/FormDialog';
import { FormErrorSummary } from '@/ui/FormErrorSummary/FormErrorSummary';
import { FormField } from '@/ui/FormField/FormField';
import { TextField } from '@/ui/TextField/TextField';
import { useToast } from '@/ui/Toast/useToast';
import { CHECK_ON_SAVE, useFormProblems } from '@/ui/useFormProblems';
import { useCreateView, useUpdateView } from './api';
import type { SavedView, ViewState } from './all-work-types';
import { saveViewSchema } from './schemas';
import type { SaveViewValues } from './schemas';
import { useViewRefusal } from './useViewRefusal';
import { viewNamed } from './view-names';
import { savedStateOf } from './view-state';

const FIELDS = ['name'] as const;

export interface SaveViewDialogProps {
  onOpenChange: (open: boolean) => void;
  /** The person's saved views: a name already used is replaced only after asking (FR-029). */
  views: SavedView[];
  /** What is saved: the state on screen. */
  state: ViewState;
  /** The saved (or replaced) view, which becomes the active one. */
  onSaved: (view: SavedView) => void;
}

/**
 * Save current view (US6, FR-029, FR-031): a name (required, at most 60 characters) and "Use as
 * my default view". A name the person already uses asks "Replace your view '<name>'?" first.
 * Mounted while open, so every opening starts empty.
 */
export function SaveViewDialog({ onOpenChange, views, state, onSaved }: SaveViewDialogProps) {
  const id = useId();
  const nameField = `${id}-name`;
  const form = useForm<SaveViewValues>({
    resolver: zodResolver(saveViewSchema),
    defaultValues: { name: '', isDefault: false },
    ...CHECK_ON_SAVE,
  });
  const { problems, onInvalid, showServerProblems, clearProblems } = useFormProblems(form, FIELDS);
  const [replacing, setReplacing] = useState<SavedView | null>(null);
  const create = useCreateView();
  const update = useUpdateView();
  const refused = useViewRefusal();
  const { show } = useToast();

  function done(view: SavedView, verb: 'saved' | 'updated') {
    show({ message: `View '${view.name}' ${verb}` });
    onSaved(view);
    onOpenChange(false);
  }

  const submit = form.handleSubmit(({ name, isDefault }) => {
    clearProblems();
    const existing = viewNamed(views, name);
    if (existing) {
      setReplacing(existing);
      return;
    }
    create.mutate(
      { name: name.trim(), state: savedStateOf(state), isDefault },
      {
        onSuccess: (view) => done(view, 'saved'),
        onError: (error) => showServerProblems(refused(error, 'name')),
      },
    );
  }, onInvalid);

  function replace(view: SavedView) {
    const isDefault = form.getValues('isDefault');
    update.mutate(
      {
        viewId: view.id,
        body: { state: savedStateOf(state), ...(isDefault && { isDefault: true }) },
      },
      {
        onSuccess: (saved) => done(saved, 'updated'),
        onError: (error) => {
          setReplacing(null);
          showServerProblems(refused(error, 'name'));
        },
      },
    );
  }

  return (
    <>
      <FormDialog
        open
        onOpenChange={onOpenChange}
        title="Save current view"
        submitLabel="Save view"
        onSubmit={() => void submit()}
        busy={create.isPending}
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
        <FormField
          control={form.control}
          name="isDefault"
          render={(field) => (
            <CheckboxField
              label="Use as my default view"
              name={`${id}-default`}
              checked={field.value}
              onChange={field.onChange}
            />
          )}
        />
      </FormDialog>
      {replacing && (
        <ConfirmDialog
          open
          onOpenChange={(open) => !open && setReplacing(null)}
          title={`Replace your view '${replacing.name}'?`}
          confirmLabel="Replace view"
          onConfirm={() => replace(replacing)}
          busy={update.isPending}
        >
          It will show what you see now.
        </ConfirmDialog>
      )}
    </>
  );
}
