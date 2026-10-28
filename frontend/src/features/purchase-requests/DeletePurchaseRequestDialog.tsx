import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router';
import { asApiError } from '@/api/errors';
import { invalidateAllButSession } from '@/features/approvals';
import { ConfirmDialog } from '@/ui/ConfirmDialog/ConfirmDialog';
import { recordHref } from '@/ui/record-links';
import { useUndoableDelete } from '@/ui/useUndoableDelete';
import { deletePurchaseRequest, purchaseRequestKeys } from './api';
import type { PurchaseRequest } from './api';
import { DELETE_PR, deleteText, deleteTitle } from './page-text';
import styles from './PurchaseRequestPage.module.css';

export interface DeletePurchaseRequestDialogProps {
  purchaseRequest: PurchaseRequest;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

/**
 * Delete purchase request (032 FR-042, FR-055): names the lines and the line items whose money is
 * released, then moves it to the recycle bin with Undo and opens its first line item.
 */
export function DeletePurchaseRequestDialog({
  purchaseRequest,
  open,
  onOpenChange,
}: DeletePurchaseRequestDialogProps) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const afterDelete = useUndoableDelete({ invalidate: [purchaseRequestKeys.all] });
  const remove = useMutation({
    mutationFn: () => deletePurchaseRequest(purchaseRequest.id),
    onSettled: () => invalidateAllButSession(queryClient),
  });
  const lineItemIds = purchaseRequest.lines.map((line) => line.lineItem.id);
  const [firstLineItem] = lineItemIds;

  function confirm() {
    remove.mutate(undefined, {
      onSuccess: (result) => {
        onOpenChange(false);
        afterDelete(result);
        void navigate(firstLineItem ? recordHref('lineItem', firstLineItem) : '/purchase-requests');
      },
    });
  }

  return (
    <ConfirmDialog
      open={open}
      onOpenChange={onOpenChange}
      title={deleteTitle(purchaseRequest.name)}
      confirmLabel={DELETE_PR}
      tone="danger"
      onConfirm={confirm}
      busy={remove.isPending}
    >
      <p>{deleteText(lineItemIds)}</p>
      {remove.error && (
        <p role="alert" className={styles.problem}>
          {asApiError(remove.error).message}
        </p>
      )}
    </ConfirmDialog>
  );
}
