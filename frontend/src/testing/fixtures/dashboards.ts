import type { components } from '@/api/generated/schema';

type PlanningDashboard = components['schemas']['PlanningDashboard'];
type DashboardFigures = components['schemas']['DashboardFigures'];
type Money = components['schemas']['Money'];

const usd = (amount: string): Money => ({ amount, currency: 'USD' });

const EUROPE = { id: 'T-1001', name: 'Lubricants Europe', colour: 'blue' as const, active: true };
const CHINA = { id: 'T-1002', name: 'Mobility China', colour: 'violet' as const, active: true };

export function figures(overrides: Partial<DashboardFigures> = {}): DashboardFigures {
  return {
    budget: usd('1200000.00'),
    plannedCost: usd('460000.00'),
    committed: usd('35652.17'),
    actual: usd('30652.17'),
    budgetUsedPercent: 38,
    budgetSpentPercent: 3,
    leftToPlan: usd('740000.00'),
    unspentBudget: usd('1169347.83'),
    overBudget: false,
    ...overrides,
  };
}

function months(): PlanningDashboard['months'] {
  return {
    items: Array.from({ length: 12 }, (_, i) => {
      const month = `2026-${String(i + 1).padStart(2, '0')}`;
      const planned = i === 2 ? '155166.64' : '10000.00';
      return {
        month,
        planned: usd(planned),
        actual: usd(i === 2 ? '20000.00' : '0.00'),
        plannedToDate: usd(`${(i + 1) * 10000}.00`),
        actualToDate: usd(i >= 2 ? '20000.00' : '0.00'),
      };
    }),
    noDates: usd('5000.00'),
    otherYears: null,
    committedOutsideYear: null,
    paidOutsideYear: null,
  };
}

export function dashboard(overrides: Partial<PlanningDashboard> = {}): PlanningDashboard {
  return {
    kind: 'executive',
    filters: { year: 2026, planType: null, currency: 'USD', teamId: null, planId: null },
    choices: {
      years: [2027, 2026],
      teams: [
        { id: 'T-1001', name: 'Lubricants Europe', active: true, mine: true },
        { id: 'T-1002', name: 'Mobility China', active: true, mine: false },
      ],
      currencies: [
        { code: 'EUR', name: 'Euro' },
        { code: 'USD', name: 'US dollar' },
      ],
    },
    team: null,
    planCount: 2,
    plansWithoutBudget: 0,
    finance: { visible: true, planCount: 2 },
    financeAsOf: '2026-10-27T09:00:00Z',
    totals: figures({ budget: usd('2000000.00'), plannedCost: usd('650000.00') }),
    markets: [
      { team: EUROPE, planCount: 1, figures: figures() },
      { team: CHINA, planCount: 1, figures: figures({ budget: usd('800000.00') }) },
    ],
    plans: [
      {
        plan: {
          id: 'P-1001',
          name: 'FY2026 Lubricants Europe',
          status: 'published',
          planType: 'operational',
        },
        teams: [EUROPE],
        figures: figures(),
        financeShown: true,
      },
    ],
    overBudget: [
      {
        campaign: { id: 'C-1009', name: 'Fleet push' },
        plan: { id: 'P-1001', name: 'FY2026 Lubricants Europe' },
        figures: figures({ overBudget: true }),
        difference: usd('25000.00'),
      },
    ],
    leftToPlan: [],
    planCampaigns: null,
    months: months(),
    approvals: {
      waiting: 2,
      oldestWaitingSince: '2026-10-26T09:00:00Z',
      approved: 4,
      declined: 2,
      autoApproved: 0,
      averageDecisionDays: '1.7',
      activitiesByStatus: [
        { status: 'created', count: 3 },
        { status: 'submitted', count: 2 },
      ],
    },
    purchaseRequests: {
      available: true,
      byStatus: [{ status: 'created', count: 1, amount: usd('25000.00') }],
      count: 1,
      amount: usd('25000.00'),
      waitingForGsap: 0,
    },
    completeness: [{ level: 'activity', total: 10, complete: 7, percent: 70 }],
    globalBudgets: [
      {
        id: 'GB-1001',
        name: 'FY2026 Global marketing budget',
        amount: usd('2500000.00'),
        allocated: usd('2000000.00'),
        unallocated: usd('500000.00'),
        plannedCost: usd('650000.00'),
        overAllocated: false,
      },
    ],
    canExport: true,
    ...overrides,
  };
}
