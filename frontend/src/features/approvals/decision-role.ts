import { useActivityApproval } from './api';
import type { ActivityApproval } from './api';
import type { ApprovalActivity } from './approval-types';

type Availability = ActivityApproval['actions']['approve'];

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

/**
 * Why a person who may decide cannot approve now (038 FR-020: Refuse while line items exceed the
 * planned cost); "" when Approve is available or the person may not decide at all.
 */
export function approveRefusal(actions: { approve: Availability; decline: Availability }): string {
  const { approve, decline } = actions;
  return !approve.available && approve.reason !== '' && decline.available ? approve.reason : '';
}
