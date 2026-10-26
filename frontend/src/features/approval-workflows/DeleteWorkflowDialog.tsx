import { ConfirmDialog } from '@/ui/ConfirmDialog/ConfirmDialog';
import { useToast } from '@/ui/Toast/useToast';
import { useUndoableDelete } from '@/ui/useUndoableDelete';
import { asApiError, isSessionLost } from '@/api/errors';
import { WORKFLOWS_KEY, useDeleteWorkflow } from './api';
import type { ApprovalWorkflow } from './api';

interface DeleteWorkflowDialogProps {
  workflow: ApprovalWorkflow;
  /** The API says the workflow is in use after all. */
  onInUse: (workflow: ApprovalWorkflow) => void;
  onDeleted: () => void;
  onClose: () => void;
}

/** "Delete approval workflow '<name>'?" — then the message with Undo (FR-008). */
export function DeleteWorkflowDialog({
  workflow,
  onInUse,
  onDeleted,
  onClose,
}: DeleteWorkflowDialogProps) {
  const { show } = useToast();
  const deleteWorkflow = useDeleteWorkflow();
  const afterDelete = useUndoableDelete({ invalidate: [WORKFLOWS_KEY] });

  async function confirm() {
    if (deleteWorkflow.isPending) {
      return;
    }
    try {
      afterDelete(await deleteWorkflow.mutateAsync(workflow.id));
      onDeleted();
      onClose();
    } catch (error) {
      const failure = asApiError(error);
      if (failure.kind === 'inUse') {
        onInUse(workflow);
        return;
      }
      if (!isSessionLost(failure)) {
        show({ message: failure.message });
      }
      onClose();
    }
  }

  return (
    <ConfirmDialog
      open
      onOpenChange={(open) => !open && onClose()}
      title={`Delete approval workflow '${workflow.name}'?`}
      confirmLabel="Delete approval workflow"
      tone="danger"
      busy={deleteWorkflow.isPending}
      onConfirm={() => void confirm()}
    >
      It moves to the recycle bin; you can restore it.
    </ConfirmDialog>
  );
}
