import { useState } from 'react';
import { ApiError } from '@/api/errors';
import { ConfirmDialog, FormDialog } from '@/components';
import { useUndoableDelete } from '@/lib/useUndoableDelete';
import { CHANGED_BY_ANY_CHANGE } from './shellAttributesApi';
import type { DeleteResult, Usage } from './shellAttributesApi';
import { useFailureToast } from './useStatusChange';
import type { RecordType } from './useStatusChange';

/** What setting it inactive does, said before offering it (plan "Inactive effects"). */
const INACTIVE_EFFECT: Record<RecordType, string> = {
  'Shell attribute':
    'Forms stop offering it; values already on records stay visible as "(inactive)" and never block publishing or submitting.',
  option: 'Forms stop offering it; records that have it keep it as "(inactive)".',
};

export interface DeleteDialogProps {
  recordType: RecordType;
  name: string;
  usage: Usage;
  active: boolean;
  /** Where it goes: "It moves to the recycle bin; you can restore it." (with its options). */
  unusedText: string;
  /** Deletes the record and answers its recycle-bin entry. */
  remove: () => Promise<DeleteResult>;
  /** Sets the record inactive and says so; answers whether it worked. */
  setInactive: () => Promise<boolean>;
  /** After the delete, before the Undo message: back to the list, for example. */
  onDeleted?: () => void;
  onClose: () => void;
}

/**
 * Delete for a Setting (FR-004, SC-004, plan "SettingDeleteDialog"). Unused: the question, then
 * the message with Undo. In use: where it is used and **Set inactive** instead — or, when it is
 * already inactive, why it can't be deleted and **Close**. A 409 in_use from the API (someone
 * started using it meanwhile) switches to the in-use question with the API's message.
 */
export function DeleteDialog({
  recordType,
  name,
  usage,
  active,
  unusedText,
  remove,
  setInactive,
  onDeleted,
  onClose,
}: DeleteDialogProps) {
  const [busy, setBusy] = useState(false);
  // The API's "'TV' is in use (used by 2 activities) — set it inactive instead."
  const [inUseMessage, setInUseMessage] = useState<string | null>(null);
  const afterDelete = useUndoableDelete({ invalidate: CHANGED_BY_ANY_CHANGE });
  const showFailure = useFailureToast(recordType);
  const closeOnDismiss = (open: boolean) => {
    if (!open) {
      onClose();
    }
  };

  async function confirmDelete() {
    setBusy(true);
    try {
      const result = await remove();
      onDeleted?.();
      afterDelete(result);
      onClose();
    } catch (error) {
      if (error instanceof ApiError && error.kind === 'inUse') {
        setInUseMessage(error.message);
        setBusy(false);
        return;
      }
      showFailure(error);
      onClose();
    }
  }

  async function confirmSetInactive() {
    setBusy(true);
    await setInactive();
    onClose();
  }

  if (!usage.inUse && inUseMessage === null) {
    return (
      <ConfirmDialog
        open
        onOpenChange={closeOnDismiss}
        title={`Delete ${recordType} '${name}'?`}
        confirmLabel={`Delete ${recordType}`}
        tone="danger"
        busy={busy}
        onConfirm={confirmDelete}
      >
        {unusedText}
      </ConfirmDialog>
    );
  }

  const title = `'${name}' is in use`;
  if (!active) {
    return (
      <FormDialog
        open
        onOpenChange={closeOnDismiss}
        title={title}
        submitLabel="Close"
        hideCancel
        onSubmit={onClose}
      >
        <p>{inUseMessage ?? `${usage.summary}, so it can't be deleted. It's already inactive.`}</p>
      </FormDialog>
    );
  }
  return (
    <ConfirmDialog
      open
      onOpenChange={closeOnDismiss}
      title={title}
      confirmLabel="Set inactive"
      busy={busy}
      onConfirm={confirmSetInactive}
    >
      <p>{inUseMessage ?? `${usage.summary} — set it inactive instead.`}</p>
      <p>{INACTIVE_EFFECT[recordType]}</p>
    </ConfirmDialog>
  );
}
