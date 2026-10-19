import { ConfirmDialog } from '@/components';
import { moneyText } from './lineItemMath';
import type { LineItem } from './lineItemsApi';
import {
  CANCEL,
  DELETE_CONFIRM,
  DELETE_RESTART,
  DELETE_TO_BIN,
  deleteEffect,
  deleteTitle,
} from './lineItemText';

export interface DeleteLineItemDialogProps {
  /** The line item to delete; null closes the dialog. */
  lineItem: LineItem | null;
  /** Adds "This activity will need approval again." (a submitted or approved activity, FR-031). */
  restartsApproval: boolean;
  busy: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

/**
 * "Delete line item 'Agency fee'?" (FR-030, design system §6.5): what it does to the unassigned
 * planned cost, that it can be restored, the re-approval line when needed, Cancel focused and the
 * red Delete line item.
 */
export function DeleteLineItemDialog({
  lineItem,
  restartsApproval,
  busy,
  onConfirm,
  onCancel,
}: DeleteLineItemDialogProps) {
  return (
    <ConfirmDialog
      open={lineItem !== null}
      onOpenChange={(open) => {
        if (!open) {
          onCancel();
        }
      }}
      title={lineItem ? deleteTitle(lineItem.name) : ''}
      confirmLabel={DELETE_CONFIRM}
      cancelLabel={CANCEL}
      tone="danger"
      busy={busy}
      onConfirm={onConfirm}
    >
      {lineItem && <p>{deleteEffect(moneyText(lineItem.plannedCostDisplay))}</p>}
      <p>{DELETE_TO_BIN}</p>
      {restartsApproval && <p>{DELETE_RESTART}</p>}
    </ConfirmDialog>
  );
}
