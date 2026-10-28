import type { components } from '@/api/generated/schema';

type S = components['schemas'];
type Availability = S['ActionAvailability'];

export const YES: Availability = { available: true, reason: '' };
export const HIDDEN: Availability = { available: false, reason: '' };
export function no(reason: string): Availability {
  return { available: false, reason };
}

export const PRIYA_REF: S['PersonRef'] = {
  id: 'U-1002',
  name: 'Priya Raman',
  initials: 'PR',
  jobTitle: 'Marketer',
  active: true,
};
export const DANIEL_REF: S['PersonRef'] = {
  id: 'U-1003',
  name: 'Daniel Okoro',
  initials: 'DO',
  jobTitle: 'Approver',
  active: true,
};
export const LEA_REF: S['PersonRef'] = {
  id: 'U-1007',
  name: 'Lea Santos',
  initials: 'LS',
  jobTitle: 'Requisitioner',
  active: true,
};

export const PR_WORKFLOW: S['WorkflowRef'] = {
  id: 'WF-1101',
  name: 'Purchase request approval (MoA)',
  active: true,
};
export const PR_CHANGE_WORKFLOW: S['WorkflowRef'] = {
  id: 'WF-1102',
  name: 'Purchase request change approval (MoA)',
  active: true,
};

export const PLAN_REF: S['RecordRef'] = {
  type: 'plan',
  id: 'P-1001',
  name: 'FY2026 Lubricants Europe',
};
export const PATH: S['RecordRef'][] = [
  PLAN_REF,
  { type: 'campaign', id: 'C-1001', name: 'Helix winter campaign' },
  { type: 'initiative', id: 'I-1002', name: 'Outdoor' },
  { type: 'activity', id: 'A-1003', name: 'Billboard winter wave' },
];

export function money(amount: string, currency = 'EUR'): S['Money'] {
  return { amount, currency };
}

export const STAND_BUILD: S['PurchaseRequestLineItemRef'] = {
  id: 'L-1006',
  name: 'Stand build',
  activityId: 'A-1003',
  activityName: 'Billboard winter wave',
  currency: 'EUR',
  plannedCost: money('9200.00'),
  swbsCode: 'SIM-GB01-0001',
  swbsStatus: 'received',
  companyCode: 'GB01',
  costCentre: 'GB01000105',
  glCode: '6100100',
};

export const GIFT_BOXES: S['PurchaseRequestLineItemRef'] = {
  id: 'L-1011',
  name: 'Gift boxes',
  activityId: 'A-1011',
  activityName: 'Christmas trade gifting',
  currency: 'USD',
  plannedCost: money('10000.00', 'USD'),
  swbsCode: 'SIM-GB01-0002',
  swbsStatus: 'received',
  companyCode: 'GB01',
  costCentre: 'GB01000105',
  glCode: '6100100',
};
