import { Card, ErrorState, Skeleton } from '@/components';
import { ActivityStatusChip } from '@/features/activities/ActivityStatusChip';
import { ApprovalChain } from './ApprovalChain';
import { ApprovalHistory } from './ApprovalHistory';
import { useActivityApproval } from './approvalsApi';
import type { ActivityApproval } from './approvalsApi';
import type { ApprovalActivity, ShowSubmitProblems } from './approvalTypes';
import { DecisionButton } from './DecisionButton';
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
  const recallReason = !actions.recall.available ? actions.recall.reason : '';

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

      {(actions.submit.available ||
        actions.resubmit.available ||
        actions.approve.available ||
        actions.decline.available ||
        actions.recall.available) && (
        <div className={styles.actions}>
          {(actions.submit.available || actions.resubmit.available) && (
            <SubmitButton activity={current} onShowProblems={onShowProblems} variant="secondary" />
          )}
          {actions.approve.available && (
            <DecisionButton mode="approve" approval={approval} variant="secondary" />
          )}
          {actions.decline.available && <DecisionButton mode="decline" approval={approval} />}
          {actions.recall.available && <RecallButton approval={approval} />}
        </div>
      )}
      {recallReason && <p className={styles.reason}>{recallReason}</p>}

      <ApprovalHistory rounds={approval.rounds} />
    </div>
  );
}
