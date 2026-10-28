import type { RoundKind } from './api';

export const APPROVAL_TITLE = 'Approval';
export const NOT_SUBMITTED = 'Not submitted yet';
export const SUBMIT = 'Submit for approval';
export const RESUBMIT = 'Re-submit';
export const RECALL = 'Recall';
export const APPROVE = 'Approve';
export const DECLINE = 'Decline';
export const CLOSE = 'Close';
export const NEEDS_FIRST = 'Submitting needs these things first:';
export const COMMENT_LABEL = 'Comment for the approvers (optional)';
export const REASON_LABEL = 'Reason (optional)';
export const NO_WORKFLOW = 'No approval workflow chosen yet.';

export const SECTION_TITLES: Record<RoundKind, string> = {
  purchaseRequest: 'First approval',
  modifiedPurchaseRequest: 'Approval of changes',
};

export function workflowText(workflow: { name: string; active: boolean } | null): string {
  if (!workflow) {
    return NO_WORKFLOW;
  }
  return `Approval workflow: ${workflow.name}${workflow.active ? '' : ' (inactive)'}`;
}

export function submitTitle(resubmit: boolean, name: string): string {
  return `${resubmit ? 'Re-submit' : 'Submit'} '${name}' for approval?`;
}

export function notReadyTitle(name: string): string {
  return `'${name}' can't be submitted yet`;
}

export function submittedToast(resubmit: boolean, name: string): string {
  return `Purchase request '${name}' ${resubmit ? 're-submitted' : 'submitted'} for approval`;
}

export function recallTitle(kind: RoundKind, name: string): string {
  return kind === 'modifiedPurchaseRequest'
    ? `Recall the change of '${name}'?`
    : `Recall the submission of '${name}'?`;
}

/** What a recall does (FR-024): a first round returns to Created, a change restores the approved version. */
export function recallText(kind: RoundKind): string {
  return kind === 'modifiedPurchaseRequest'
    ? 'The open level is cancelled. The approved version is restored.'
    : 'The open level is cancelled. It returns to Created.';
}

export function recalledToast(kind: RoundKind, name: string): string {
  return kind === 'modifiedPurchaseRequest'
    ? `Change of '${name}' recalled`
    : `Submission of '${name}' recalled`;
}
