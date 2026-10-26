import { useId, useState } from 'react';
import { CheckboxField } from '@/ui/CheckboxField/CheckboxField';
import { ConfirmDialog } from '@/ui/ConfirmDialog/ConfirmDialog';
import { FormDialog } from '@/ui/FormDialog/FormDialog';
import { FormErrorSummary } from '@/ui/FormErrorSummary/FormErrorSummary';
import { TextField } from '@/ui/TextField/TextField';
import { useToast } from '@/ui/Toast/useToast';
import type { FieldProblem } from '@/ui/FormErrorSummary/FormErrorSummary';
import { useCreateView, useUpdateView } from './allWorkApi';
import type { SavedView, ViewState } from './allWorkTypes';
import { useViewRefusal } from './useViewRefusal';
import { nameProblem, viewNamed } from './viewNames';
import { savedStateOf } from './viewState';

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
  const [name, setName] = useState('');
  const [isDefault, setIsDefault] = useState(false);
  const [errors, setErrors] = useState<FieldProblem[]>([]);
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

  function submit() {
    const problem = nameProblem(name);
    if (problem) {
      setErrors([{ field: nameField, message: problem }]);
      return;
    }
    setErrors([]);
    const existing = viewNamed(views, name);
    if (existing) {
      setReplacing(existing);
      return;
    }
    create.mutate(
      { name: name.trim(), state: savedStateOf(state), isDefault },
      {
        onSuccess: (view) => done(view, 'saved'),
        onError: (error) => setErrors(refused(error, nameField)),
      },
    );
  }

  function replace(view: SavedView) {
    update.mutate(
      {
        viewId: view.id,
        body: { state: savedStateOf(state), ...(isDefault && { isDefault: true }) },
      },
      {
        onSuccess: (saved) => done(saved, 'updated'),
        onError: (error) => {
          setReplacing(null);
          setErrors(refused(error, nameField));
        },
      },
    );
  }

  const nameError = errors.find((error) => error.field === nameField)?.message;

  return (
    <>
      <FormDialog
        open
        onOpenChange={onOpenChange}
        title="Save current view"
        submitLabel="Save view"
        onSubmit={submit}
        busy={create.isPending}
      >
        <FormErrorSummary errors={errors} />
        <TextField
          label="Name"
          name={nameField}
          marking="Required"
          value={name}
          onChange={setName}
          error={nameError}
        />
        <CheckboxField
          label="Use as my default view"
          name={`${id}-default`}
          checked={isDefault}
          onChange={setIsDefault}
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
