import { useState } from 'react';
import { useCurrentUser } from '@/app/useSession';
import { ConfirmDialog, TextAreaField, useToast } from '@/components';
import { usePeople } from '@/features/people';
import { useFormatters } from '@/lib/useFormatters';
import { useDecide } from './approvalsApi';
import type { ActivityApproval } from './approvalsApi';
import {
  alreadyDecidedMessage,
  approvedToast,
  asApiError,
  isSessionLost,
  levelOf,
  names,
} from './messages';
import styles from './approvalDialogs.module.css';

/** The longest comment the API takes. */
const COMMENT_MAX = 2000;
const COMMENT_NEEDED = 'Add a comment explaining why the activity is declined.';

export interface DecisionDialogProps {
  mode: 'approve' | 'decline';
  /** The activity's approval as the person sees it: its open round, current level and viewer. */
  approval: ActivityApproval;
  /**
   * The owner's name when the caller knows it (the approvals page's rows and preview); null for an
   * activity without owner. Left out, the dialog finds it.
   */
  ownerName?: string | null;
  onClose: () => void;
  /** After the decision was recorded, with the activity's approval as it is now. */
  onDecided?: (approval: ActivityApproval) => void;
}

/**
 * Approve or decline the current level (US2, FR-014 – FR-018): the level and its approvers, the
 * next level or "the last level", "You own this activity.", the on-behalf sentence when the person
 * is not an approver of the level, and the comment — optional to approve, required to decline
 * (checked before anything is sent). A decision someone else made first is explained with its time.
 */
export function DecisionDialog({
  mode,
  approval,
  ownerName,
  onClose,
  onDecided,
}: DecisionDialogProps) {
  const { show } = useToast();
  const { dateFormat } = useFormatters();
  const decide = useDecide();
  const [comment, setComment] = useState('');
  const [commentError, setCommentError] = useState<string>();
  const round = approval.rounds[0];
  const levelNumber = round?.currentLevel ?? 0;
  const level = round?.levels.find((each) => each.number === levelNumber);
  const next = round?.levels.find((each) => each.number === levelNumber + 1);
  const name = approval.activityName;
  const approve = mode === 'approve';

  async function confirm() {
    if (!round || decide.isPending) {
      return;
    }
    const text = comment.trim();
    if (!approve && text === '') {
      setCommentError(COMMENT_NEEDED);
      return;
    }
    try {
      const result = await decide.mutateAsync({
        mode,
        activityId: approval.activityId,
        roundId: round.id,
        level: levelNumber,
        comment: text,
      });
      show({
        message: approve
          ? approvedToast(name, levelNumber, result.rounds[0])
          : `Activity '${name}' declined`,
      });
      onClose();
      onDecided?.(result);
    } catch (error) {
      const failure = asApiError(error);
      const commentProblem = failure.fields.find((field) => field.field === 'comment');
      if (failure.kind === 'validation' && commentProblem) {
        setCommentError(commentProblem.message);
        return;
      }
      if (isSessionLost(failure)) {
        return;
      }
      if (failure.kind === 'conflict') {
        show({ message: alreadyDecidedMessage(failure, dateFormat) });
        onClose();
        return;
      }
      show({ message: failure.message });
      // Could not reach the server, or it failed: the dialog stays, so Approve can be tried again.
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
      title={approve ? `Approve '${name}'?` : `Decline '${name}'?`}
      confirmLabel={approve ? 'Approve' : 'Decline activity'}
      tone={approve ? undefined : 'danger'}
      busy={decide.isPending}
      onConfirm={() => void confirm()}
    >
      <div className={styles.body}>
        {round && level && (
          <p className={styles.levels}>
            {`${levelOf(levelNumber, round.levelCount)} · ${names(level.approvers)}`}
          </p>
        )}
        {approve &&
          round &&
          (next ? (
            <p>{`Next: level ${next.number} · ${names(next.approvers)}.`}</p>
          ) : (
            <p>This is the last level — the activity becomes Approved.</p>
          ))}
        {!approve && <GoesBackTo approval={approval} ownerName={ownerName} />}
        {approval.viewer.isOwner && <p>You own this activity.</p>}
        {!approval.viewer.isCurrentApprover && approval.viewer.onBehalfOf.length > 0 && (
          <p className={styles.note}>
            {`You are not an approver of level ${levelNumber}. Your decision will be recorded as made on behalf of ${names(approval.viewer.onBehalfOf)}.`}
          </p>
        )}
        {approve ? (
          <TextAreaField
            label="Comment (optional)"
            name="comment"
            rows={3}
            maxLength={COMMENT_MAX}
            value={comment}
            onChange={setComment}
            error={commentError}
          />
        ) : (
          <TextAreaField
            label="Why are you declining it?"
            marking="Required"
            name="comment"
            rows={3}
            maxLength={COMMENT_MAX}
            value={comment}
            onChange={(value) => {
              setComment(value);
              setCommentError(undefined);
            }}
            error={commentError}
          />
        )}
      </div>
    </ConfirmDialog>
  );
}

interface GoesBackToProps {
  approval: ActivityApproval;
  ownerName?: string | null;
}

/** "The activity goes back to Priya Raman, who can fix it and re-submit." */
function GoesBackTo({ approval, ownerName }: GoesBackToProps) {
  const user = useCurrentUser();
  if (ownerName !== undefined || approval.viewer.isOwner || !approval.ownerId) {
    const known = approval.viewer.isOwner ? user.name : ownerName;
    return <GoesBackToText ownerName={known ?? null} />;
  }
  return <GoesBackToPerson ownerId={approval.ownerId} />;
}

/** The owner's name from the people list (shared with every person field). */
function GoesBackToPerson({ ownerId }: { ownerId: string }) {
  const people = usePeople();
  const owner = people.data?.items.find((person) => person.id === ownerId);
  return <GoesBackToText ownerName={owner?.name ?? null} />;
}

function GoesBackToText({ ownerName }: { ownerName: string | null }) {
  return (
    <p>
      {ownerName
        ? `The activity goes back to ${ownerName}, who can fix it and re-submit.`
        : 'The activity goes back to its owner, who can fix it and re-submit.'}
    </p>
  );
}
