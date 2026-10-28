import type { components } from '@/api/generated/schema';
import type { ApprovalWorkflow, ApprovalWorkflowDetails } from '@/features/approval-workflows';
import { ALEX, CARLOS, DANIEL } from './approvals';

type ApprovalWorkflowLevel = components['schemas']['ApprovalWorkflowLevel'];

const META = {
  version: 4,
  createdAt: '2026-01-05T09:00:00Z',
  createdBy: { id: ALEX.id, name: ALEX.name },
  updatedAt: '2026-03-02T15:30:00Z',
  updatedBy: { id: ALEX.id, name: ALEX.name },
};

export const WORKFLOW_LEVEL_1: ApprovalWorkflowLevel = {
  number: 1,
  name: '',
  label: 'Level 1',
  approvers: [CARLOS],
};
export const WORKFLOW_LEVEL_2: ApprovalWorkflowLevel = {
  number: 2,
  name: 'Global brand',
  label: 'Global brand',
  approvers: [ALEX],
};

export function workflowOf(overrides: Partial<ApprovalWorkflow> = {}): ApprovalWorkflow {
  return {
    id: 'WF-1001',
    name: 'Lubricants EU approval',
    description: 'Every Lubricants Europe activity.',
    active: true,
    levels: [WORKFLOW_LEVEL_1, WORKFLOW_LEVEL_2],
    levelsText: 'Level 1 · Carlos Mendes → Global brand · Alex Morgan',
    usedByCount: 3,
    waitingCount: 2,
    warnings: [],
    objectType: 'activity',
    objectTypeFixed: (overrides.usedByCount ?? 3) > 0,
    ...META,
    ...overrides,
  };
}

export const CHINA_WORKFLOW = workflowOf({
  id: 'WF-1002',
  name: 'China BC approval workflow',
  description: '',
  active: false,
  levels: [{ number: 1, name: '', label: 'Level 1', approvers: [DANIEL] }],
  levelsText: 'Level 1 · Daniel Okoro',
  usedByCount: 0,
  waitingCount: 0,
  version: 2,
});

export function detailsOf(workflow = workflowOf()): ApprovalWorkflowDetails {
  return {
    workflow,
    activities: [
      {
        activity: { type: 'activity', id: 'A-1042', name: 'Launch film' },
        path: [
          { type: 'plan', id: 'P-1001', name: 'FY2026 Lubricants Europe' },
          { type: 'campaign', id: 'C-1001', name: 'Spring launch' },
          { type: 'initiative', id: 'I-1001', name: 'Brand film' },
        ],
        displayStatus: 'submitted',
        waiting: true,
      },
    ],
  };
}

const TEAM = { id: 'T-1001', name: 'Lubricants Europe', colour: 'blue', active: true } as const;
const PERSON_META = {
  version: 1,
  createdAt: '2026-01-01T09:00:00Z',
  createdBy: { id: ALEX.id, name: ALEX.name },
  updatedAt: '2026-01-01T09:00:00Z',
  updatedBy: { id: ALEX.id, name: ALEX.name },
};

const person = (id: string, name: string, initials: string, active = true) => ({
  id,
  name,
  email: `${name.toLowerCase().replace(' ', '.')}@shell.com`,
  jobTitle: '',
  initials,
  active,
  teams: [TEAM],
  ...PERSON_META,
});

export const PEOPLE = {
  body: {
    total: 4,
    items: [
      person('U-1001', 'Alex Morgan', 'AM'),
      person('U-1005', 'Carlos Mendes', 'CM'),
      person('U-1003', 'Daniel Okoro', 'DO'),
      person('U-1002', 'Priya Raman', 'PR'),
    ],
  },
};

export const RULE_OPTIONS = {
  body: {
    activityTypes: [
      { id: 'AT-1001', name: 'Event', active: true },
      { id: 'AT-1002', name: 'Sponsorship', active: true },
      { id: 'AT-1009', name: 'Print advert', active: false },
    ],
    businessUnits: [
      { id: 'OP-1001', label: 'Lubricants', active: true },
      { id: 'OP-1002', label: 'Mobility', active: true },
      { id: 'OP-1003', label: 'Retail', active: false },
    ],
    roles: [
      { id: 'RL-1002', name: 'Administrator', builtIn: 'administrator' as const },
      { id: 'RL-1005', name: 'Approver', builtIn: 'approver' as const },
    ],
    objectTypes: [
      { id: 'activity' as const, label: 'Activity' },
      { id: 'purchaseRequest' as const, label: 'Purchase request' },
      { id: 'modifiedPurchaseRequest' as const, label: 'Modified purchase request' },
    ],
    recordFields: [
      { objectType: 'purchaseRequest' as const, id: 'moaApprover', label: 'MoA approver' },
      { objectType: 'modifiedPurchaseRequest' as const, id: 'moaApprover', label: 'MoA approver' },
    ],
  },
};
