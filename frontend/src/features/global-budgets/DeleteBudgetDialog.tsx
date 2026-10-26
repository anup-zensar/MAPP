import { useNavigate } from 'react-router';
import { ApiError } from '@/api/errors';
import { ConfirmDialog } from '@/ui/ConfirmDialog/ConfirmDialog';
import { useToast } from '@/ui/Toast/useToast';
import { useUndoableDelete } from '@/ui/useUndoableDelete';
import { GLOBAL_BUDGETS_KEY, useDeleteGlobalBudget } from './api';
import type { GlobalBudget } from './api';

/** "Delete global budget '<name>'?" — then back to the list, with Undo (FR-035). */
export function DeleteBudgetDialog({
  budget,
  onInUse,
  onClose,
}: {
  budget: GlobalBudget;
  onInUse: (reason: string) => void;
  onClose: () => void;
}) {
  const navigate = useNavigate();
  const { show } = useToast();
  const remove = useDeleteGlobalBudget();
  const afterDelete = useUndoableDelete({ invalidate: [GLOBAL_BUDGETS_KEY] });

  async function confirm() {
    if (remove.isPending) {
      return;
    }
    try {
      afterDelete(await remove.mutateAsync(budget.id));
      onClose();
      void navigate('/budgets');
    } catch (error) {
      const apiError = error instanceof ApiError ? error : new ApiError('other');
      if (apiError.kind === 'inUse') {
        onInUse(apiError.message);
        return;
      }
      if (apiError.kind !== 'signInRequired' && apiError.kind !== 'sessionExpired') {
        show({ message: apiError.message });
      }
      onClose();
    }
  }

  return (
    <ConfirmDialog
      open
      onOpenChange={(open) => !open && onClose()}
      title={`Delete global budget '${budget.name}'?`}
      confirmLabel="Delete global budget"
      tone="danger"
      busy={remove.isPending}
      onConfirm={() => void confirm()}
    >
      It moves to the recycle bin; you can restore it.
    </ConfirmDialog>
  );
}
