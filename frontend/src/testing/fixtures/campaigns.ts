import type { components } from '@/api/generated/schema';
import type { Plan } from '@/features/plans';
import { plan } from './plans';

type BudgetAndCost = components['schemas']['BudgetAndCost'];
type Campaign = components['schemas']['Campaign'];
type Destination = components['schemas']['Destination'];
type DestinationList = components['schemas']['DestinationList'];
type Initiative = components['schemas']['Initiative'];
type Money = components['schemas']['Money'];
type OptionRef = components['schemas']['OptionRef'];
type PersonRef = components['schemas']['PersonRef'];
type PlanCampaigns = components['schemas']['PlanCampaigns'];

export const PLAN_ID = 'P-1001';
export const PLAN_NAME = 'FY2026 Lubricants Europe';

export function usd(amount: string): Money {
  return { amount, currency: 'USD' };
}

export const PRIYA: PersonRef = {
  id: 'U-1002',
  name: 'Priya Raman',
  initials: 'PR',
  jobTitle: 'Marketer',
  active: true,
};
export const ALEX: PersonRef = {
  id: 'U-1001',
  name: 'Alex Morgan',
  initials: 'AM',
  jobTitle: 'Head of marketing',
  active: true,
};

const META = {
  version: 3,
  createdAt: '2026-02-10T09:00:00Z',
  createdBy: { id: 'U-1001', name: 'Alex Morgan' },
  updatedAt: '2026-03-05T14:00:00Z',
  updatedBy: { id: 'U-1002', name: 'Priya Raman' },
};

export function budgetAndCost(overrides: Partial<BudgetAndCost> = {}): BudgetAndCost {
  return {
    allocatedToChildren: usd('0.00'),
    remainingForChildren: null,
    overAllocatedBy: null,
    plannedCost: null,
    budgetUsedPercent: null,
    overBudget: false,
    leftToPlan: null,
    ...overrides,
  };
}

function inUsd(
  record: Pick<Initiative, 'budget' | 'parentRemainingBudget' | 'budgetAndCost' | 'goals'>,
) {
  return {
    budget: record.budget,
    parentRemainingBudget: record.parentRemainingBudget,
    budgetAndCost: record.budgetAndCost,
    goals: record.goals,
  };
}

export function initiative(overrides: Partial<Initiative> = {}): Initiative {
  const record: Omit<Initiative, 'display'> = {
    id: 'I-1001',
    planId: PLAN_ID,
    campaignId: 'C-1001',
    name: 'Digital & social',
    description: '',
    ownerId: PRIYA.id,
    owner: PRIYA,
    startDate: '2026-02-01',
    endDate: '2026-11-30',
    budget: usd('300000.00'),
    attributes: {},
    plan: { id: PLAN_ID, name: PLAN_NAME, year: 2026 },
    campaign: {
      id: 'C-1001',
      name: 'Helix launch',
      startDate: '2026-01-01',
      endDate: '2026-12-31',
    },
    parentRemainingBudget: usd('100000.00'),
    budgetAndCost: budgetAndCost({ leftToPlan: usd('300000.00') }),
    approvalProgress: { state: 'noActivities', approved: 0, total: 0 },
    timing: 'running',
    completeness: { missing: [], itemsBelowNeedingAttention: 0 },
    goals: { inq: null, mql: null, revenue: null },
    counts: { activities: 0, approvedActivities: 0, lineItems: 0 },
    overspend: { mode: 'warn', canChange: { available: true, reason: '' } },
    ...META,
    ...overrides,
  };
  return { ...record, display: overrides.display ?? inUsd(record) };
}

export function campaign(overrides: Partial<Campaign> = {}): Campaign {
  const record: Omit<Campaign, 'display'> = {
    id: 'C-1001',
    planId: PLAN_ID,
    name: 'Helix launch',
    description: 'Launch of the new Helix range.',
    ownerId: ALEX.id,
    owner: ALEX,
    startDate: '2026-01-01',
    endDate: '2026-12-31',
    budget: usd('400000.00'),
    attributes: {},
    plan: { id: PLAN_ID, name: PLAN_NAME, year: 2026 },
    outsidePlanYear: false,
    parentRemainingBudget: usd('200000.00'),
    budgetAndCost: budgetAndCost({
      allocatedToChildren: usd('300000.00'),
      remainingForChildren: usd('100000.00'),
      leftToPlan: usd('400000.00'),
    }),
    approvalProgress: { state: 'noActivities', approved: 0, total: 0 },
    timing: 'running',
    completeness: { missing: [], itemsBelowNeedingAttention: 0 },
    goals: { inq: null, mql: null, revenue: null },
    counts: { initiatives: 1, activities: 0, approvedActivities: 0, lineItems: 0 },
    initiatives: [initiative()],
    overspend: { mode: 'warn', canChange: { available: true, reason: '' } },
    ...META,
    ...overrides,
  };
  return { ...record, display: overrides.display ?? inUsd(record) };
}

export function planCampaigns(
  items: Campaign[] = [campaign()],
  totals: Partial<PlanCampaigns['totals']> = {},
  displayTotals?: Partial<PlanCampaigns['totals']>,
): PlanCampaigns {
  const usdTotals = {
    campaigns: items.length,
    initiatives: items.reduce((sum, item) => sum + item.initiatives.length, 0),
    activities: 0,
    allocatedToCampaigns: usd('400000.00'),
    planBudget: usd('600000.00'),
    remainingPlanBudget: usd('200000.00'),
    overAllocatedBy: null,
    plannedCost: null,
    ...totals,
  };
  return {
    plan: { id: PLAN_ID, name: PLAN_NAME, year: 2026 },
    planOwner: PRIYA,
    items,
    total: items.length,
    totals: usdTotals,
    displayTotals: { ...usdTotals, ...displayTotals },
  };
}

export function planAnswer(businessUnit: OptionRef[] = [], overrides: Partial<Plan> = {}): Plan {
  const attributes: Plan['attributes'] =
    businessUnit.length > 0 ? { 'SA-BU': businessUnit.map((unit) => unit.id) } : {};
  return plan({ businessUnit, attributes, ...overrides });
}

const PERSON_META = {
  version: 1,
  createdAt: '2026-01-01T09:00:00Z',
  createdBy: { id: 'U-1001', name: 'Alex Morgan' },
  updatedAt: '2026-01-01T09:00:00Z',
  updatedBy: { id: 'U-1001', name: 'Alex Morgan' },
};

export const PEOPLE = {
  total: 2,
  items: [ALEX, PRIYA].map((person) => ({
    ...person,
    email: `${person.name.toLowerCase().replace(' ', '.')}@shell.com`,
    teams: [],
    ...PERSON_META,
  })),
};

export function destination(overrides: Partial<Destination> = {}): Destination {
  return {
    id: 'C-1002',
    name: 'Rimula fleet',
    current: false,
    available: true,
    unavailableReason: null,
    planStatus: null,
    startDate: '2026-03-01',
    endDate: '2026-12-31',
    suggestedName: null,
    remainingAfter: null,
    overAllocatedBy: null,
    consequences: null,
    ...overrides,
  };
}

export function planDestination(overrides: Partial<Destination> = {}): Destination {
  return destination({
    id: 'P-1002',
    name: 'FY2026 Mobility China',
    planStatus: 'draft',
    startDate: null,
    endDate: null,
    ...overrides,
  });
}

export function destinationList(items: Destination[], rule: string): DestinationList {
  return { items, total: items.length, rule };
}
