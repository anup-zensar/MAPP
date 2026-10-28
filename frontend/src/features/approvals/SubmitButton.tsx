import { useQueryClient } from '@tanstack/react-query';
import { Send } from 'lucide-react';
import { useState } from 'react';
import { Button } from '@/ui/Button/Button';
import { useToast } from '@/ui/Toast/useToast';
import { useFormatters } from '@/format/useFormatters';
import { asApiError, isSessionLost } from '@/api/errors';
import { fetchSubmission, useSubmitActivity } from './api';
import type { ApprovalSubmission } from './api';
import type { ApprovalActivity, ShowSubmitProblems } from './approval-types';
import { submitConflictMessage } from './decision-messages';
import { SubmitDialog } from './SubmitDialog';

export interface SubmitButtonProps {
  activity: ApprovalActivity;
  onShowProblems: ShowSubmitProblems;
  variant?: 'primary' | 'secondary';
}

export function SubmitButton({ activity, onShowProblems, variant = 'primary' }: SubmitButtonProps) {
  const queryClient = useQueryClient();
  const { show } = useToast();
  const { dateFormat } = useFormatters();
  const submit = useSubmitActivity();
  const [checking, setChecking] = useState(false);
  const [submission, setSubmission] = useState<ApprovalSubmission | null>(null);
  const [commentError, setCommentError] = useState<string>();
  const label = activity.status === 'created' ? 'Submit for approval' : 'Re-submit for approval';

  function showFailure(error: unknown) {
    const failure = asApiError(error);
    if (!isSessionLost(failure)) {
      show({ message: failure.message });
    }
  }

  async function check() {
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
