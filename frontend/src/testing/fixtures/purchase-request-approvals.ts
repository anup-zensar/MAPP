import type { components } from '@/api/generated/schema';
import { HIDDEN, PR_WORKFLOW, YES } from './purchase-request-refs';

type S = components['schemas'];

export const APPROVAL_LEVEL: S['ApprovalWorkflowLevel'] = {
  number: 1,
  name: 'MoA approver',
  label: 'MoA approver',
  approvers: [{ id: 'U-1003', name: 'Daniel Okoro', initials: 'DO', active: true }],
  summary: 'Level 1 · MoA approver',
};

export const OPEN_ROUND: S['ApprovalRound'] = {
  id: 'AR-1101',
  kind: 'purchaseRequest',
  number: 1,
  outcome: 'open',
  workflow: { id: PR_WORKFLOW.id, name: PR_WORKFLOW.name },
  submittedAt: '2026-10-27T10:00:00Z',
  submittedBy: { id: 'U-1002', name: 'Priya Raman' },
  comment: '',
  endedAt: null,
  currentLevel: 1,
  levelCount: 1,
  levels: [
    {
      number: 1,
      label: 'MoA approver',
      approvers: APPROVAL_LEVEL.approvers,
      state: 'waiting',
      decision: null,
      decisions: [],
      requiredCount: 1,
      noApprover: false,
      advice: '',
    },
  ],
  history: [
    {
      id: 'AH-1101',
      roundNumber: 1,
      level: 1,
      levelLabel: 'MoA approver',
      approvers: APPROVAL_LEVEL.approvers,
      actor: { id: 'U-1002', name: 'Priya Raman' },
      onBehalfOf: [],
      action: 'submitted',
      at: '2026-10-27T10:00:00Z',
      comment: '',
      changes: [],
    },
  ],
};

export const APPROVAL_ACTIONS: S['PurchaseRequestApprovalActions'] = {
  submit: YES,
  resubmit: HIDDEN,
  approve: HIDDEN,
  decline: HIDDEN,
  recall: HIDDEN,
};

export function approvalSection(
  overrides: Partial<S['PurchaseRequestApprovalSection']> = {},
): S['PurchaseRequestApprovalSection'] {
  return {
    kind: 'purchaseRequest',
    workflow: PR_WORKFLOW,
    upcomingLevels: [APPROVAL_LEVEL],
    rounds: [],
    latestDecline: null,
    viewer: { isOwner: true, isCurrentApprover: false, onBehalfOf: [] },
    actions: APPROVAL_ACTIONS,
    ...overrides,
  };
}

export function purchaseRequestApproval(
  overrides: Partial<S['PurchaseRequestApproval']> = {},
): S['PurchaseRequestApproval'] {
  return {
    purchaseRequestId: 'PR-1002',
    purchaseRequestName: 'PR for Stand build',
    purchaseRequestVersion: 3,
    status: 'created',
    changePending: false,
    owner: { id: 'U-1002', name: 'Priya Raman', initials: 'PR', active: true },
    moaApprover: { id: 'U-1003', name: 'Daniel Okoro', initials: 'DO', active: true },
    sections: [approvalSection()],
    ...overrides,
  };
}

/** PR-1002 submitted, waiting at level 1 for Daniel, as Daniel sees it. */
export function waitingApproval(): S['PurchaseRequestApproval'] {
  return purchaseRequestApproval({
    status: 'submitted',
    sections: [
      approvalSection({
        rounds: [OPEN_ROUND],
        viewer: { isOwner: false, isCurrentApprover: true, onBehalfOf: [] },
        actions: { ...APPROVAL_ACTIONS, submit: HIDDEN, approve: YES, decline: YES },
      }),
    ],
  });
}

export function submission(
  overrides: Partial<S['PurchaseRequestSubmission']> = {},
): S['PurchaseRequestSubmission'] {
  return {
    purchaseRequestId: 'PR-1002',
    version: 3,
    kind: 'submit',
    roundKind: 'purchaseRequest',
    available: true,
    reason: '',
    ready: true,
    problems: [],
    workflow: PR_WORKFLOW,
    levels: [APPROVAL_LEVEL],
    ...overrides,
  };
}
