/**
 * Answers of the Home endpoints for tests (feature 009). `HOME_ANSWERS` answers every request Home
 * makes with an empty dashboard, for tests of other parts of the app that land on Home.
 */
import type { components } from '@/api/schema';
import type { MockAnswer } from '@/test/mockApi';
import type {
  ActivityLogEntry,
  HomeAttentionItem,
  HomeAttentionReason,
  HomePlanRow,
  HomeSummary,
  Money,
  MyActivityGroup,
  MyActivityRow,
  WaitingItem,
} from './homeApi';

type Currency = components['schemas']['Currency'];
type Team = components['schemas']['Team'];

/** An amount in US dollars ("1200.00" → $1,200.00). */
export function usd(amount: string): Money {
  return { amount, currency: 'USD' };
}

/** Priya's summary with the seed (plan "Demo data"), in US dollars. */
export function summaryOf(overrides: Partial<HomeSummary> = {}): HomeSummary {
  return {
    scope: 'myTeams',
    displayCurrency: 'USD',
    unavailableCurrency: null,
    gettingStarted: null,
    plans: { count: 1, published: 1, draft: 0 },
    activitiesOwned: { count: 6, declined: 1, submitted: 2 },
    waitingForMe: { count: 0, oldestWaitingSince: null },
    plannedCost: {
      planCount: 1,
      plannedCost: usd('1003000.00'),
      planBudget: usd('1200000.00'),
      budgetUsedPercent: 84,
      overBudget: false,
      plansWithoutBudget: 0,
    },
    ...overrides,
  };
}

/** A currency as GET /api/currencies answers it. */
export function currencyOf(code: string, name: string, active = true): Currency {
  const alex = { id: 'U-1001', name: 'Alex Morgan' };
  return {
    code,
    name,
    symbol: '',
    rate: '1',
    inverseRate: '1',
    rateChangedAt: '2026-01-05T09:00:00Z',
    rateChangedBy: alex,
    active,
    usage: { inUse: false, summary: 'Not used yet', parts: [], records: [] },
    locks: [],
    version: 1,
    createdAt: '2026-01-05T09:00:00Z',
    createdBy: alex,
    updatedAt: '2026-01-05T09:00:00Z',
    updatedBy: alex,
  };
}

/** USD, EUR and GBP active; CNY inactive. */
export const CURRENCIES: Currency[] = [
  currencyOf('USD', 'US dollar'),
  currencyOf('CNY', 'Chinese yuan', false),
  currencyOf('EUR', 'Euro'),
  currencyOf('GBP', 'Pound sterling'),
];

/** A row of Plans at a glance: FY2026 Lubricants Europe with the seed's figures. */
export function planRowOf(overrides: Partial<HomePlanRow> = {}): HomePlanRow {
  return {
    id: 'P-1001',
    name: 'FY2026 Lubricants Europe',
    year: 2026,
    strategicAop: false,
    status: 'published',
    teams: [{ id: 'T-1001', name: 'Lubricants Europe', mine: true }],
    campaignCount: 3,
    activityCount: 7,
    approvedActivityCount: 3,
    budget: usd('1200000.00'),
    plannedCost: usd('1003000.00'),
    budgetUsedPercent: 84,
    overBudget: false,
    ...overrides,
  };
}

/** A team as GET /api/teams answers it. */
export function teamOf(id: string, name: string, colour: Team['colour']): Team {
  const alex = { id: 'U-1001', name: 'Alex Morgan' };
  return {
    id,
    name,
    description: '',
    colour,
    active: true,
    members: [],
    plans: [],
    version: 1,
    createdAt: '2026-01-05T09:00:00Z',
    createdBy: alex,
    updatedAt: '2026-01-05T09:00:00Z',
    updatedBy: alex,
  };
}

/** The seed's teams (plan "Demo data"). */
export const TEAMS: Team[] = [
  teamOf('T-1003', 'Global Marketing Ops', 'violet'),
  teamOf('T-1001', 'Lubricants Europe', 'blue'),
  teamOf('T-1002', 'Mobility China', 'teal'),
];

/** The plan of the seed's Lubricants Europe activities. */
export const LUBRICANTS_PLAN = { type: 'plan', id: 'P-1001', name: 'FY2026 Lubricants Europe' };

/** A reason a record needs attention; the fields that do not apply are empty. */
export function reasonOf(
  kind: HomeAttentionReason['kind'],
  overrides: Partial<HomeAttentionReason> = {},
): HomeAttentionReason {
  return {
    kind,
    amount: null,
    declinedBy: null,
    declinedAt: null,
    comment: null,
    missing: [],
    itemsBelow: 0,
    ...overrides,
  };
}

/** Priya's declined activity "Launch film" (US3 scenario 1). */
export function attentionItemOf(overrides: Partial<HomeAttentionItem> = {}): HomeAttentionItem {
  return {
    recordType: 'activity',
    recordTypeLabel: 'activity',
    id: 'A-1042',
    name: 'Launch film',
    plan: LUBRICANTS_PLAN,
    path: [
      LUBRICANTS_PLAN,
      { type: 'campaign', id: 'C-1001', name: 'Helix launch' },
      { type: 'initiative', id: 'I-1001', name: 'Digital & social' },
    ],
    changedAt: '2026-03-12T08:00:00Z',
    reasons: [
      reasonOf('declined', {
        declinedBy: { id: 'U-1003', name: 'Daniel Okoro' },
        declinedAt: '2026-03-12T10:30:00Z',
        comment: 'Please cut the agency fee',
      }),
    ],
    ...overrides,
  };
}

/** An activity waiting for Daniel at level 1 of 1 (US3 scenario 5). */
export function waitingItemOf(overrides: Partial<WaitingItem> = {}): WaitingItem {
  return {
    id: 'A-1045',
    name: 'Dealer summit',
    plan: LUBRICANTS_PLAN,
    owner: { id: 'U-1002', name: 'Priya Raman' },
    plannedCost: usd('48000.00'),
    level: 1,
    levels: 1,
    waitingSince: '2026-03-09T09:00:00Z',
    ...overrides,
  };
}

/** One of Priya's activities in My activities: "Launch film", approved, with dates. */
export function myActivityOf(overrides: Partial<MyActivityRow> = {}): MyActivityRow {
  return {
    id: 'A-1042',
    name: 'Launch film',
    plan: LUBRICANTS_PLAN,
    campaign: { type: 'campaign', id: 'C-1001', name: 'Helix launch' },
    initiative: { type: 'initiative', id: 'I-1001', name: 'Digital & social' },
    status: 'approved',
    startDate: '2026-03-01',
    endDate: '2026-05-31',
    ongoing: false,
    plannedCost: usd('85000.00'),
    missing: [],
    itemsBelow: 0,
    ...overrides,
  };
}

/** A group of My activities holding the given rows. */
export function groupOf(
  status: MyActivityGroup['status'],
  items: MyActivityRow[],
): MyActivityGroup {
  return { status, count: items.length, items: items.map((item) => ({ ...item, status })) };
}

/** An activity-log entry: Priya changed the planned cost of "Launch film" (US4 scenario 3). */
export function logEntryOf(overrides: Partial<ActivityLogEntry> = {}): ActivityLogEntry {
  return {
    id: 'LOG-1001',
    seq: 1,
    at: '2026-03-12T08:59:30Z',
    actor: { id: 'U-1002', name: 'Priya Raman' },
    action: 'changed',
    verb: 'changed',
    recordType: 'activity',
    recordTypeLabel: 'activity',
    recordId: 'A-1042',
    recordName: 'Launch film',
    path: [LUBRICANTS_PLAN],
    changes: [
      {
        field: 'plannedCost',
        label: 'Planned cost',
        old: '80000.00',
        new: '85000.00',
        oldText: '$80,000.00',
        newText: '$85,000.00',
      },
    ],
    comment: '',
    details: '',
    items: [],
    recordNow: { state: 'exists', name: 'Launch film', path: [LUBRICANTS_PLAN], binEntryId: '' },
    ...overrides,
  };
}

/** An empty page of a list. */
export const EMPTY_PAGE = { items: [], total: 0 };

/** Every request of Home, answered with a dashboard that has nothing in it yet. */
export const HOME_ANSWERS: Record<string, MockAnswer> = {
  'GET /api/home/summary': {
    body: summaryOf({
      plans: { count: 0, published: 0, draft: 0 },
      activitiesOwned: { count: 0, declined: 0, submitted: 0 },
      plannedCost: {
        planCount: 0,
        plannedCost: usd('0.00'),
        planBudget: null,
        budgetUsedPercent: null,
        overBudget: false,
        plansWithoutBudget: 0,
      },
    }),
  },
  'GET /api/home/plans': { body: EMPTY_PAGE },
  'GET /api/home/attention': { body: EMPTY_PAGE },
  'GET /api/home/waiting': { body: EMPTY_PAGE },
  'GET /api/home/my-activities': { body: { groups: [], total: 0 } },
  'GET /api/home/recent-changes': { body: EMPTY_PAGE },
  'GET /api/currencies': { body: { items: CURRENCIES, total: CURRENCIES.length } },
  'GET /api/teams': { body: EMPTY_PAGE },
};
