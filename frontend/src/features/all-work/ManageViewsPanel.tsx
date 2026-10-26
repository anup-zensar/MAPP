import { useId, useState } from 'react';
import { Button } from '@/ui/Button/Button';
import { ConfirmDialog } from '@/ui/ConfirmDialog/ConfirmDialog';
import { FormDialog } from '@/ui/FormDialog/FormDialog';
import { FormErrorSummary } from '@/ui/FormErrorSummary/FormErrorSummary';
import { SidePanel } from '@/ui/SidePanel/SidePanel';
import { TextField } from '@/ui/TextField/TextField';
import { useToast } from '@/ui/Toast/useToast';
import type { FieldProblem } from '@/ui/FormErrorSummary/FormErrorSummary';
import { useCreateView, useDeleteView, useUpdateView } from './api';
import type { SavedView } from './all-work-types';
import styles from './ManageViewsPanel.module.css';
import { useViewRefusal } from './useViewRefusal';
import { nameProblem, nameTaken, viewNamed } from './view-names';

export interface ManageViewsPanelProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** The person's saved views, by name. */
  views: SavedView[];
}

/**
 * Your saved views (US6, FR-031): set or remove the default, rename, and delete — after a
 * question, with Undo in the message, which saves the view again.
 */
export function ManageViewsPanel({ open, onOpenChange, views }: ManageViewsPanelProps) {
  const [renaming, setRenaming] = useState<SavedView | null>(null);
  const [deleting, setDeleting] = useState<SavedView | null>(null);
  const update = useUpdateView();
  const remove = useDeleteView();
  const create = useCreateView();
  const refused = useViewRefusal();
  const { show } = useToast();

  function setDefault(view: SavedView, isDefault: boolean) {
    update.mutate(
      { viewId: view.id, body: { isDefault } },
      {
        onSuccess: (saved) =>
          show({
            message: isDefault
              ? `'${saved.name}' is now your default view`
              : `'${saved.name}' is no longer your default view`,
          }),
        onError: (error) => refused(error, 'name'),
      },
    );
  }

  function confirmDelete(view: SavedView) {
    remove.mutate(view.id, {
      onSuccess: (deleted) => {
        setDeleting(null);
        show({
          message: `View '${deleted.name}' deleted`,
          action: {
            label: 'Undo',
            altText: 'Save the view again',
            onClick: () => restore(deleted),
          },
        });
      },
      onError: (error) => {
        setDeleting(null);
        refused(error, 'name');
      },
    });
  }

  /** Undo: the view is saved again with its name, state and default flag (a new ID). */
  function restore(view: SavedView) {
    create.mutate(
      { name: view.name, state: view.state, isDefault: view.isDefault },
      {
        onSuccess: (saved) => show({ message: `View '${saved.name}' restored` }),
        onError: (error) => refused(error, 'name'),
      },
    );
  }

  return (
    <>
      <SidePanel
        open={open}
        onOpenChange={onOpenChange}
        title="Your saved views"
        footer={<Button onClick={() => onOpenChange(false)}>Close</Button>}
      >
        <ul className={styles.list}>
          {views.map((view) => (
            <li key={view.id} className={styles.view}>
              <div className={styles.name}>
                <span className={styles.label}>{view.name}</span>
                {view.isDefault && <span className={styles.default}>Default</span>}
              </div>
              <div className={styles.actions}>
                <Button size="sm" onClick={() => setDefault(view, !view.isDefault)}>
                  {view.isDefault ? 'Remove default' : 'Set as default'}{' '}
                  <span className="visually-hidden">{view.name}</span>
                </Button>
                <Button size="sm" onClick={() => setRenaming(view)}>
                  Rename <span className="visually-hidden">{view.name}</span>
                </Button>
                <Button size="sm" onClick={() => setDeleting(view)}>
                  Delete <span className="visually-hidden">{view.name}</span>
                </Button>
              </div>
            </li>
          ))}
        </ul>
      </SidePanel>
      {renaming && (
        <RenameViewDialog view={renaming} views={views} onClose={() => setRenaming(null)} />
      )}
      {deleting && (
        <ConfirmDialog
          open
          onOpenChange={(isOpen) => !isOpen && setDeleting(null)}
          title={`Delete view '${deleting.name}'?`}
          confirmLabel="Delete view"
          tone="danger"
          onConfirm={() => confirmDelete(deleting)}
          busy={remove.isPending}
        >
          Only your saved view is deleted; no plans or records change.
        </ConfirmDialog>
      )}
    </>
  );
}

interface RenameViewDialogProps {
  view: SavedView;
  views: SavedView[];
  onClose: () => void;
}

/** Rename view: the same name rules as saving; another view's name is refused (FR-029). */
function RenameViewDialog({ view, views, onClose }: RenameViewDialogProps) {
  const id = useId();
  const nameField = `${id}-name`;
  const [name, setName] = useState(view.name);
  const [errors, setErrors] = useState<FieldProblem[]>([]);
  const update = useUpdateView();
  const refused = useViewRefusal();
  const { show } = useToast();

  function submit() {
    const problem = nameProblem(name) ?? (viewNamed(views, name, view.id) ? nameTaken(name) : null);
    if (problem) {
      setErrors([{ field: nameField, message: problem }]);
      return;
    }
    update.mutate(
      { viewId: view.id, body: { name: name.trim() } },
      {
        onSuccess: (saved) => {
          show({ message: `View '${saved.name}' renamed` });
          onClose();
        },
        onError: (error) => {
          const problems = refused(error, nameField);
          if (problems.length > 0) {
            setErrors(problems);
          } else {
            onClose();
          }
        },
      },
    );
  }

  return (
    <FormDialog
      open
      onOpenChange={(isOpen) => !isOpen && onClose()}
      title="Rename view"
      submitLabel="Rename view"
      onSubmit={submit}
      busy={update.isPending}
    >
      <FormErrorSummary errors={errors} />
      <TextField
        label="Name"
        name={nameField}
        marking="Required"
        value={name}
        onChange={setName}
        error={errors.find((error) => error.field === nameField)?.message}
      />
    </FormDialog>
  );
}
