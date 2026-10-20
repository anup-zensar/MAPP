import { ConfirmDialog } from '@/components';
import { formatMoneyText } from '@/lib/decimal';
import type { Activity } from './activitiesApi';
import styles from './DeleteActivityDialog.module.css';
import { lineItemsText } from './activityText';

/** What the question needs of the activity. */
export type DeletedActivity = Pick<Activity, 'name' | 'status' | 'display' | 'lineItems'>;

export interface DeleteActivityDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  activity: DeletedActivity;
  onConfirm: () => void;
  busy?: boolean;
}

/**
 * "Delete activity '<name>'?" (FR-031): what goes with it, what it does to the planned costs above
 * it and to an approval, and that it can be restored. **Cancel** has the focus; **Delete
 * activity** is red.
 */
export function DeleteActivityDialog({
  open,
  onOpenChange,
  activity,
  onConfirm,
  busy,
}: DeleteActivityDialogProps) {
  const { count } = activity.lineItems;
  // In the reader's display currency (D-220).
  const cost = activity.display.plannedCost;
  const lines = [
    count === 1
      ? '1 line item goes with it.'
      : count > 1
        ? `${lineItemsText(count)} go with it.`
        : null,
    cost
      ? `The initiative, campaign and plan planned costs go down by ${formatMoneyText(cost)}.`
      : null,
    activity.status === 'submitted' ? "It leaves the approvers' inbox." : null,
    activity.status !== 'created'
      ? "If it's restored later from the recycle bin, it comes back as Created."
      : null,
    'It moves to the recycle bin; you can restore it.',
  ].filter((line): line is string => line !== null);

  return (
    <ConfirmDialog
      open={open}
      onOpenChange={onOpenChange}
      title={`Delete activity '${activity.name}'?`}
      confirmLabel="Delete activity"
      tone="danger"
      onConfirm={onConfirm}
      busy={busy}
    >
      <div className={styles.lines}>
        {lines.map((line) => (
          <p key={line}>{line}</p>
        ))}
      </div>
    </ConfirmDialog>
  );
}
