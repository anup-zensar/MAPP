import { useActivityApproval } from './api';
import type { ActivityApproval } from './api';
import type { ApprovalActivity } from './approval-types';

type Availability = ActivityApproval['actions']['approve'];

/** `owner`: the rules let them decide (D-020) but they own or submitted it, so only a warning ⋯ menu. */
export type DecisionRole = 'decide' | 'owner' | 'none';

export function decisionRole(approval: ActivityApproval): DecisionRole {
  const { approve, decline } = approval.actions;
  if (!approve.available && !decline.available) {
    return 'none';
  }
  return approval.viewer.isOwner || approval.viewer.isSubmitter ? 'owner' : 'decide';
}

export function useDecidesNow(activity: Pick<ApprovalActivity, 'id' | 'status'>): boolean {
  const submitted = activity.status === 'submitted';
  const approval = useActivityApproval(activity.id, submitted);
  return submitted && approval.data !== undefined && decisionRole(approval.data) === 'decide';
}

/** Why a person who may decide cannot approve now (038 FR-020); "" otherwise. */
export function approveRefusal(actions: { approve: Availability; decline: Availability }): string {
  const { approve, decline } = actions;
  return !approve.available && approve.reason !== '' && decline.available ? approve.reason : '';
}
