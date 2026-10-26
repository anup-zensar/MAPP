/**
 * Approval records for feature 008's component tests, shaped exactly as the contract answers them
 * (the seeded demo people and the "Launch film" activity A-1042). Tests change only what they need:
 *
 *   approvalOf({ status: 'declined', latestDecline: DECLINE })
 */
import type { components } from '@/api/generated/schema';
import type {
  ActivityApproval,
  ApprovalPerson,
  ApprovalPreview,
  ApprovalRound,
  ApprovalRow,
  ApprovalSubmission,
} from './approvalsApi';

type HistoryEntry = components['schemas']['ApprovalHistoryEntry'];
type DeclineNote = components['schemas']['DeclineNote'];

export const ALEX: ApprovalPerson = {
  id: 'U-1001',
  name: 'Alex Morgan',
  initials: 'AM',
  active: true,
};
export const PRIYA: ApprovalPerson = {
  id: 'U-1002',
  name: 'Priya Raman',
  initials: 'PR',
  active: true,
};
export const DANIEL: ApprovalPerson = {
  id: 'U-1003',
  name: 'Daniel Okoro',
  initials: 'DO',
  active: true,
};
export const CARLOS: ApprovalPerson = {
  id: 'U-1005',
  name: 'Carlos Mendes',
  initials: 'CM',
  active: true,
};

export const WORKFLOW_REF = { id: 'WF-1001', name: 'Lubricants EU approval', active: true };

const actor = (person: ApprovalPerson) => ({ id: person.id, name: person.name });

/** Priya submitted "Launch film" on 10 Mar 2026 at 09:00. */
export const SUBMITTED: HistoryEntry = {
  id: 'AH-1001',
  roundNumber: 1,
  level: 1,
  levelLabel: 'Level 1',
  approvers: [CARLOS],
  actor: actor(PRIYA),
  onBehalfOf: [],
  action: 'submitted',
  at: '2026-03-10T09:00:00Z',
  comment: 'Ready for review',
  changes: [],
};

/** Round 1 of "Launch film", waiting at level 1 of 2 (Carlos, then Alex). */
export function roundOf(overrides: Partial<ApprovalRound> = {}): ApprovalRound {
  return {
    id: 'AR-1001',
    number: 1,
    outcome: 'open',
    workflow: { id: WORKFLOW_REF.id, name: WORKFLOW_REF.name },
    submittedAt: '2026-03-10T09:00:00Z',
    submittedBy: actor(PRIYA),
    comment: 'Ready for review',
    endedAt: null,
    currentLevel: 1,
    levelCount: 2,
    levels: [
      { number: 1, label: 'Level 1', approvers: [CARLOS], state: 'waiting', decision: null },
      { number: 2, label: 'Level 2', approvers: [ALEX], state: 'notReached', decision: null },
    ],
    history: [SUBMITTED],
    ...overrides,
  };
}

const NOT_AVAILABLE = { available: false, reason: '' };
const AVAILABLE = { available: true, reason: '' };

/** "Launch film" (A-1042), Submitted and waiting at level 1 for Carlos; the viewer is Priya. */
export function approvalOf(overrides: Partial<ActivityApproval> = {}): ActivityApproval {
  return {
    activityId: 'A-1042',
    activityName: 'Launch film',
    activityVersion: 3,
    status: 'submitted',
    displayStatus: 'submitted',
    ownerId: PRIYA.id,
    owner: PRIYA,
    workflow: WORKFLOW_REF,
    upcomingLevels: [],
    rounds: [roundOf()],
    latestDecline: null,
    viewer: { isOwner: true, isCurrentApprover: false, onBehalfOf: [CARLOS] },
    actions: {
      submit: NOT_AVAILABLE,
      resubmit: NOT_AVAILABLE,
      approve: AVAILABLE,
      decline: AVAILABLE,
      recall: AVAILABLE,
    },
    ...overrides,
  };
}

/** Daniel declined level 1 on behalf of Carlos on 11 Mar 2026 at 14:32. */
export const DECLINE: DeclineNote = {
  roundNumber: 1,
  level: 1,
  levelLabel: 'Level 1',
  actor: actor(DANIEL),
  onBehalfOf: [actor(CARLOS)],
  at: '2026-03-11T14:32:00Z',
  comment: 'The agency fee is missing its GL code.',
};

/** What Submit would do for the Created "Launch film": ready, two levels. */
export function submissionOf(overrides: Partial<ApprovalSubmission> = {}): ApprovalSubmission {
  return {
    activityId: 'A-1042',
    activityName: 'Launch film',
    activityVersion: 3,
    kind: 'submit',
    available: true,
    reason: '',
    roundNumber: 1,
    ready: true,
    problems: [],
    workflow: WORKFLOW_REF,
    levels: [
      { number: 1, name: '', label: 'Level 1', approvers: [CARLOS] },
      { number: 2, name: '', label: 'Level 2', approvers: [ALEX] },
    ],
    unassignedPlannedCost: null,
    ...overrides,
  };
}

const PATH = [
  { type: 'plan', id: 'P-1001', name: 'FY2026 Lubricants Europe' },
  { type: 'campaign', id: 'C-1001', name: 'Spring launch' },
  { type: 'initiative', id: 'I-1001', name: 'Brand film' },
];

/** A row of the Waiting for me tab: "Launch film", waiting at level 1 of 2 for Carlos. */
export function rowOf(overrides: Partial<ApprovalRow> = {}): ApprovalRow {
  return {
    activity: {
      id: 'A-1042',
      name: 'Launch film',
      status: 'submitted',
      displayStatus: 'submitted',
    },
    path: PATH,
    owner: PRIYA,
    plannedCost: { amount: '50000.00', currency: 'USD' },
    workflow: WORKFLOW_REF,
    round: {
      id: 'AR-1001',
      number: 1,
      outcome: 'open',
      currentLevel: 1,
      levelCount: 2,
      levelLabel: 'Level 1',
      approvers: [CARLOS],
    },
    submittedAt: '2026-03-10T09:00:00Z',
    waitingSince: '2026-03-10T09:00:00Z',
    startDate: '2026-04-01',
    startDatePassed: false,
    myDecision: null,
    ...overrides,
  };
}

/** The preview of "Launch film" with one line item. */
export function previewOf(overrides: Partial<ApprovalPreview> = {}): ApprovalPreview {
  return {
    activity: {
      id: 'A-1042',
      name: 'Launch film',
      status: 'submitted',
      displayStatus: 'submitted',
      missing: [],
      path: PATH,
      owner: PRIYA,
      startDate: '2026-04-01',
      endDate: '2026-06-30',
      ongoing: false,
      plannedCost: { amount: '50000.00', currency: 'USD' },
      typeName: 'Video production',
      description: '',
    },
    lineItems: [
      {
        id: 'L-1003',
        name: 'Agency fee',
        plannedCost: { amount: '40000.00', currency: 'EUR' },
        plannedCostDisplay: { amount: '43200.00', currency: 'USD' },
        spendType: 'Media',
        glCode: '',
        financeBudgetItem: 'Brand films',
      },
    ],
    unassignedPlannedCost: { amount: '6800.00', currency: 'USD' },
    approval: approvalOf({ viewer: { isOwner: false, isCurrentApprover: true, onBehalfOf: [] } }),
    ...overrides,
  };
}

type Delegation = components['schemas']['ApprovalDelegation'];
type Monitoring = components['schemas']['ApprovalMonitoring'];

/** Daniel delegates to Carlos (017, the seeded DLG-1001 shape); tests set dates and state. */
export function delegationOf(overrides: Partial<Delegation> = {}): Delegation {
  return {
    id: 'DLG-1001',
    person: DANIEL,
    delegate: CARLOS,
    startDate: '2026-11-03',
    endDate: '2026-11-14',
    note: 'Annual leave',
    state: 'planned',
    canChange: true,
    cancelledAt: null,
    cancelledBy: null,
    version: 1,
    createdAt: '2026-10-20T09:00:00Z',
    createdBy: actor(DANIEL),
    updatedAt: '2026-10-20T09:00:00Z',
    updatedBy: actor(DANIEL),
    ...overrides,
  };
}

/** GET /api/v1/approval-delegations without any delegation. */
export const NO_DELEGATIONS = { body: { items: [], total: 0, canManageOthers: false } };

/** The monitoring figures: one overdue approval of "Launch film", waiting 6 days at level 1. */
export function monitoringOf(overrides: Partial<Monitoring> = {}): Monitoring {
  return {
    cards: {
      overdue: 1,
      escalated: 0,
      averageDecisionDays: '2.4',
      decidedThisMonth: 3,
      autoApprovedThisMonth: 2,
    },
    ageing: [
      { bucket: 'under1', label: 'Under 1 day', count: 1 },
      { bucket: 'days1to2', label: '1–2 days', count: 0 },
      { bucket: 'days3to5', label: '3–5 days', count: 2 },
      { bucket: 'days6to10', label: '6–10 days', count: 1 },
      { bucket: 'over10', label: 'More than 10 days', count: 0 },
    ],
    byLevel: [
      {
        workflow: { id: WORKFLOW_REF.id, name: WORKFLOW_REF.name },
        level: 1,
        levelLabel: 'Level 1',
        waiting: 3,
        overdue: 1,
        oldestWaitingSince: '2026-03-10T09:00:00Z',
      },
    ],
    overdue: [rowOf({ overdue: true, waitingDays: 6 })],
    canRunEscalations: false,
    ...overrides,
  };
}
