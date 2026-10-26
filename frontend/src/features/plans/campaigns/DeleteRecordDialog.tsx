import { useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { ApiError } from '@/api/errors';
import { ConfirmDialog } from '@/ui/ConfirmDialog/ConfirmDialog';
import { useToast } from '@/ui/Toast/useToast';
import { useUndoableDelete } from '@/ui/useUndoableDelete';
import { PLANNING_KEYS, refreshPlanning, useDeleteRecord } from './api';
import type { RecordTarget } from './record-target';
import styles from './DeleteRecordDialog.module.css';
import {
  approvedSentence,
  contentsSentence,
  deleteLabel,
  deleteTitle,
  MOVES_TO_RECYCLE_BIN,
  PLAN_TOTALS_WILL_UPDATE,
  withItGo,
} from './texts';

export interface DeleteRecordDialogProps {
  /** The campaign or initiative to delete; null keeps the dialog closed. */
  target: RecordTarget | null;
  onClose: () => void;
  /** After the delete, before the Undo message: a panel of the record closes here. */
  onDeleted?: () => void;
}

/**
 * "Delete campaign 'Helix launch'?" (US5, FR-028, FR-029, design system §6.5): what goes with it,
 * that plan totals update, that approved activities come back as Created (R14), and that it moves
 * to the recycle bin. Cancel has the focus; the red button names the action. After the delete,
 * the message offers Undo for 10 seconds.
 */
export function DeleteRecordDialog({ target, onClose, onDeleted }: DeleteRecordDialogProps) {
  const queryClient = useQueryClient();
  const { show } = useToast();
  const remove = useDeleteRecord();
  const afterDelete = useUndoableDelete({ invalidate: PLANNING_KEYS });
  const [busy, setBusy] = useState(false);

  if (!target) {
    return null;
  }
  const { level, id, name, counts } = target;
  const contents = contentsSentence(counts);

  async function confirm() {
    setBusy(true);
    try {
      const result = await remove.mutateAsync({ level, id });
      onClose();
      onDeleted?.();
      afterDelete(result);
    } catch (error) {
      // Gone already, or refused: the API's message ("This campaign no longer exists …").
      onClose();
      show({ message: (error instanceof ApiError ? error : new ApiError('other')).message });
    } finally {
      setBusy(false);
      await refreshPlanning(queryClient);
    }
  }

  return (
    <ConfirmDialog
      open
      onOpenChange={(open) => !open && onClose()}
      title={deleteTitle(level, name)}
      confirmLabel={deleteLabel(level)}
      tone="danger"
      busy={busy}
      onConfirm={() => void confirm()}
    >
      <div className={styles.text}>
        {contents && <p>{withItGo(contents)}</p>}
        <p>{PLAN_TOTALS_WILL_UPDATE}</p>
        {counts.approvedActivities > 0 && <p>{approvedSentence(counts.approvedActivities)}</p>}
        <p>{MOVES_TO_RECYCLE_BIN}</p>
      </div>
    </ConfirmDialog>
  );
}
