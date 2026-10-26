import { zodResolver } from '@hookform/resolvers/zod';
import { useForm } from 'react-hook-form';
import type { components } from '@/api/generated/schema';
import { ConfirmDialog } from '@/ui/ConfirmDialog/ConfirmDialog';
import { FormField } from '@/ui/FormField/FormField';
import { TextAreaField } from '@/ui/TextAreaField/TextAreaField';
import { CHECK_ON_SAVE } from '@/ui/useFormProblems';
import { useFormatters } from '@/format/useFormatters';
import type { ApprovalSubmission } from './api';
import { leftOutText } from './chain-messages';
import { names } from './messages';
import { commentSchema } from './schemas';
import styles from './approval-dialogs.module.css';

type SubmissionLevel = components['schemas']['ApprovalSubmissionLevel'];

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
 * "Submit 'Launch film' for approval?" (008 US1, FR-011, FR-020; 017 FR-011, FR-013, FR-014): the
 * approval workflow and what each level will do — its approvers and where they come from, "Not
 * needed" or "Approved automatically" with the rule, who is left out and why, the backup approver
 * or the advice when nobody can approve — "This activity will be approved automatically." when no
 * level needs a decision, the new round's number when re-submitting, a positive unassigned planned
 * cost as a note, and an optional comment for the approvers. Cancel has the focus.
 */
export function SubmitDialog({
  submission,
  busy,
  commentError,
  onConfirm,
  onCancel,
}: SubmitDialogProps) {
  const { money } = useFormatters();
  const form = useForm({
    resolver: zodResolver(commentSchema),
    defaultValues: { comment: '' },
    ...CHECK_ON_SAVE,
  });
  const resubmit = submission.kind === 'resubmit';
  const verb = resubmit ? 'Re-submit' : 'Submit';
  // Before 017 the API answers only the levels; then each level reads as "its approvers decide".
  const outcomes = submission.outcomes ?? [];

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
      onConfirm={() => void form.handleSubmit(({ comment }) => onConfirm(comment.trim()))()}
    >
      <div className={styles.body}>
        {submission.workflow && <p>{`Approval workflow: ${submission.workflow.name}`}</p>}
        {outcomes.length > 0 ? (
          <ol role="list" className={styles.levels}>
            {outcomes.map((level) => (
              <OutcomeItem key={level.number} level={level} />
            ))}
          </ol>
        ) : (
          submission.levels.length > 0 && (
            <ol role="list" className={styles.levels}>
              {submission.levels.map((level) => (
                <li key={level.number}>{`${level.label} · ${names(level.approvers)}`}</li>
              ))}
            </ol>
          )
        )}
        {submission.approvedAutomatically && (
          <p className={styles.strong}>This activity will be approved automatically.</p>
        )}
        {resubmit && !submission.approvedAutomatically && (
          <p>{`Round ${submission.roundNumber} starts at level 1.`}</p>
        )}
        {submission.unassignedPlannedCost && (
          <p className={styles.note}>
            {`${money(submission.unassignedPlannedCost)} of the planned cost is not yet assigned to a line item.`}
          </p>
        )}
        <FormField
          control={form.control}
          name="comment"
          render={(field) => (
            <TextAreaField
              label="Comment for the approvers (optional)"
              rows={3}
              maxLength={COMMENT_MAX}
              {...field}
              error={commentError}
            />
          )}
        />
      </div>
    </ConfirmDialog>
  );
}

/** One level of the submission: what it will do, then its notes in secondary ink. */
function OutcomeItem({ level }: { level: SubmissionLevel }) {
  const backup = level.backupUsed ? level.approvers[0] : undefined;
  const notes = [
    ...level.excluded.map(leftOutText),
    backup ? `The backup approver ${backup.name} approves this level.` : '',
    level.noApprover ? level.advice : '',
  ].filter(Boolean);

  return (
    <li className={styles.outcome}>
      <span>{outcomeText(level)}</span>
      {notes.map((note) => (
        <span key={note} className={styles.outcomeNote}>
          {note}
        </span>
      ))}
    </li>
  );
}

/**
 * "Level 1 · Daniel Okoro (line manager of Priya Raman)", "Level 2 · Not needed — only when …",
 * "Level 3 · Approved automatically — planned cost is below $10,000.00", "Level 1 · No approver".
 */
function outcomeText(level: SubmissionLevel): string {
  switch (level.outcome) {
    case 'skipped':
      return `${level.label} · Not needed — ${level.ruleText}`;
    case 'autoApproved':
      return `${level.label} · Approved automatically — ${level.ruleText}`;
    case 'decide': {
      if (level.approvers.length === 0) {
        return `${level.label} · No approver`;
      }
      const source = level.source && !level.backupUsed ? ` (${level.source})` : '';
      return `${level.label} · ${names(level.approvers)}${source}`;
    }
  }
}
