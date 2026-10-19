import type { ApprovalActivity, ShowSubmitProblems } from './approvalTypes';

export interface ApprovalStatusActionProps {
  activity: ApprovalActivity;
  onShowProblems: ShowSubmitProblems;
}

/**
 * The status button in feature 006's activity header (008 plan, "Activity page parts"): Submit
 * for approval, Re-submit for approval or Approve — STUB of the shared foundation: no button yet.
 * Feature 008 replaces this file; 006's ActivityPage already places it.
 */
export function ApprovalStatusAction({ activity, onShowProblems }: ApprovalStatusActionProps) {
  void activity;
  void onShowProblems;
  return null;
}
