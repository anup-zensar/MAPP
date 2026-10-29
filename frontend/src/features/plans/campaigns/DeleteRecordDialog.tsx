import { useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { ApiError } from '@/api/errors';
import { ConfirmDialog } from '@/ui/ConfirmDialog/ConfirmDialog';
import { linkedTransactionsText, useFinancialRecord } from '@/features/line-items';
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

export function DeleteRecordDialog({ target, onClose, onDeleted }: DeleteRecordDialogProps) {
  const queryClient = useQueryClient();
  const { show } = useToast();
  const remove = useDeleteRecord();
  const afterDelete = useUndoableDelete({ invalidate: PLANNING_KEYS });
  const [busy, setBusy] = useState(false);
  const financials = useFinancialRecord(target?.id ?? '', target !== null);
  const linked = linkedTransactionsText(financials.data?.figures.transactionCount ?? 0);

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
        {linked && <p>{linked}</p>}
        <p>{MOVES_TO_RECYCLE_BIN}</p>
      </div>
    </ConfirmDialog>
  );
}
