import { ConfirmDialog } from '@/ui/ConfirmDialog/ConfirmDialog';
import { useLinkedTransactionsNotice } from '@/features/line-items';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { usePlan } from '../api';
import { contentsText } from '../plan-text';
import styles from './dialogs.module.css';

export interface DeletePlanDialogProps {
  open: boolean;
  planId: string;
  planName: string;
  onDelete: () => void;
  onClose: () => void;
  busy: boolean;
}

/** Approved activities return as Created if the plan is restored later (R14). */
export function DeletePlanDialog({
  open,
  planId,
  planName,
  onDelete,
  onClose,
  busy,
}: DeletePlanDialogProps) {
  const plan = usePlan(planId);
  const data = plan.data;
  const approved = data?.contents.approvedActivities ?? 0;
  const holds = data ? contentsText(data.contents) : '';
  const linked = useLinkedTransactionsNotice(planId, open);

  return (
    <ConfirmDialog
      open={open}
      onOpenChange={(next) => !next && onClose()}
      title={`Delete plan '${planName}'?`}
      confirmLabel="Delete plan"
      tone="danger"
      onConfirm={onDelete}
      busy={busy || linked.checking}
    >
      <div className={styles.stack}>
        {data ? (
          <>
            <p>{holds ? `It holds ${holds}.` : 'It holds nothing yet.'}</p>
            {data.status === 'published' && <p>This plan is published.</p>}
            {approved > 0 && (
              <p>
                {approved === 1
                  ? '1 of its activities is approved'
                  : `${approved} of its activities are approved`}
                ; if the plan is restored later from the recycle bin, they return as Created.
              </p>
            )}
          </>
        ) : plan.isError ? null : (
          <Skeleton lines={2} />
        )}
        {linked.text && <p>{linked.text}</p>}
        <p>It moves to the recycle bin; you can restore it.</p>
      </div>
    </ConfirmDialog>
  );
}
