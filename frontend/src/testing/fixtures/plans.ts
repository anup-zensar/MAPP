/**
 * Plans, people, teams and Settings for the plans tests, shaped like the API's answers and the demo
 * data (FY2026 Lubricants Europe, FY2026 Mobility China, FY2025 Lubricants Europe).
 */
import type { HttpHandler } from 'msw';
import type { components } from '@/api/generated/schema';
import type { Plan, PlanDefaults, PlanList, PlanSummary } from '@/features/plans';
import { LUBRICANTS, MOBILITY, planForm } from './attribute-forms';
import { reply } from '../replies';

type Model = components['schemas']['Model'];
type Person = components['schemas']['Person'];
type Team = components['schemas']['Team'];
type TeamRef = components['schemas']['TeamRef'];

export const META = {
  version: 1,
  createdAt: '2026-03-01T09:00:00Z',
  createdBy: { id: 'U-1001', name: 'Alex Morgan' },
  updatedAt: '2026-03-10T14:30:00Z',
  updatedBy: { id: 'U-1003', name: 'Daniel Okoro' },
};

const NO_USAGE: Model['usage'] = { inUse: false, summary: '', parts: [], records: [] };

export const EUROPE: TeamRef = {
  id: 'T-1001',
  name: 'Lubricants Europe',
  colour: 'blue',
  active: true,
};
export const CHINA: TeamRef = {
  id: 'T-1002',
  name: 'Mobility China',
  colour: 'teal',
  active: true,
};

export const PRIYA_REF = {
  id: 'U-1002',
  name: 'Priya Raman',
  initials: 'PR',
  jobTitle: 'Marketer',
  active: true,
};

function person(id: string, name: string, initials: string, teams: TeamRef[]): Person {
  return {
    id,
    name,
    email: `${name.toLowerCase().replace(' ', '.')}@shell.com`,
    jobTitle: 'Marketer',
    initials,
    active: true,
    teams,
    ...META,
  };
}

export const PEOPLE = {
  total: 2,
  items: [
    person('U-1002', 'Priya Raman', 'PR', [EUROPE]),
    person('U-1004', 'Mei Lin', 'ML', [CHINA]),
  ],
};

function team(ref: TeamRef): Team {
  return { ...ref, description: '', members: [], plans: [], ...META };
}

export const TEAMS = { total: 2, items: [team(EUROPE), team(CHINA)] };

export function model(overrides: Partial<Model> = {}): Model {
  return {
    id: 'MOD-1001',
    name: 'Lubricants 1.0',
    inqToMqlRate: '2.00',
    mqlToDealRate: '5.00',
    averageDealSize: { amount: '10000.00', currency: 'USD' },
    activityTypes: [],
    active: true,
    usage: NO_USAGE,
    ...META,
    ...overrides,
  };
}

export const MODELS = {
  total: 2,
  items: [model(), model({ id: 'MOD-1002', name: 'Mobility 2.0', inqToMqlRate: '4.00' })],
};

export const FINANCE_BUDGETS = {
  total: 1,
  year: 2026,
  years: [2025, 2026, 2027],
  items: [
    {
      id: 'FB-1001',
      year: 2026,
      name: '2026 Lubricants Budget',
      items: [],
      activeItemCount: 1,
      usage: NO_USAGE,
      ...META,
    },
  ],
};

export function planDefaults(overrides: Partial<PlanDefaults> = {}): PlanDefaults {
  return {
    currentYear: 2026,
    years: [2026, 2027, 2028, 2029, 2030, 2031, 2032, 2033, 2034, 2035, 2036],
    year: 2026,
    teamIds: ['T-1001'],
    ownerId: 'U-1002',
    modelId: null,
    aop: false,
    bcpir: false,
    defaultSpread: 'evenly',
    goalStage: 'inq',
    ...overrides,
  };
}

const NO_MONEY = { amount: '0.00', currency: 'USD' };

/** FY2026 Lubricants Europe as GET /api/v1/plans/P-1001 answers it: complete, nothing inside yet. */
export function plan(overrides: Partial<Plan> = {}): Plan {
  return {
    id: 'P-1001',
    name: 'FY2026 Lubricants Europe',
    year: 2026,
    status: 'draft',
    model: {
      id: 'MOD-1001',
      name: 'Lubricants 1.0',
      active: true,
      inqToMqlRate: '2.00',
      mqlToDealRate: '5.00',
      averageDealSize: { amount: '10000.00', currency: 'USD' },
    },
    teams: [EUROPE],
    owner: PRIYA_REF,
    ownerProblem: '',
    aop: true,
    description: 'Lubricants marketing in Europe.',
    budget: { amount: '1200000.00', currency: 'USD' },
    defaultFinanceBudget: null,
    defaultSpread: 'evenly',
    bcpir: false,
    goalStage: 'inq',
    goalAmount: '250000',
    attributes: { 'SA-BU': [LUBRICANTS] },
    businessUnit: [{ id: LUBRICANTS, label: 'Lubricants', active: true }],
    figures: {
      currency: 'USD',
      budget: { amount: '1200000.00', currency: 'USD' },
      plannedCost: null,
      campaignBudgets: NO_MONEY,
      remainingBudget: { amount: '1200000.00', currency: 'USD' },
      overAllocatedBy: null,
      budgetUsedPercent: 0,
      overBudget: false,
      activities: 0,
      activitiesApproved: 0,
      approvalProgress: 'noActivities',
    },
    goals: {
      stage: 'inq',
      plan: {
        inq: 250000,
        mql: 5000,
        deals: 250,
        revenue: { amount: '2500000.00', currency: 'USD' },
        reasons: {},
      },
      activities: { inq: 0, mql: 0, revenue: NO_MONEY },
      plannedPercent: 0,
    },
    completeness: { missing: [], itemsBelow: [] },
    contents: { campaigns: 0, initiatives: 0, activities: 0, approvedActivities: 0, lineItems: 0 },
    ...META,
    ...overrides,
  };
}

/** A plan as the list shows it. */
export function summary(overrides: Partial<PlanSummary> = {}): PlanSummary {
  const full = plan();
  return {
    id: full.id,
    name: full.name,
    year: full.year,
    status: full.status,
    aop: full.aop,
    owner: full.owner,
    teams: full.teams,
    businessUnit: full.businessUnit,
    figures: full.figures,
    completeness: full.completeness,
    ...overrides,
  };
}

/** The three demo plans: newest year first, then by name. */
export function seedList(overrides: Partial<PlanList> = {}): PlanList {
  return {
    items: [
      summary({
        figures: {
          ...plan().figures,
          plannedCost: { amount: '960000.00', currency: 'USD' },
          budgetUsedPercent: 80,
          activities: 7,
          activitiesApproved: 4,
          approvalProgress: 'inProgress',
        },
      }),
      summary({
        id: 'P-1002',
        name: 'FY2026 Mobility China',
        status: 'draft',
        aop: false,
        owner: null,
        teams: [CHINA],
        businessUnit: [],
        figures: {
          ...plan().figures,
          budget: null,
          remainingBudget: null,
          budgetUsedPercent: null,
        },
        completeness: {
          missing: [{ field: 'attributes.SA-BU', label: 'Business unit' }],
          itemsBelow: [],
        },
      }),
      summary({
        id: 'P-1003',
        name: 'FY2025 Lubricants Europe',
        year: 2025,
        status: 'published',
        aop: false,
        figures: {
          ...plan().figures,
          plannedCost: { amount: '1300000.00', currency: 'USD' },
          budgetUsedPercent: 108,
          overBudget: true,
          activities: 2,
          activitiesApproved: 2,
          approvalProgress: 'allApproved',
        },
      }),
    ],
    total: 3,
    summary: {
      plans: 3,
      draft: 2,
      published: 1,
      plannedCost: { amount: '2260000.00', currency: 'USD' },
      budget: { amount: '2400000.00', currency: 'USD' },
    },
    years: [2026, 2025],
    ...overrides,
  };
}

/** What the plan form asks besides the plan: models, the attribute form, people, teams, budgets. */
export const FORM_ANSWERS: HttpHandler[] = [
  reply('GET /api/v1/models', { body: MODELS }),
  reply('GET /api/v1/shell-attribute-forms/{level}', { body: planForm() }, { level: 'plan' }),
  reply('GET /api/v1/people', { body: PEOPLE }),
  reply('GET /api/v1/teams', { body: TEAMS }),
  reply('GET /api/v1/finance-budgets', { body: FINANCE_BUDGETS }),
];

/**
 * 015's Financials of these records as an API that does not serve 015 yet answers them (404), so
 * the Financials section shows nothing and tests of the other sections need not mock it.
 */
export function financialsNotServed(...recordIds: string[]): HttpHandler[] {
  return recordIds.map((recordId) =>
    reply(
      'GET /api/v1/financials/records/{recordId}',
      { status: 404, body: { error: { code: 'not_found', message: 'Not found', fields: [] } } },
      { recordId },
    ),
  );
}

export { LUBRICANTS, MOBILITY };
