import { useState } from 'react';
import { useCurrentUser } from '@/session/useSession';
import { ConfirmDialog } from '@/ui/ConfirmDialog/ConfirmDialog';
import { TextAreaField } from '@/ui/TextAreaField/TextAreaField';
import { useToast } from '@/ui/Toast/useToast';
import { useFormatters } from '@/format/useFormatters';
import { useDecide } from './approvalsApi';
import type { ActivityApproval } from './approvalsApi';
import {
  alreadyDecidedMessage,
  approvedToast,
  asApiError,
  capacitySentence,
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
  onClose: () => void;
  /** After the decision was recorded, with the activity's approval as it is now. */
  onDecided?: (approval: ActivityApproval) => void;
}

/**
 * Approve or decline the current level (008 US2, FR-014 – FR-018; 017 FR-015, FR-019): the level
 * and its approvers, the next level or "the last level", the capacity the person decides in ("You
 * approve as delegate of …", "… as backup approver of level 2", or the on-behalf sentence), on a
 * level where all must approve how many approved and whether this approval completes it, "You own
 * this activity." while the rules still let the owner decide, and the comment — optional to
 * approve, required to decline (checked before anything is sent). A decision someone else made
 * first is explained with its time.
 */
export function DecisionDialog({ mode, approval, onClose, onDecided }: DecisionDialogProps) {
  const { show } = useToast();
  const { dateFormat } = useFormatters();
  const user = useCurrentUser();
  const decide = useDecide();
  const [comment, setComment] = useState('');
  const [commentError, setCommentError] = useState<string>();
  const round = approval.rounds[0];
  const levelNumber = round?.currentLevel ?? 0;
  const level = round?.levels.find((each) => each.number === levelNumber);
  // The next level that will wait: levels not needed or approved automatically are passed (017).
  const next = round?.levels.find(
    (each) => each.number > levelNumber && each.state === 'notReached',
  );
  const name = approval.activityName;
  const approve = mode === 'approve';
  const { viewer } = approval;
  const capacity = viewer.capacity ?? null;
  const capacityText = capacitySentence(capacity, viewer.asDelegateOf ?? [], levelNumber);
  const onBehalf = capacity
    ? capacity === 'onBehalf'
    : !viewer.isCurrentApprover && viewer.onBehalfOf.length > 0;
  const allLevel = level ? allLevelLines(level, capacity, viewer, user.id) : null;
  const completes = !allLevel || allLevel.completes;

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
        {approve && allLevel && (
          <p>
            {allLevel.approved} {allLevel.after}
          </p>
        )}
        {approve &&
          round &&
          completes &&
          (next ? (
            <p>
              {next.approvers.length > 0
                ? `Next: level ${next.number} · ${names(next.approvers)}.`
                : `Next: level ${next.number}.`}
            </p>
          ) : (
            <p>This is the last level — the activity becomes Approved.</p>
          ))}
        {!approve && <GoesBackTo owner={approval.owner} />}
        {viewer.isOwner && <p>You own this activity.</p>}
        {capacityText && <p className={styles.note}>{capacityText}</p>}
        {onBehalf && viewer.onBehalfOf.length > 0 && (
          <p className={styles.note}>
            {`You are not an approver of level ${levelNumber}. Your decision will be recorded as made on behalf of ${names(viewer.onBehalfOf)}.`}
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

/**
 * "The activity goes back to Priya Raman, who can fix it and re-submit." — the owner as the API
 * names them with the approval (D-214); "its owner" for an activity without one.
 */
function GoesBackTo({ owner }: { owner: ActivityApproval['owner'] }) {
  return (
    <p>
      {owner
        ? `The activity goes back to ${owner.name}, who can fix it and re-submit.`
        : 'The activity goes back to its owner, who can fix it and re-submit.'}
    </p>
  );
}

type ChainLevel = ActivityApproval['rounds'][number]['levels'][number];

/**
 * On a level where all must approve (017 FR-015): "1 of 3 already approved." and what this
 * approval does — "Level 2 is complete after this approval." or "Still waiting for … after this
 * approval." An approval counts for the approver, or for the approver a delegate acts for; a
 * backup's or an on-behalf approval completes the level. Null on other levels.
 */
function allLevelLines(
  level: ChainLevel,
  capacity: string | null,
  viewer: ActivityApproval['viewer'],
  userId: string,
): { approved: string; after: string; completes: boolean } | null {
  const required = level.requiredCount ?? 0;
  if (level.mode !== 'all' || required < 2) {
    return null;
  }
  const approvedIds = new Set(
    (level.decisions ?? [])
      .filter((decision) => decision.action === 'approved')
      .map((decision) =>
        decision.capacity === 'delegate' && decision.asDelegateOf
          ? decision.asDelegateOf.id
          : decision.actor.id,
      ),
  );
  const countsFor =
    capacity === 'delegate' ? (viewer.asDelegateOf ?? [])[0]?.id : capacity ? null : userId;
  const completesAll = capacity === 'backup' || capacity === 'onBehalf';
  const stillWaiting = level.approvers.filter(
    (person) => !approvedIds.has(person.id) && person.id !== countsFor,
  );
  const completes = completesAll || stillWaiting.length === 0;
  const approvedCount = level.approvedCount ?? approvedIds.size;
  return {
    approved: `${approvedCount} of ${required} already approved.`,
    after: completes
      ? `Level ${level.number} is complete after this approval.`
      : `Still waiting for ${names(stillWaiting)} after this approval.`,
    completes,
  };
}
