import { useActivityApproval } from './approvalsApi';
import type { ApprovalActivity, ShowSubmitProblems } from './approvalTypes';
import { DecisionButtons } from './DecisionActions';
import { decisionRole } from './decisionRole';
import { SubmitButton } from './SubmitButton';

export interface ApprovalStatusActionProps {
  activity: ApprovalActivity;
  onShowProblems: ShowSubmitProblems;
  /**
   * The page makes Approve its one primary action (design system §6.3: an approver of the
   * current level sees Approve as primary and Edit in ⋯). Leave it out while Edit is primary.
   */
  primaryDecision?: boolean;
}

/**
 * The status buttons in feature 006's activity header (008 plan, "Activity page parts"; FR-009,
 * FR-014): Created → **Submit for approval**; Declined → **Re-submit for approval**; Submitted →
 * **Decline** and **Approve** side by side for someone who may decide; Approved (In progress,
 * Complete) → nothing.
 *
 * The owner (or submitter) of a Submitted activity is never offered Approve as a button (design
 * system §6.3, functional review A-01): where the rules let them decide, the Approval section's ⋯
 * offers it with a warning.
 */
export function ApprovalStatusAction({
  activity,
  onShowProblems,
  primaryDecision = false,
}: ApprovalStatusActionProps) {
  switch (activity.status) {
    case 'created':
    case 'declined':
      return (
        <SubmitButton activity={activity} onShowProblems={onShowProblems} variant="secondary" />
      );
    case 'submitted':
      return <DecideActions activityId={activity.id} primary={primaryDecision} />;
    default:
      return null;
  }
}

/** Decline and Approve, once the activity's approval (shared with its section) has arrived. */
function DecideActions({ activityId, primary }: { activityId: string; primary: boolean }) {
  const approval = useActivityApproval(activityId);
  if (!approval.data || decisionRole(approval.data) !== 'decide') {
    return null;
  }
  return <DecisionButtons approval={approval.data} primary={primary} />;
}
