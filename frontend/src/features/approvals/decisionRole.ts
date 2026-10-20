import { useActivityApproval } from './approvalsApi';
import type { ActivityApproval } from './approvalsApi';
import type { ApprovalActivity } from './approvalTypes';

/**
 * How the signed-in person meets the decision on an activity now:
 * - `decide`: they may approve or decline it and did not own or submit it — Approve and Decline
 *   are offered as buttons side by side;
 * - `owner`: the rules still let them decide (D-020, until feature 014's access rule bars it), but
 *   they own or submitted the activity — never offered as a button, only in a ⋯ menu that warns
 *   them (design system §6.3, functional review A-01);
 * - `none`: nothing to decide.
 */
export type DecisionRole = 'decide' | 'owner' | 'none';

export function decisionRole(approval: ActivityApproval): DecisionRole {
  const { approve, decline } = approval.actions;
  if (!approve.available && !decline.available) {
    return 'none';
  }
  return approval.viewer.isOwner || approval.viewer.isSubmitter ? 'owner' : 'decide';
}

/**
 * True when the signed-in person may approve or decline the Submitted activity now and does not
 * own or submit it — the activity page then makes Approve its primary action and moves Edit into
 * ⋯ (design system §6.3). False while the approval is loading and for every other status.
 */
export function useDecidesNow(activity: Pick<ApprovalActivity, 'id' | 'status'>): boolean {
  const submitted = activity.status === 'submitted';
  const approval = useActivityApproval(activity.id, submitted);
  return submitted && approval.data !== undefined && decisionRole(approval.data) === 'decide';
}
