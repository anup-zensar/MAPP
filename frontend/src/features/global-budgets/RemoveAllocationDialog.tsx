import { ApiError } from '@/api/errors';
import { ConfirmDialog } from '@/ui/ConfirmDialog/ConfirmDialog';
import { useToast } from '@/ui/Toast/useToast';
import { useFormatters } from '@/format/useFormatters';
import { useRemoveAllocation } from './api';
import type { BudgetAllocationRow } from './api';
import { removeEffect, removeQuestion, toasts } from './global-budgets-text';

/** "Remove the allocation to 'FY2026 Mobility China'?" — the plan keeps its budget (FR-034). */
export function RemoveAllocationDialog({
  row,
  onClose,
}: {
  row: BudgetAllocationRow;
  onClose: () => void;
}) {
  const { show } = useToast();
  const { money } = useFormatters();
  const remove = useRemoveAllocation();

  async function confirm() {
    if (remove.isPending) {
      return;
    }
    try {
      await remove.mutateAsync(row.id);
      show({ message: toasts.removed(row.plan.name) });
    } catch (error) {
      const apiError = error instanceof ApiError ? error : new ApiError('other');
      if (apiError.kind !== 'signInRequired' && apiError.kind !== 'sessionExpired') {
        show({ message: apiError.message });
      }
    } finally {
      onClose();
    }
  }

  return (
    <ConfirmDialog
      open
      onOpenChange={(open) => !open && onClose()}
      title={removeQuestion(row.plan.name)}
      confirmLabel="Remove allocation"
      tone="danger"
      busy={remove.isPending}
      onConfirm={() => void confirm()}
    >
      {removeEffect(row.plan.name, money(row.amount))}
    </ConfirmDialog>
  );
}
