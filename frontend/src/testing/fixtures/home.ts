/**
 * Answers of the Home endpoints for tests (feature 009). `HOME_ANSWERS` answers every request Home
 * makes with an empty dashboard, for tests of other parts of the app that land on Home.
 */
import type { HttpHandler } from 'msw';
import type { components } from '@/api/generated/schema';
import type {
  ActivityLogEntry,
  FinancialOverview,
  HomeAttentionItem,
  HomeAttentionReason,
  HomePlanRow,
  HomeSummary,
  Money,
  MyActivityGroup,
  MyActivityRow,
  WaitingItem,
} from '@/features/home';
import { reply } from '../replies';

type DisplayCurrencyOption = components['schemas']['DisplayCurrencyOption'];

/** An amount in US dollars ("1200.00" → $1,200.00). */
export function usd(amount: string): Money {
  return { amount, currency: 'USD' };
}

/** Priya's summary with the seed (plan "Demo data"), in US dollars. */
export function summaryOf(overrides: Partial<HomeSummary> = {}): HomeSummary {
  return {
    scope: 'myTeams',
    displayCurrency: 'USD',
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

/**
 * GET /api/v1/me/display-currencies for a person who shows USD: the active currencies, by code (003
 * FR-023). A person whose display currency was set inactive also gets theirs, marked inactive.
 */
export const DISPLAY_CURRENCIES: DisplayCurrencyOption[] = [
  { code: 'EUR', name: 'Euro', active: true },
  { code: 'GBP', name: 'Pound sterling', active: true },
  { code: 'USD', name: 'US dollar', active: true },
];

/** A row of Plans at a glance: FY2026 Lubricants Europe with the seed's figures. */
export function planRowOf(overrides: Partial<HomePlanRow> = {}): HomePlanRow {
  return {
    id: 'P-1001',
    name: 'FY2026 Lubricants Europe',
    year: 2026,
    strategicAop: false,
    status: 'published',
    teams: [{ id: 'T-1001', name: 'Lubricants Europe', colour: 'blue', active: true, mine: true }],
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

/**
 * Priya's Financials card with the seed (015 US6 scenario 1): Committed 35,652.17 of 460,000.00
 * planned (8 %), Invoiced 33,152.17, Paid 30,652.17, Accrued (Sep 2026) 2,500.00; one activity
 * committed above planned cost and one transaction not linked.
 */
export function overviewOf(overrides: Partial<FinancialOverview> = {}): FinancialOverview {
  return {
    year: 2026,
    scope: 'myTeams',
    figures: {
      plannedCost: usd('460000.00'),
      committed: usd('35652.17'),
      invoiced: usd('33152.17'),
      paid: usd('30652.17'),
      accrued: usd('2500.00'),
      notYetCommitted: usd('424347.83'),
      committedPercent: 8,
      committedAbovePlanned: false,
      lineItemsAbovePlanned: 1,
      firstLineItemAbovePlanned: null,
      transactionCount: 9,
    },
    activitiesAbovePlanned: 1,
    notLinked: 1,
    source: {
      dataAsOf: '2026-10-19T02:00:00Z',
      feedName: 'Demo data (simulated GSAP)',
      fileName: null,
      accrualMonth: '2026-09',
    },
    ...overrides,
  };
}

/** A Financials card without any transaction for the scope (015 US6 scenario 3). */
export function emptyOverview(): FinancialOverview {
  const overview = overviewOf({ activitiesAbovePlanned: 0, notLinked: 0 });
  return {
    ...overview,
    figures: {
      ...overview.figures,
      committed: usd('0.00'),
      invoiced: usd('0.00'),
      paid: usd('0.00'),
      accrued: usd('0.00'),
      committedPercent: 0,
      lineItemsAbovePlanned: 0,
      transactionCount: 0,
    },
  };
}

/** The summary of a person who has nothing yet. */
export const EMPTY_SUMMARY = summaryOf({
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
});

/** Every request of Home, answered with a dashboard that has nothing in it yet. */
export const HOME_ANSWERS: HttpHandler[] = [
  reply('GET /api/v1/home/summary', {
    body: EMPTY_SUMMARY,
  }),
  reply('GET /api/v1/home/plans', { body: EMPTY_PAGE }),
  reply('GET /api/v1/home/attention', { body: EMPTY_PAGE }),
  reply('GET /api/v1/home/waiting', { body: EMPTY_PAGE }),
  reply('GET /api/v1/home/my-activities', { body: { groups: [], total: 0 } }),
  reply('GET /api/v1/home/recent-changes', { body: EMPTY_PAGE }),
  reply('GET /api/v1/financials/overview', { body: emptyOverview() }),
  reply('GET /api/v1/me/display-currencies', {
    body: { items: DISPLAY_CURRENCIES, total: DISPLAY_CURRENCIES.length },
  }),
];
