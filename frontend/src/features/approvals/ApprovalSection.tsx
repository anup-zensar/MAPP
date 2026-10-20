import { Card, ErrorState, Skeleton } from '@/components';
import { ActivityStatusChip } from '@/features/activities/ActivityStatusChip';
import { useFormatters } from '@/lib/useFormatters';
import { ApprovalAlert } from './ApprovalAlert';
import { ApprovalChain } from './ApprovalChain';
import { ApprovalHistory } from './ApprovalHistory';
import { useActivityApproval } from './approvalsApi';
import type { ActivityApproval } from './approvalsApi';
import type { ApprovalActivity, ShowSubmitProblems } from './approvalTypes';
import { DecisionButtons, OwnerDecisionMenu } from './DecisionActions';
import { decisionRole } from './decisionRole';
import { declineBoxText } from './messages';
import { RecallButton } from './RecallButton';
import { SubmitButton } from './SubmitButton';
import styles from './ApprovalSection.module.css';

export interface ApprovalSectionProps {
  activity: ApprovalActivity;
  onShowProblems: ShowSubmitProblems;
}

/**
 * The Approval section of feature 006's activity page (008 plan, "Activity page parts"; FR-016,
 * FR-021, FR-024, FR-025): the latest decline on top, the status and workflow, the approval chain
 * (or "Not submitted yet" with the levels to come), the actions and the approval history. 006's
 * page places it inside the element with id "approval".
 */
export function ApprovalSection({ activity, onShowProblems }: ApprovalSectionProps) {
  const approval = useActivityApproval(activity.id);
  const round = approval.data?.rounds[0];

  return (
    <Card
      title="Approval"
      subtitle={round ? `Round ${round.number} · ${round.workflow.name}` : undefined}
    >
      {approval.data ? (
        <ApprovalBody
          approval={approval.data}
          activity={activity}
          onShowProblems={onShowProblems}
        />
      ) : approval.isError ? (
        <ErrorState error={approval.error} onRetry={() => void approval.refetch()} />
      ) : (
        <Skeleton lines={5} height="18px" />
      )}
    </Card>
  );
}

interface ApprovalBodyProps extends ApprovalSectionProps {
  approval: ActivityApproval;
}

function ApprovalBody({ approval, activity, onShowProblems }: ApprovalBodyProps) {
  const round = approval.rounds[0];
  const { actions } = approval;
  // The submit sends the version of the activity the page shows (006's), with the latest status.
  const current: ApprovalActivity = {
    ...activity,
    name: approval.activityName,
    status: approval.status,
  };
  const role = decisionRole(approval);
  // Why an action is not offered, when the API says so; an empty reason hides it (014 FR-025).
  const reasons = [
    !actions.approve.available && approval.status === 'submitted' ? actions.approve.reason : '',
    !actions.recall.available ? actions.recall.reason : '',
  ].filter((reason, index, all) => reason !== '' && all.indexOf(reason) === index);
  const canSubmit = actions.submit.available || actions.resubmit.available;

  return (
    <div className={styles.body}>
      <ApprovalNoticeOf approval={approval} />

      <div className={styles.status}>
        <ActivityStatusChip status={approval.displayStatus} />
        <p>
          {approval.workflow
            ? `Approval workflow: ${approval.workflow.name}${approval.workflow.active ? '' : ' (inactive)'}`
            : 'No approval workflow chosen yet.'}
        </p>
      </div>

      {round ? (
        <ApprovalChain round={round} />
      ) : (
        <div className={styles.upcoming}>
          <h3 className={styles.subheading}>Not submitted yet</h3>
          {approval.upcomingLevels.length > 0 && (
            <ApprovalChain upcomingLevels={approval.upcomingLevels} />
          )}
        </div>
      )}

      {(canSubmit || role !== 'none' || actions.recall.available) && (
        <div className={styles.actions}>
          {canSubmit && (
            <SubmitButton activity={current} onShowProblems={onShowProblems} variant="secondary" />
          )}
          {role === 'decide' && <DecisionButtons approval={approval} />}
          {actions.recall.available && <RecallButton approval={approval} />}
          {role === 'owner' && <OwnerDecisionMenu approval={approval} />}
        </div>
      )}
      {reasons.map((reason) => (
        <p key={reason} className={styles.reason}>
          {reason}
        </p>
      ))}

      <ApprovalHistory rounds={approval.rounds} />
    </div>
  );
}

/**
 * The approval notice of an activity that was declined: who declined it, when, and why (design
 * system §2.4 B: an InlineAlert, never buried). Shows nothing for other statuses.
 */
export function ApprovalNotice({ activityId }: { activityId: string }) {
  const approval = useActivityApproval(activityId);
  return approval.data ? <ApprovalNoticeOf approval={approval.data} /> : null;
}

function ApprovalNoticeOf({ approval }: { approval: ActivityApproval }) {
  const { dateFormat } = useFormatters();
  const decline = approval.latestDecline;
  if (approval.status !== 'declined' || !decline) {
    return null;
  }
  return (
    <ApprovalAlert tone="danger" title={declineBoxText(decline, dateFormat)}>
      {decline.comment && <p className={styles.comment}>{decline.comment}</p>}
    </ApprovalAlert>
  );
}
