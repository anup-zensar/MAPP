/**
 * The demo global budget's answers for tests (014 plan, "Demo data"): GB-1001 "FY2026 Global
 * marketing budget", 2,500,000.00 USD, owned by Carlos Mendes, allocated to FY2026 Lubricants
 * Europe (1,200,000.00) and FY2026 Mobility China (800,000.00).
 */
import type {
  AllocationOption,
  BudgetAllocationRow,
  GlobalBudget,
  GlobalBudgetSummary,
} from '@/features/global-budgets/globalBudgetsApi';

const usd = (amount: string) => ({ amount, currency: 'USD' });
const AVAILABLE = { available: true, reason: '' };

const CARLOS = {
  id: 'U-1005',
  name: 'Carlos Mendes',
  initials: 'CM',
  jobTitle: 'Business champion',
  active: true,
};
const PRIYA = {
  id: 'U-1002',
  name: 'Priya Raman',
  initials: 'PR',
  jobTitle: 'Marketer',
  active: true,
};
const MEI = { id: 'U-1004', name: 'Mei Lin', initials: 'ML', jobTitle: 'Marketer', active: true };
const ALEX = {
  id: 'U-1001',
  name: 'Alex Morgan',
  initials: 'AM',
  jobTitle: 'Marketing administrator',
  active: true,
};

export const LUBRICANTS: BudgetAllocationRow = {
  id: 'BA-1001',
  plan: {
    id: 'P-1001',
    name: 'FY2026 Lubricants Europe',
    year: 2026,
    status: 'published',
    version: 4,
  },
  teams: [{ id: 'T-1001', name: 'Lubricants Europe', colour: 'blue', active: true }],
  owner: PRIYA,
  amount: usd('1200000.00'),
  amountUsd: usd('1200000.00'),
  plannedCost: usd('460000.00'),
  leftToPlan: usd('740000.00'),
  budgetUsedPercent: '38.33',
  version: 1,
  access: { edit: AVAILABLE, remove: AVAILABLE },
};

export const MOBILITY: BudgetAllocationRow = {
  id: 'BA-1002',
  plan: { id: 'P-1002', name: 'FY2026 Mobility China', year: 2026, status: 'draft', version: 2 },
  teams: [{ id: 'T-1002', name: 'Mobility China', colour: 'teal', active: true }],
  owner: MEI,
  amount: usd('800000.00'),
  amountUsd: usd('800000.00'),
  plannedCost: usd('210000.00'),
  leftToPlan: usd('590000.00'),
  budgetUsedPercent: '26.25',
  version: 1,
  access: { edit: AVAILABLE, remove: AVAILABLE },
};

export const BRAND_OPTION: AllocationOption = {
  plan: { id: 'P-1005', name: 'FY2026 Brand Europe', year: 2026, status: 'draft', version: 1 },
  teams: [{ id: 'T-1003', name: 'Global Brand', colour: 'violet', active: true }],
  owner: ALEX,
  budget: usd('300000.00'),
  canAllocate: AVAILABLE,
};

/** GB-1001 as Carlos sees it (US6 scenario 1); pass what differs. */
export function globalBudget(overrides: Partial<GlobalBudget> = {}): GlobalBudget {
  return {
    id: 'GB-1001',
    name: 'FY2026 Global marketing budget',
    year: 2026,
    amount: usd('2500000.00'),
    owner: CARLOS,
    description: 'Shared out to market plans by Global Marketing Ops.',
    figures: {
      amount: usd('2500000.00'),
      allocated: usd('2000000.00'),
      unallocated: usd('500000.00'),
      plannedCost: usd('670000.00'),
      overAllocated: false,
    },
    allocations: [LUBRICANTS, MOBILITY],
    otherMarkets: null,
    plansWithoutAllocation: [BRAND_OPTION],
    locks: [{ field: 'year', reason: "The year can't change once the global budget is saved." }],
    access: { edit: AVAILABLE, delete: AVAILABLE, allocate: AVAILABLE },
    version: 3,
    createdAt: '2026-10-01T09:00:00Z',
    createdBy: { id: 'U-1005', name: 'Carlos Mendes' },
    updatedAt: '2026-10-01T09:00:00Z',
    updatedBy: { id: 'U-1005', name: 'Carlos Mendes' },
    ...overrides,
  };
}

/** The list's summary of GB-1001. */
export function budgetSummary(overrides: Partial<GlobalBudgetSummary> = {}): GlobalBudgetSummary {
  const budget = globalBudget();
  return {
    id: budget.id,
    name: budget.name,
    year: budget.year,
    owner: budget.owner,
    figures: budget.figures,
    allocationCount: 2,
    ...overrides,
  };
}
