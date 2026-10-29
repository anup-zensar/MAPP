import { useId } from 'react';
import { Card } from '@/ui/Card/Card';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { ActivityStatusChip } from '@/ui/ActivityStatusChip/ActivityStatusChip';
import { useFormatters } from '@/format/useFormatters';
import { ApprovalAlert } from './ApprovalAlert';
import { ApprovalChain } from './ApprovalChain';
import { ApprovalHistory } from './ApprovalHistory';
import { useActivityApproval } from './api';
import type { ActivityApproval } from './api';
import type { ApprovalActivity, ShowSubmitProblems } from './approval-types';
import { DecisionButtons, OwnerDecisionMenu } from './DecisionActions';
import { decisionRole } from './decision-role';
import { declineBoxText } from './decision-messages';
import { approveRefusal, OverPlannedCostAlert } from './OverPlannedCost';
import { RecallButton } from './RecallButton';
import { SubmitButton } from './SubmitButton';
import styles from './ApprovalSection.module.css';

export interface ApprovalSectionProps {
  activity: ApprovalActivity;
  onShowProblems: ShowSubmitProblems;
}

/**
 * The Approval section of feature 006's activity page (008 plan, "Activity page parts"; FR-016,
 * FR-021, FR-024, FR-025): the status and workflow, the approval chain (or "Not submitted yet" with
 * the levels to come), the actions and the approval history. The latest decline and its reason
 * show in the notice under the page header (design system §2.4 B, UI audit AC-02), so they are not
 * repeated here. 006's page places it inside the element with id "approval".
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
  const refusal = approveRefusal(actions);
  const refusalId = useId();

  return (
    <div className={styles.body}>
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

      {actions.approve.available && <OverPlannedCostAlert overages={approval.overPlannedCost} />}

      {(canSubmit || role !== 'none' || actions.recall.available) && (
        <div className={styles.actions}>
          {canSubmit && (
            <SubmitButton activity={current} onShowProblems={onShowProblems} variant="secondary" />
          )}
          {role === 'decide' && <DecisionButtons approval={approval} refusalId={refusalId} />}
          {actions.recall.available && <RecallButton approval={approval} />}
          {role === 'owner' && <OwnerDecisionMenu approval={approval} />}
        </div>
      )}
      {reasons.map((reason) => (
        <p key={reason} id={reason === refusal ? refusalId : undefined} className={styles.reason}>
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
