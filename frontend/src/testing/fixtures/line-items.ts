/**
 * Contract-shaped answers for the line-item tests (007 tasks T031): the activity "Launch film"
 * (planned cost 120,000.00 USD, 1 Mar – 31 May 2026) with "Agency fee" (50,000.00 USD) and "Film
 * production" (46,000.00 EUR = 50,000.00 USD), their currencies and finance budget items.
 */
import type { components } from '@/api/generated/schema';

type LineItem = components['schemas']['LineItem'];
type LineItemActivity = components['schemas']['LineItemActivity'];
type LineItemSection = components['schemas']['LineItemSection'];
type LineItemDetails = components['schemas']['LineItemDetails'];
type LineItemCurrency = components['schemas']['LineItemCurrency'];
type FinanceBudgetItemRef = components['schemas']['FinanceBudgetItemRef'];

export const usd = (amount: string) => ({ amount, currency: 'USD' });
export const eur = (amount: string) => ({ amount, currency: 'EUR' });

export const PRIYA_REF = {
  id: 'U-1002',
  name: 'Priya Raman',
  initials: 'PR',
  jobTitle: 'Marketer',
  active: true,
};

const ACTOR = { id: 'U-1002', name: 'Priya Raman' };
const DANIEL = { id: 'U-1003', name: 'Daniel Okoro' };

export const USD: LineItemCurrency = {
  code: 'USD',
  name: 'US dollar',
  symbol: '$',
  ratePerUsd: '1',
  active: true,
};
export const EUR: LineItemCurrency = {
  code: 'EUR',
  name: 'Euro',
  symbol: '€',
  ratePerUsd: '0.92',
  active: true,
};
export const GBP: LineItemCurrency = {
  code: 'GBP',
  name: 'Pound sterling',
  symbol: '£',
  ratePerUsd: '0.79',
  active: true,
};
export const INR_INACTIVE: LineItemCurrency = {
  code: 'INR',
  name: 'Indian rupee',
  symbol: '₹',
  ratePerUsd: '83.50',
  active: false,
};

function item(
  id: string,
  name: string,
  code: string,
  budgetName = '2026 Lubricants Budget',
  year = 2026,
  active = true,
): FinanceBudgetItemRef {
  return { id, year, budgetName, name, code, active };
}

export const DIGITAL = item('FBI-1001', 'Europe – Digital', 'LUB-EU-DIG');
export const EVENTS = item('FBI-1002', 'Europe – Events', 'LUB-EU-EVT');
export const BRAND_INACTIVE = item(
  'FBI-1003',
  'Europe – Brand',
  'LUB-EU-BRD',
  '2026 Lubricants Budget',
  2026,
  false,
);
export const MOBILITY_RETAIL = item(
  'FBI-1101',
  'China – Retail',
  'MOB-CN-RET',
  '2026 Mobility Budget',
);
export const DIGITAL_2027 = item(
  'FBI-2001',
  'Europe – Digital',
  'LUB-EU-DIG',
  '2027 Lubricants Budget',
  2027,
);

export function activity(overrides: Partial<LineItemActivity> = {}): LineItemActivity {
  return {
    id: 'A-1001',
    name: 'Launch film',
    status: 'created',
    restartsApproval: false,
    startDate: '2026-03-01',
    endDate: '2026-05-31',
    ongoing: false,
    plannedCost: usd('120000.00'),
    path: [
      { type: 'plan', id: 'P-1001', name: '2026 Lubricants Europe' },
      { type: 'campaign', id: 'C-1001', name: 'Helix Ultra relaunch' },
      { type: 'initiative', id: 'I-1001', name: 'Brand awareness' },
    ],
    planYear: 2026,
    defaultSpreadMethod: 'evenly',
    defaultFinanceBudget: '2026 Lubricants Budget',
    businessUnitOptionIds: [],
    ...overrides,
  };
}

/** An approved activity: line-item changes of cost restart its approval (FR-031). */
export function approvedActivity(): LineItemActivity {
  return activity({ status: 'approved', restartsApproval: true });
}

export function lineItem(overrides: Partial<LineItem> = {}): LineItem {
  const plannedCostUsd = overrides.plannedCostUsd ?? usd('50000.00');
  return {
    id: 'L-1001',
    activityId: 'A-1001',
    name: 'Agency fee',
    owner: PRIYA_REF,
    description: '',
    currency: USD,
    plannedCost: usd('50000.00'),
    plannedCostUsd,
    // The reader shows amounts in USD unless a test says otherwise (D-213).
    plannedCostDisplay: plannedCostUsd,
    spread: {
      method: 'evenly',
      state: 'spread',
      months: [
        spreadMonth('2026-03', usd('16666.66'), usd('16666.66')),
        spreadMonth('2026-04', usd('16666.66'), usd('16666.66')),
        spreadMonth('2026-05', usd('16666.68'), usd('16666.68')),
      ],
      customTotal: null,
      addsUp: true,
    },
    attributes: {},
    contractedAmount: null,
    requestSwbs: true,
    swbsStatus: 'waitingForApproval',
    financeBudgetYear: 2026,
    financeBudgetItem: null,
    missing: [
      { field: 'attributes.SA-ST', label: 'Spend type' },
      { field: 'attributes.SA-GL', label: 'GL code' },
      { field: 'financeBudgetItemId', label: 'Finance budget item' },
    ],
    version: 1,
    createdAt: '2026-03-12T09:00:00Z',
    createdBy: ACTOR,
    updatedAt: '2026-03-14T10:30:00Z',
    updatedBy: DANIEL,
    ...overrides,
  };
}

export function spreadMonth(
  month: string,
  amount: { amount: string; currency: string },
  amountUsd: { amount: string; currency: string },
  outsideActivityDates = false,
) {
  return { month, amount, amountUsd, outsideActivityDates };
}

/** "Film production": 46,000.00 EUR = 50,000.00 USD at 0.92, complete. */
export function filmProduction(): LineItem {
  return lineItem({
    id: 'L-1002',
    name: 'Film production',
    currency: EUR,
    plannedCost: eur('46000.00'),
    plannedCostUsd: usd('50000.00'),
    spread: {
      method: 'evenly',
      state: 'spread',
      months: [
        spreadMonth('2026-03', eur('15333.33'), usd('16666.66')),
        spreadMonth('2026-04', eur('15333.33'), usd('16666.66')),
        spreadMonth('2026-05', eur('15333.34'), usd('16666.68')),
      ],
      customTotal: null,
      addsUp: true,
    },
    requestSwbs: false,
    swbsStatus: 'notRequested',
    financeBudgetItem: DIGITAL,
    missing: [],
    attributes: { 'SA-ST': 'SO-ST-ATL', 'SA-GL': 'SO-GL-DIG' },
  });
}

/** The Launch film section of US1 scenario 1: 20,000.00 USD unassigned, Mar–May. */
export function launchFilmSection(overrides: Partial<LineItemSection> = {}): LineItemSection {
  return {
    activity: activity(),
    items: [lineItem(), filmProduction()],
    balance: {
      plannedCost: usd('120000.00'),
      assigned: usd('100000.00'),
      balance: usd('20000.00'),
      state: 'left',
      roundingHint: false,
    },
    grid: {
      columns: [
        { key: '2026-03', month: '2026-03', outsideActivityDates: false },
        { key: '2026-04', month: '2026-04', outsideActivityDates: false },
        { key: '2026-05', month: '2026-05', outsideActivityDates: false },
      ],
      rows: [
        {
          lineItemId: 'L-1001',
          name: 'Agency fee',
          cells: [usd('16666.66'), usd('16666.66'), usd('16666.68')],
          total: usd('50000.00'),
        },
        {
          lineItemId: 'L-1002',
          name: 'Film production',
          cells: [usd('16666.66'), usd('16666.66'), usd('16666.68')],
          total: usd('50000.00'),
        },
      ],
      totals: [usd('33333.32'), usd('33333.32'), usd('33333.36')],
      total: usd('100000.00'),
    },
    ...overrides,
  };
}

export function details(overrides: Partial<LineItemDetails> = {}): LineItemDetails {
  return {
    lineItem: lineItem(),
    activity: activity(),
    balance: launchFilmSection().balance,
    ...overrides,
  };
}
