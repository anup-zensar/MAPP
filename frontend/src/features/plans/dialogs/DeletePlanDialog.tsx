import { ConfirmDialog, Skeleton } from '@/components';
import { usePlan } from '../plansApi';
import { contentsText } from '../planText';
import styles from './dialogs.module.css';

export interface DeletePlanDialogProps {
  open: boolean;
  planId: string;
  planName: string;
  onDelete: () => void;
  onClose: () => void;
  busy: boolean;
}

/**
 * Delete a plan (FR-029, design system §6.5): what goes with it, whether it is published or holds
 * approved activities (they return as Created if it is restored later, R14), and that it moves to
 * the recycle bin. **Cancel** has the focus; **Delete plan** is red.
 */
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

  return (
    <ConfirmDialog
      open={open}
      onOpenChange={(next) => !next && onClose()}
      title={`Delete plan '${planName}'?`}
      confirmLabel="Delete plan"
      tone="danger"
      onConfirm={onDelete}
      busy={busy}
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
        <p>It moves to the recycle bin; you can restore it.</p>
      </div>
    </ConfirmDialog>
  );
}
