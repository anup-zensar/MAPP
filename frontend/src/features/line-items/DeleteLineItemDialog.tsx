import { ConfirmDialog } from '@/ui/ConfirmDialog/ConfirmDialog';
import { linkedTransactionsText, useFinancialRecord } from './gsap-financials';
import { moneyText } from './line-item-math';
import type { LineItem } from './api';
import { CANCEL, DELETE_CONFIRM, DELETE_TO_BIN, deleteEffect, deleteTitle } from './line-item-text';

export interface DeleteLineItemDialogProps {
  /** The line item to delete; null closes the dialog. */
  lineItem: LineItem | null;
  busy: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

/** Asks before moving a line item to the recycle bin; the activity keeps its approval (D-322). */
export function DeleteLineItemDialog({
  lineItem,
  busy,
  onConfirm,
  onCancel,
}: DeleteLineItemDialogProps) {
  // The line item page's Financials query; without figures the dialog says nothing of them.
  const financials = useFinancialRecord(lineItem?.id ?? '', lineItem !== null);
  const linked = linkedTransactionsText(financials.data?.figures.transactionCount ?? 0);
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
      {linked && <p>{linked}</p>}
      <p>{DELETE_TO_BIN}</p>
    </ConfirmDialog>
  );
}
