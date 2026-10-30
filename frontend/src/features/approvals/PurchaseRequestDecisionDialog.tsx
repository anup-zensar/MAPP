import { zodResolver } from '@hookform/resolvers/zod';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';
import { asApiError, isSessionLost } from '@/api/errors';
import { ConfirmDialog } from '@/ui/ConfirmDialog/ConfirmDialog';
import { FormField } from '@/ui/FormField/FormField';
import { TextAreaField } from '@/ui/TextAreaField/TextAreaField';
import { useToast } from '@/ui/Toast/useToast';
import { CHECK_ON_SAVE } from '@/ui/useFormProblems';
import { levelOf, names } from './messages';
import { OverPlannedCostAlert } from './OverPlannedCost';
import { usePurchaseRequestDecision } from './purchase-request-api';
import type {
  PurchaseRequestApproval,
  PurchaseRequestApprovalSection,
} from './purchase-request-api';
import {
  DECLINE_COMMENT_NEEDED,
  decidedToast,
  decisionTitle,
  declineText,
  lastLevelText,
} from './purchase-request-texts';
import styles from './approval-dialogs.module.css';

const COMMENT_MAX = 2000;

const approveSchema = z.object({ comment: z.string().max(COMMENT_MAX) });
const declineSchema = z.object({
  comment: z
    .string()
    .max(COMMENT_MAX)
    .refine((comment) => comment.trim() !== '', DECLINE_COMMENT_NEEDED),
});

export interface PurchaseRequestDecisionDialogProps {
  mode: 'approve' | 'decline';
  approval: PurchaseRequestApproval;
  /** The section whose open round is decided: the one in focus. */
  section: PurchaseRequestApprovalSection;
  onClose: () => void;
  onDecided?: (approval: PurchaseRequestApproval) => void;
}

/**
 * Approve or decline the waiting level of a purchase request (032 FR-025, FR-027, FR-029): the
 * level and its approvers, the next level or what the last one does, and the comment — optional to
 * approve, needed to decline. Self-approval is allowed, so the owner is never warned off.
 */
export function PurchaseRequestDecisionDialog({
  mode,
  approval,
  section,
  onClose,
  onDecided,
}: PurchaseRequestDecisionDialogProps) {
  const { show } = useToast();
  const decide = usePurchaseRequestDecision();
  const [problem, setProblem] = useState('');
  const approve = mode === 'approve';
  const form = useForm({
    resolver: zodResolver(approve ? approveSchema : declineSchema),
    defaultValues: { comment: '' },
    ...CHECK_ON_SAVE,
  });
  const round = section.rounds[0];
  const levelNumber = round?.currentLevel ?? 0;
  const level = round?.levels.find((each) => each.number === levelNumber);
  const next = round?.levels.find(
    (each) => each.number > levelNumber && each.state === 'notReached',
  );
  const name = approval.purchaseRequestName;

  async function send(roundId: string, comment: string) {
    try {
      const result = await decide.mutateAsync({
        mode,
        purchaseRequestId: approval.purchaseRequestId,
        roundId,
        level: levelNumber,
        comment,
      });
      show({ message: decidedToast(mode, name) });
      onClose();
      onDecided?.(result);
    } catch (error) {
      const failure = asApiError(error);
      const commentProblem = failure.fields.find((field) => field.field === 'comment');
      if (failure.kind === 'validation' && commentProblem) {
        form.setError('comment', { message: commentProblem.message });
      } else if (!isSessionLost(failure)) {
        setProblem(failure.message);
      }
    }
  }

  function confirm() {
    if (round && !decide.isPending) {
      void form.handleSubmit(({ comment }) => send(round.id, comment.trim()))();
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
      title={decisionTitle(mode, name)}
      confirmLabel={approve ? 'Approve' : 'Decline'}
      tone={approve ? undefined : 'danger'}
      busy={decide.isPending}
      onConfirm={confirm}
    >
      <div className={styles.body}>
        {approve && <OverPlannedCostAlert overages={approval.overPlannedCost} />}
        {round && level && (
          <p className={styles.levels}>
            {`${levelOf(levelNumber, round.levelCount)} · ${names(level.approvers)}`}
          </p>
        )}
        {approve &&
          (next ? (
            <p>{`Next: level ${next.number} · ${names(next.approvers)}.`}</p>
          ) : (
            <p>{lastLevelText(section.kind)}</p>
          ))}
        {!approve && <p>{declineText(section.kind, approval.owner?.name ?? 'its owner')}</p>}
        {problem && (
          <p role="alert" className={styles.note}>
            {problem}
          </p>
        )}
        <FormField
          control={form.control}
          name="comment"
          render={(field) => (
            <TextAreaField
              label={approve ? 'Comment (optional)' : 'Why are you declining it?'}
              marking={approve ? undefined : 'Required'}
              rows={3}
              maxLength={COMMENT_MAX}
              {...field}
            />
          )}
        />
      </div>
    </ConfirmDialog>
  );
}
