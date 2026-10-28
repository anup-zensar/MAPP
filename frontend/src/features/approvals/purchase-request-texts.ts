import type { components } from '@/api/generated/schema';

type Status = components['schemas']['PurchaseRequestStatus'];
type RoundKind = components['schemas']['PurchaseRequestRoundKind'];

export const PR_STATUS_WORDS: Record<Status, string> = {
  created: 'Created',
  submitted: 'Submitted',
  inReview: 'In review',
  declined: 'Declined',
  approved: 'Approved',
  closed: 'Closed',
};

export const DECLINE_COMMENT_NEEDED = 'Add a comment to say why you decline.';
export const OPEN_PURCHASE_REQUEST = 'Open purchase request';
export const PURCHASE_REQUEST_GONE =
  'This purchase request is no longer here — it was deleted, or it is outside your access.';

export function decisionTitle(mode: 'approve' | 'decline', name: string): string {
  return mode === 'approve'
    ? `Approve purchase request '${name}'?`
    : `Decline purchase request '${name}'?`;
}

/** What approving the last level does: the PR, or its change, is approved and routed (FR-030). */
export function lastLevelText(kind: RoundKind): string {
  return kind === 'modifiedPurchaseRequest'
    ? 'This is the last level — the change is approved and the purchase request is routed to the requisitioner again.'
    : 'This is the last level — the purchase request becomes Approved and is routed to the requisitioner.';
}

/** What declining does (FR-025, FR-037): the money stays reserved; a change is undone. */
export function declineText(kind: RoundKind, owner: string): string {
  return kind === 'modifiedPurchaseRequest'
    ? 'The change is declined and the approved version is restored.'
    : `The purchase request goes back to ${owner}, who can fix it and re-submit. The money stays reserved.`;
}

export function decidedToast(mode: 'approve' | 'decline', name: string): string {
  return mode === 'approve'
    ? `Purchase request '${name}' approved`
    : `Purchase request '${name}' declined`;
}
