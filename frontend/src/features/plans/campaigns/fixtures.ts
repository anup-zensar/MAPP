/**
 * Contract-shaped answers for this feature's tests (typed with the API schema, so a contract
 * change shows here first). Only test files import this file.
 */
import type { components } from '@/api/generated/schema';
import type {
  Campaign,
  Destination,
  DestinationList,
  Initiative,
  Money,
  PersonRef,
  PlanCampaigns,
} from './api';

type BudgetAndCost = components['schemas']['BudgetAndCost'];

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

/**
 * A record's amounts as the API answers them to a reader whose display currency is USD: the same
 * amounts again (D-220). Tests of another display currency pass `display` themselves.
 */
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
    // A reader in USD gets the same totals again (D-220).
    displayTotals: { ...usdTotals, ...displayTotals },
  };
}

/** The part of feature 004's GET /api/v1/plans/{planId} answer the forms read: the business unit. */
export function planAnswer(businessUnit: { id: string; label: string }[] = []) {
  return { id: PLAN_ID, name: PLAN_NAME, year: 2026, businessUnit };
}

const PERSON_META = {
  version: 1,
  createdAt: '2026-01-01T09:00:00Z',
  createdBy: { id: 'U-1001', name: 'Alex Morgan' },
  updatedAt: '2026-01-01T09:00:00Z',
  updatedBy: { id: 'U-1001', name: 'Alex Morgan' },
};

/** GET /api/v1/people: Alex Morgan and Priya Raman. */
export const PEOPLE = {
  total: 2,
  items: [ALEX, PRIYA].map((person) => ({
    ...person,
    email: `${person.name.toLowerCase().replace(' ', '.')}@shell.com`,
    teams: [],
    ...PERSON_META,
  })),
};

/** A campaign to move or copy an initiative to (GET …/destinations); available, no warnings. */
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

/** A plan to move or copy a campaign to. */
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
