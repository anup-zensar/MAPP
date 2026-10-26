import { useState } from 'react';
import { ConfirmDialog } from '@/ui/ConfirmDialog/ConfirmDialog';
import { TextAreaField } from '@/ui/TextAreaField/TextAreaField';
import { useToast } from '@/ui/Toast/useToast';
import { useFormatters } from '@/lib/useFormatters';
import { useRecall } from './approvalsApi';
import type { ActivityApproval } from './approvalsApi';
import { alreadyDecidedMessage, asApiError, isSessionLost } from './messages';
import styles from './approvalDialogs.module.css';

/** The longest reason the API takes. */
const REASON_MAX = 2000;

export interface RecallDialogProps {
  /** The activity's approval as the person sees it: its open round. */
  approval: ActivityApproval;
  onClose: () => void;
}

/**
 * "Recall the submission of 'Launch film'?" (US5, FR-021): what a recall does and an optional
 * reason; Cancel has the focus. The recall names the round the person saw, so a round that ended
 * meanwhile (decided, restarted, recalled by someone else) changes nothing and is explained with
 * its time.
 */
export function RecallDialog({ approval, onClose }: RecallDialogProps) {
  const { show } = useToast();
  const { dateFormat } = useFormatters();
  const recall = useRecall();
  const [reason, setReason] = useState('');
  const [reasonError, setReasonError] = useState<string>();
  const round = approval.rounds[0];
  const name = approval.activityName;

  async function confirm() {
    if (!round || recall.isPending) {
      return;
    }
    try {
      await recall.mutateAsync({
        activityId: approval.activityId,
        roundId: round.id,
        reason: reason.trim(),
      });
      show({ message: `Submission of '${name}' recalled` });
      onClose();
    } catch (error) {
      const failure = asApiError(error);
      const reasonProblem = failure.fields.find((field) => field.field === 'reason');
      if (failure.kind === 'validation' && reasonProblem) {
        setReasonError(reasonProblem.message);
        return;
      }
      if (isSessionLost(failure)) {
        return;
      }
      show({
        message:
          failure.kind === 'conflict'
            ? alreadyDecidedMessage(failure, dateFormat)
            : failure.message,
      });
      // Could not reach the server, or it failed: the dialog stays, so the recall can be tried again.
      if (failure.kind !== 'unreachable' && failure.kind !== 'other') {
        onClose();
      }
    }
  }

  return (
    <ConfirmDialog
      open
      onOpenChange={(open) => {
        if (!open) {
          onClose();
        }
      }}
      title={`Recall the submission of '${name}'?`}
      confirmLabel="Recall submission"
      busy={recall.isPending}
      onConfirm={() => void confirm()}
    >
      <div className={styles.body}>
        <p>
          It goes back to Created and leaves the approvers' lists. You can submit it again later.
        </p>
        <TextAreaField
          label="Reason (optional)"
          name="reason"
          rows={3}
          maxLength={REASON_MAX}
          value={reason}
          onChange={(value) => {
            setReason(value);
            setReasonError(undefined);
          }}
          error={reasonError}
        />
      </div>
    </ConfirmDialog>
  );
}
