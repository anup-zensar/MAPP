import { zodResolver } from '@hookform/resolvers/zod';
import { useForm } from 'react-hook-form';
import { ConfirmDialog } from '@/ui/ConfirmDialog/ConfirmDialog';
import { FormField } from '@/ui/FormField/FormField';
import { TextAreaField } from '@/ui/TextAreaField/TextAreaField';
import { useToast } from '@/ui/Toast/useToast';
import { CHECK_ON_SAVE } from '@/ui/useFormProblems';
import { useFormatters } from '@/format/useFormatters';
import { useRecall } from './api';
import type { ActivityApproval } from './api';
import { alreadyDecidedMessage } from './decision-messages';
import { asApiError, isSessionLost } from './messages';
import { recallSchema } from './schemas';
import styles from './approval-dialogs.module.css';

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
  const form = useForm({
    resolver: zodResolver(recallSchema),
    defaultValues: { reason: '' },
    ...CHECK_ON_SAVE,
  });
  const round = approval.rounds[0];
  const name = approval.activityName;

  function confirm() {
    if (round && !recall.isPending) {
      void form.handleSubmit(({ reason }) => recallRound(round.id, reason.trim()))();
    }
  }

  async function recallRound(roundId: string, reason: string) {
    try {
      await recall.mutateAsync({ activityId: approval.activityId, roundId, reason });
      show({ message: `Submission of '${name}' recalled` });
      onClose();
    } catch (error) {
      const failure = asApiError(error);
      const reasonProblem = failure.fields.find((field) => field.field === 'reason');
      if (failure.kind === 'validation' && reasonProblem) {
        form.setError('reason', { message: reasonProblem.message });
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
      onConfirm={confirm}
    >
      <div className={styles.body}>
        <p>
          It goes back to Created and leaves the approvers' lists. You can submit it again later.
        </p>
        <FormField
          control={form.control}
          name="reason"
          render={(field) => (
            <TextAreaField
              label="Reason (optional)"
              rows={3}
              maxLength={REASON_MAX}
              {...field}
              onChange={(value) => {
                field.onChange(value);
                form.clearErrors('reason');
              }}
            />
          )}
        />
      </div>
    </ConfirmDialog>
  );
}
