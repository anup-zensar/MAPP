import { useActivityApproval } from './approvalsApi';
import type { ApprovalActivity, ShowSubmitProblems } from './approvalTypes';
import { DecisionButton } from './DecisionButton';
import { SubmitButton } from './SubmitButton';

export interface ApprovalStatusActionProps {
  activity: ApprovalActivity;
  onShowProblems: ShowSubmitProblems;
}

/**
 * The status button in feature 006's activity header (008 plan, "Activity page parts"; FR-009,
 * FR-014): Created → **Submit for approval**; Declined → **Re-submit for approval**; Submitted →
 * **Approve**; Approved (In progress, Complete) → nothing.
 */
export function ApprovalStatusAction({ activity, onShowProblems }: ApprovalStatusActionProps) {
  switch (activity.status) {
    case 'created':
    case 'declined':
      return <SubmitButton activity={activity} onShowProblems={onShowProblems} />;
    case 'submitted':
      return <ApproveAction activityId={activity.id} />;
    default:
      return null;
  }
}

/** Approve, once the activity's approval (shared with its Approval section) has arrived. */
function ApproveAction({ activityId }: { activityId: string }) {
  const approval = useActivityApproval(activityId);
  if (!approval.data?.actions.approve.available) {
    return null;
  }
  return <DecisionButton mode="approve" approval={approval.data} />;
}
