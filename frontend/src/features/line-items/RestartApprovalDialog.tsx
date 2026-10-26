import type { components } from '@/api/generated/schema';
import { ConfirmDialog } from '@/ui/ConfirmDialog/ConfirmDialog';
import { KEEP_EDITING, RESTART_CONFIRM, RESTART_TITLE, restartText } from './lineItemText';

type ActivityStatus = components['schemas']['ActivityStatus'];

export interface RestartApprovalDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  activityName: string;
  activityStatus: ActivityStatus;
  /** The changes that restart approval: "Planned cost of 'Agency fee': 50,000.00 USD → …". */
  reasons: string[];
  onConfirm: () => void;
  busy?: boolean;
}

/**
 * "This activity will need approval again" (FR-031): what the change does to the submitted or
 * approved activity, the changes, Save and restart approval and Keep editing (focused).
 */
export function RestartApprovalDialog({
  open,
  onOpenChange,
  activityName,
  activityStatus,
  reasons,
  onConfirm,
  busy,
}: RestartApprovalDialogProps) {
  return (
    <ConfirmDialog
      open={open}
      onOpenChange={onOpenChange}
      title={RESTART_TITLE}
      confirmLabel={RESTART_CONFIRM}
      cancelLabel={KEEP_EDITING}
      onConfirm={onConfirm}
      busy={busy}
    >
      <p>{restartText(activityName, activityStatus)}</p>
      <ul>
        {reasons.map((reason) => (
          <li key={reason}>{reason}</li>
        ))}
      </ul>
    </ConfirmDialog>
  );
}
