import { useQueryClient } from '@tanstack/react-query';
import { Send } from 'lucide-react';
import { useState } from 'react';
import { Button, useToast } from '@/components';
import { useFormatters } from '@/lib/useFormatters';
import { fetchSubmission, useSubmitActivity } from './approvalsApi';
import type { ApprovalSubmission } from './approvalsApi';
import type { ApprovalActivity, ShowSubmitProblems } from './approvalTypes';
import { asApiError, isSessionLost, submitConflictMessage } from './messages';
import { SubmitDialog } from './SubmitDialog';

export interface SubmitButtonProps {
  /** A Created activity (Submit) or a Declined one (Re-submit). */
  activity: ApprovalActivity;
  /** 006's function that shows "Submitting needs <n> things — view list" and lists them. */
  onShowProblems: ShowSubmitProblems;
  variant?: 'primary' | 'secondary';
}

/**
 * **Submit for approval** / **Re-submit for approval** and what follows (US1, FR-009 – FR-011,
 * FR-020): the API checks the activity afresh; with problems, 006's Ready to submit list shows
 * every one of them at once; otherwise the confirmation shows the workflow and its levels, and the
 * submit sends the version of the activity the person saw.
 */
export function SubmitButton({ activity, onShowProblems, variant = 'primary' }: SubmitButtonProps) {
  const queryClient = useQueryClient();
  const { show } = useToast();
  const { dateFormat } = useFormatters();
  const submit = useSubmitActivity();
  const [checking, setChecking] = useState(false);
  const [submission, setSubmission] = useState<ApprovalSubmission | null>(null);
  const [commentError, setCommentError] = useState<string>();
  const label = activity.status === 'declined' ? 'Re-submit for approval' : 'Submit for approval';

  function showFailure(error: unknown) {
    const failure = asApiError(error);
    if (!isSessionLost(failure)) {
      show({ message: failure.message });
    }
  }

  async function check() {
    // One check at a time: another click while it is on its way is ignored.
    if (checking) {
      return;
    }
    setChecking(true);
    try {
      const found = await fetchSubmission(queryClient, activity.id);
      if (!found.available) {
        show({ message: found.reason });
      } else if (!found.ready) {
        onShowProblems(found.problems.length);
      } else {
        setCommentError(undefined);
        setSubmission(found);
      }
    } catch (error) {
      showFailure(error);
    } finally {
      setChecking(false);
    }
  }

  async function confirm(comment: string) {
    if (!submission || submit.isPending) {
      return;
    }
    try {
      const result = await submit.mutateAsync({
        activityId: activity.id,
        version: activity.version,
        comment,
      });
      const verb = submission.kind === 'resubmit' ? 're-submitted' : 'submitted';
      // No level needed a decision: the rules approved it at once (017 FR-014).
      const automatic = submission.approvedAutomatically || result.status === 'approved';
      show({
        message: automatic
          ? `Activity '${submission.activityName}' approved automatically`
          : `Activity '${submission.activityName}' ${verb} for approval`,
      });
      setSubmission(null);
    } catch (error) {
      const failure = asApiError(error);
      const commentProblem = failure.fields.find((field) => field.field === 'comment');
      if (failure.kind === 'validation' && commentProblem) {
        setCommentError(commentProblem.message);
        return;
      }
      setSubmission(null);
      if (failure.kind === 'validation') {
        // Something changed since the check: the list shows the problems as they are now.
        onShowProblems(failure.fields.length);
      } else if (failure.kind === 'conflict') {
        show({ message: submitConflictMessage(failure, dateFormat) });
      } else {
        showFailure(failure);
      }
    }
  }

  return (
    <>
      <Button
        variant={variant}
        icon={<Send />}
        aria-busy={checking || undefined}
        onClick={() => void check()}
      >
        {label}
      </Button>
      {submission && (
        <SubmitDialog
          submission={submission}
          busy={submit.isPending}
          commentError={commentError}
          onConfirm={(comment) => void confirm(comment)}
          onCancel={() => setSubmission(null)}
        />
      )}
    </>
  );
}
