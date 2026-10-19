import { useState } from 'react';
import { ConfirmDialog, TextAreaField } from '@/components';
import { useFormatters } from '@/lib/useFormatters';
import type { ApprovalSubmission } from './approvalsApi';
import { names } from './messages';
import styles from './approvalDialogs.module.css';

/** The longest comment the API takes. */
const COMMENT_MAX = 2000;

export interface SubmitDialogProps {
  /** What Submit would do now (the API's submission check, ready). */
  submission: ApprovalSubmission;
  /** The submit is on its way. */
  busy: boolean;
  /** What is wrong with the comment, from the API. */
  commentError?: string;
  onConfirm: (comment: string) => void;
  onCancel: () => void;
}

/**
 * "Submit 'Launch film' for approval?" (US1, FR-011, FR-020): the approval workflow, each level
 * with its approvers in order, the new round's number when re-submitting, a positive unassigned
 * planned cost as a note, and an optional comment for the approvers. Cancel has the focus.
 */
export function SubmitDialog({
  submission,
  busy,
  commentError,
  onConfirm,
  onCancel,
}: SubmitDialogProps) {
  const { money } = useFormatters();
  const [comment, setComment] = useState('');
  const resubmit = submission.kind === 'resubmit';
  const verb = resubmit ? 'Re-submit' : 'Submit';

  return (
    <ConfirmDialog
      open
      onOpenChange={(open) => {
        if (!open) {
          onCancel();
        }
      }}
      title={`${verb} '${submission.activityName}' for approval?`}
      confirmLabel={`${verb} for approval`}
      busy={busy}
      onConfirm={() => onConfirm(comment.trim())}
    >
      <div className={styles.body}>
        {submission.workflow && <p>{`Approval workflow: ${submission.workflow.name}`}</p>}
        {submission.levels.length > 0 && (
          <ol role="list" className={styles.levels}>
            {submission.levels.map((level) => (
              <li key={level.number}>{`${level.label} · ${names(level.approvers)}`}</li>
            ))}
          </ol>
        )}
        {resubmit && <p>{`Round ${submission.roundNumber} starts at level 1.`}</p>}
        {submission.unassignedPlannedCost && (
          <p className={styles.note}>
            {`${money(submission.unassignedPlannedCost)} of the planned cost is not yet assigned to a line item.`}
          </p>
        )}
        <TextAreaField
          label="Comment for the approvers (optional)"
          name="comment"
          rows={3}
          maxLength={COMMENT_MAX}
          value={comment}
          onChange={setComment}
          error={commentError}
        />
      </div>
    </ConfirmDialog>
  );
}
