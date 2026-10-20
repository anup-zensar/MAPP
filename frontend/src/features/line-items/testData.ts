/**
 * Contract-shaped answers for the line-item tests (007 tasks T031): the activity "Launch film"
 * (planned cost 120,000.00 USD, 1 Mar – 31 May 2026) with "Agency fee" (50,000.00 USD) and "Film
 * production" (46,000.00 EUR = 50,000.00 USD), its form options, the line-item Shell attributes
 * (Spend type → GL code) and the answers every form asks for.
 */
import type { components } from '@/api/schema';
import { attribute, option } from '@/test/attributeForms';
import type { MockHandler } from '@/test/mockApi';

type LineItem = components['schemas']['LineItem'];
type LineItemActivity = components['schemas']['LineItemActivity'];
type LineItemSection = components['schemas']['LineItemSection'];
type LineItemFormOptions = components['schemas']['LineItemFormOptions'];
type LineItemDetails = components['schemas']['LineItemDetails'];
type LineItemCurrency = components['schemas']['LineItemCurrency'];
type FinanceBudgetItemRef = components['schemas']['FinanceBudgetItemRef'];
type AttributeForm = components['schemas']['AttributeForm'];
type Person = components['schemas']['Person'];
type ErrorBody = components['schemas']['Error'];

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

/** The form options of "Launch film": 20,000.00 USD unassigned (US2 scenario 1). */
export function formOptions(overrides: Partial<LineItemFormOptions> = {}): LineItemFormOptions {
  return {
    activity: activity(),
    balance: launchFilmSection().balance,
    defaults: {
      name: 'Launch film – line item',
      owner: PRIYA_REF,
      currency: 'USD',
      plannedCost: usd('20000.00'),
      plannedCostSource: 'unassignedBalance',
      spreadMethod: 'evenly',
      financeBudgetYear: 2026,
      financeBudgetItemId: null,
      financeBudgetItemPreselected: false,
      requestSwbs: true,
    },
    currencies: [USD, EUR, GBP],
    financeBudgetYears: [2026, 2027],
    financeBudgetItems: [DIGITAL, EVENTS, MOBILITY_RETAIL, DIGITAL_2027],
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

/** The line-item Shell attributes: Spend type → GL code (R11), both needed to submit. */
export function lineItemForm(): AttributeForm {
  return {
    level: 'lineItem',
    businessUnitAttributeId: 'SA-BU',
    businessUnitOptions: [option('SO-BU-LUB', 'Lubricants'), option('SO-BU-MOB', 'Mobility')],
    sections: [
      {
        name: 'Finance coding',
        conditional: false,
        attributes: [
          attribute({
            id: 'SA-ST',
            label: 'Spend type',
            marking: 'neededToSubmit',
            options: [option('SO-ST-ATL', 'ATL – Media'), option('SO-ST-BTL', 'BTL – Event')],
          }),
          attribute({
            id: 'SA-GL',
            label: 'GL code',
            marking: 'neededToSubmit',
            parentAttributeId: 'SA-ST',
            options: [
              option('SO-GL-DIG', '7250080 – Advertising digital', ['SO-ST-ATL']),
              option('SO-GL-PRT', '7250090 – Advertising print', ['SO-ST-ATL']),
              option('SO-GL-EVT', '7260010 – Event costs', ['SO-ST-BTL']),
            ],
          }),
        ],
      },
    ],
  };
}

const PRIYA: Person = {
  ...PRIYA_REF,
  email: 'priya.raman@northstar.demo',
  teams: [{ id: 'T-1001', name: 'Lubricants Europe', colour: 'blue', active: true }],
  version: 1,
  createdAt: '2026-01-01T00:00:00Z',
  createdBy: ACTOR,
  updatedAt: '2026-01-01T00:00:00Z',
  updatedBy: ACTOR,
};

/**
 * What every line-item form asks besides its own data: people and attributes. The plan's Business
 * unit options come with the activity (D-212), so no plan is asked for.
 */
export function formAnswers(): Record<string, MockHandler> {
  return {
    'GET /api/people': { body: { items: [PRIYA], total: 1 } },
    'GET /api/shell-attribute-forms/lineItem': { body: lineItemForm() },
  };
}

/** A 400 answer with these field problems. */
export function validationError(fields: { field: string; message: string }[]): {
  status: number;
  body: ErrorBody;
} {
  return {
    status: 400,
    body: { code: 'validation_failed', message: 'Check the highlighted fields.', fields },
  };
}

/** The 400 that says the person must confirm "This activity will need approval again" (D-189). */
export function restartRefusal() {
  return validationError([
    {
      field: 'restartApproval',
      message:
        'This activity will need approval again. Confirm the change to send it back for approval.',
    },
  ]);
}

/** A 404 with the API's message. */
export function notFound(message: string): { status: number; body: ErrorBody } {
  return { status: 404, body: { code: 'not_found', message, fields: [] } };
}

/** The answer to a delete: the recycle-bin entry for Undo. */
export function deleteResult(name = 'Agency fee', id = 'L-1001') {
  return {
    recycleBinEntry: {
      id: 'BIN-1001',
      recordType: 'lineItem',
      recordTypeLabel: 'line item',
      recordId: id,
      recordName: name,
      path: activity().path,
      parent: { type: 'activity', id: 'A-1001', name: 'Launch film' },
      deletedAt: '2026-03-14T10:30:00Z',
      deletedBy: ACTOR,
      contents: [],
    },
  };
}

/** Today's rates (GET /api/exchange-rates/current), for the ⓘ that names the rates used (016 FR-009). */
export function currentRates() {
  return {
    today: '2026-03-12',
    total: 3,
    items: [
      {
        currency: 'USD',
        rate: '1',
        inverseRate: '1',
        rateDate: null,
        source: null,
        importFileName: null,
      },
      {
        currency: 'EUR',
        rate: '0.92',
        inverseRate: '1.086957',
        rateDate: '2026-03-01',
        source: 'settings' as const,
        importFileName: null,
      },
      {
        currency: 'GBP',
        rate: '0.79',
        inverseRate: '1.265823',
        rateDate: '2026-03-01',
        source: 'gsapImport' as const,
        importFileName: 'rates-2026-03-01.csv',
      },
    ],
  };
}
