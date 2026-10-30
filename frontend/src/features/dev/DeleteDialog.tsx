import { useState } from 'react';
import type { components } from '@/api/generated/schema';
import { ConfirmDialog } from '@/ui/ConfirmDialog/ConfirmDialog';
import { useUndoableDelete } from '@/ui/useUndoableDelete';
import { SAMPLE_PAGE_LISTS } from './api';
import { useFailureMessage } from './useFailureMessage';

type DeleteResult = components['schemas']['DeleteResult'];

/** "Sample folder 'Ideas' deleted" with Undo, restorable on this page too. */
const UNDO_OPTIONS = {
  invalidate: SAMPLE_PAGE_LISTS,
  altText: 'Restore it from Recently deleted on this page',
};

interface DeleteDialogProps {
  title: string;
  /** What goes to the recycle bin with it. */
  text: string;
  confirmLabel: string;
  /** Deletes the record and answers its recycle-bin entry. */
  remove: () => Promise<DeleteResult>;
  onClose: () => void;
}

export function DeleteDialog({ title, text, confirmLabel, remove, onClose }: DeleteDialogProps) {
  const [busy, setBusy] = useState(false);
  const afterDelete = useUndoableDelete(UNDO_OPTIONS);
  const showFailure = useFailureMessage();

  async function removeAndTell() {
    try {
      afterDelete(await remove());
    } catch (error) {
      showFailure(error, removeAndTell);
    }
  }

  async function confirm() {
    setBusy(true);
    await removeAndTell();
    onClose();
  }

  return (
    <ConfirmDialog
      open
      onOpenChange={(open) => {
        if (!open) {
          onClose();
        }
      }}
      title={title}
      confirmLabel={confirmLabel}
      tone="danger"
      busy={busy}
      onConfirm={confirm}
    >
      {text}
    </ConfirmDialog>
  );
}
