import { BuildingSection } from '@/features/placeholders/BuildingPage';
import type { ApprovalActivity, ShowSubmitProblems } from './approvalTypes';

export interface ApprovalSectionProps {
  activity: ApprovalActivity;
  onShowProblems: ShowSubmitProblems;
}

/**
 * The Approval section of feature 006's activity page (008 plan, "Activity page parts"): the
 * latest decline, the workflow, the approval chain, the actions and the approval history — STUB
 * of the shared foundation. Feature 008 replaces this file; 006's ActivityPage already places it
 * inside the element with id "approval".
 */
export function ApprovalSection({ activity, onShowProblems }: ApprovalSectionProps) {
  void activity;
  void onShowProblems;
  return <BuildingSection title="Approval" />;
}
