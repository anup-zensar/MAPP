import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router';
import { asApiError } from '@/api/errors';
import { invalidateAllButSession } from '@/features/approvals';
import { useAccess } from '@/session/access';
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
 * Delete purchase request (032 FR-042, FR-055; 038 FR-013): names the lines and the line items whose
 * money is released, then moves it to the recycle bin with Undo and opens Purchase requests (its
 * first line item when the person may not see that page). The deleted record is never asked for
 * again: its queries are dropped and the lists refreshed only once the page has gone.
 */
export function DeletePurchaseRequestDialog({
  purchaseRequest,
  open,
  onOpenChange,
}: DeletePurchaseRequestDialogProps) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { has } = useAccess();
  const afterDelete = useUndoableDelete({ invalidate: [purchaseRequestKeys.all] });
  const remove = useMutation({ mutationFn: () => deletePurchaseRequest(purchaseRequest.id) });
  const lineItemIds = purchaseRequest.lines.map((line) => line.lineItem.id);
  const [firstLineItem] = lineItemIds;

  function landing(): string {
    if (has('purchaseRequests.list') || !firstLineItem) {
      return '/purchase-requests';
    }
    return recordHref('lineItem', firstLineItem);
  }

  async function confirm() {
    try {
      const result = await remove.mutateAsync();
      queryClient.removeQueries({ predicate: (query) => isOf(query.queryKey, purchaseRequest.id) });
      onOpenChange(false);
      await navigate(landing());
      afterDelete(result);
    } catch {
      // The refusal is shown in the dialog (remove.error).
    }
    await invalidateAllButSession(queryClient);
  }

  return (
    <ConfirmDialog
      open={open}
      onOpenChange={onOpenChange}
      title={deleteTitle(purchaseRequest.name)}
      confirmLabel={DELETE_PR}
      tone="danger"
      onConfirm={() => void confirm()}
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

/** Whether a query is one of the purchase request's own: its page, approval, submission and so on. */
function isOf(queryKey: readonly unknown[], id: string): boolean {
  return queryKey[0] === purchaseRequestKeys.all[0] && queryKey.includes(id);
}
