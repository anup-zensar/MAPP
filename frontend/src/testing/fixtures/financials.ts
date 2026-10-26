/**
 * The demo data's answers for the Financials tests (015 data-model §6): the 11 transactions of
 * FIM-1001, in USD display with the seeded rates EUR 0.92 and GBP 0.79.
 */
import type {
  FinancialImport,
  FinancialImportCheck,
  FinancialRecordRef,
  FinancialTransaction,
  FinancialTransactionPage,
  FinancialTransactionRow,
  LineItemPreview,
} from '@/features/financials/financialsApi';

const PLAN: FinancialRecordRef = { type: 'plan', id: 'P-1001', name: 'FY2026 Lubricants Europe' };
const HELIX: FinancialRecordRef = { type: 'campaign', id: 'C-1001', name: 'Helix launch' };
const DIGITAL: FinancialRecordRef = { type: 'initiative', id: 'I-1001', name: 'Digital & social' };
const PRINT: FinancialRecordRef = { type: 'initiative', id: 'I-1002', name: 'Print & outdoor' };
const SOCIAL: FinancialRecordRef = { type: 'activity', id: 'A-1002', name: 'Social always-on' };
const BILLBOARD: FinancialRecordRef = {
  type: 'activity',
  id: 'A-1003',
  name: 'Billboard winter wave',
};

const PATHS: Record<string, FinancialRecordRef[]> = {
  'L-1003': [PLAN, HELIX, DIGITAL, SOCIAL, { type: 'lineItem', id: 'L-1003', name: 'Venue hire' }],
  'L-1004': [
    PLAN,
    HELIX,
    DIGITAL,
    SOCIAL,
    { type: 'lineItem', id: 'L-1004', name: 'Speaker fees' },
  ],
  'L-1006': [
    PLAN,
    HELIX,
    PRINT,
    BILLBOARD,
    { type: 'lineItem', id: 'L-1006', name: 'Stand build' },
  ],
};

const usd = (amount: string) => ({ amount, currency: 'USD' });

/** A linked row; pass what differs. */
function row(
  id: string,
  fields: Partial<FinancialTransactionRow> & { lineItemId: string | null },
): FinancialTransactionRow {
  const { lineItemId, ...rest } = fields;
  return {
    id,
    type: 'purchaseOrder',
    name: '',
    documentNumber: '',
    lineNumber: '',
    poNumber: '',
    poLineNumber: '',
    postingDate: '2026-01-01',
    accrualMonth: null,
    amount: usd('0.00'),
    amountDisplay: usd('0.00'),
    currencyMissing: false,
    vendor: null,
    description: '',
    swbsCode: '',
    link: lineItemId
      ? {
          state: 'linked',
          matchedBy: 'poNumber',
          reason: null,
          lineItemId,
          path: PATHS[lineItemId] ?? [],
          manualBy: null,
          manualAt: null,
        }
      : {
          state: 'notLinked',
          matchedBy: null,
          reason: 'No line item for HX key HX1099 — check the tracking key in GSAP.',
          lineItemId: null,
          path: [],
          manualBy: null,
          manualAt: null,
        },
    ...rest,
  };
}

const NORTHLIGHT = { code: 'V-100101', name: 'Northlight Events GmbH' };
const KEYNOTE = { code: 'V-100102', name: 'Keynote Speakers Ltd' };
const FRAMES = { code: 'V-100103', name: 'Outdoor Frames Ltd' };
const BRIGHTSIDE = { code: 'V-100104', name: 'Brightside Media BV' };

export const FT_1001 = row('FT-1001', {
  name: 'PO 4500001001 / 10',
  documentNumber: '4500001001',
  lineNumber: '10',
  postingDate: '2026-01-12',
  amount: { amount: '15800.00', currency: 'GBP' },
  amountDisplay: usd('20000.00'),
  vendor: NORTHLIGHT,
  description: 'Venue hire Berlin',
  swbsCode: 'DE03-M26-0003',
  lineItemId: 'L-1003',
});
FT_1001.link.matchedBy = 'trackingKey';

export const FT_1002 = row('FT-1002', {
  name: 'PO 4500001002 / 10',
  documentNumber: '4500001002',
  lineNumber: '10',
  postingDate: '2026-05-04',
  amount: usd('5000.00'),
  amountDisplay: usd('5000.00'),
  vendor: KEYNOTE,
  description: 'Keynote speakers',
  lineItemId: 'L-1004',
});
FT_1002.link.matchedBy = 'trackingKey';

export const FT_1003 = row('FT-1003', {
  name: 'PO 4500001003 / 10',
  documentNumber: '4500001003',
  lineNumber: '10',
  postingDate: '2026-01-08',
  amount: { amount: '9800.00', currency: 'EUR' },
  amountDisplay: usd('10652.17'),
  vendor: FRAMES,
  description: 'Stand build',
  swbsCode: 'GB01-M26-0006',
  lineItemId: 'L-1006',
});
FT_1003.link.matchedBy = 'trackingKey';

export const FT_1004 = row('FT-1004', {
  name: 'PO 4500001004 / 10',
  documentNumber: '4500001004',
  lineNumber: '10',
  postingDate: '2026-09-15',
  amount: usd('12000.00'),
  amountDisplay: usd('12000.00'),
  vendor: BRIGHTSIDE,
  description: 'Retail display production',
  lineItemId: null,
});

export const FT_1005 = row('FT-1005', {
  type: 'invoice',
  name: 'Invoice 5100002001',
  documentNumber: '5100002001',
  poNumber: '4500001001',
  postingDate: '2026-02-20',
  amount: { amount: '15800.00', currency: 'GBP' },
  amountDisplay: usd('20000.00'),
  vendor: NORTHLIGHT,
  description: 'Venue hire Berlin',
  lineItemId: 'L-1003',
});

export const FT_1006 = row('FT-1006', {
  type: 'invoice',
  name: 'Invoice 5100002002',
  documentNumber: '5100002002',
  poNumber: '4500001002',
  postingDate: '2026-09-10',
  amount: usd('2500.00'),
  amountDisplay: usd('2500.00'),
  vendor: KEYNOTE,
  description: 'Keynote speakers, first half',
  lineItemId: 'L-1004',
});

export const FT_1007 = row('FT-1007', {
  type: 'invoice',
  name: 'Invoice 5100002003',
  documentNumber: '5100002003',
  poNumber: '4500001003',
  postingDate: '2026-03-02',
  amount: { amount: '9800.00', currency: 'EUR' },
  amountDisplay: usd('10652.17'),
  vendor: FRAMES,
  description: 'Stand build',
  swbsCode: 'GB01-M26-0006',
  lineItemId: 'L-1006',
});

export const FT_1008 = row('FT-1008', {
  type: 'payment',
  name: 'Payment 1500003001',
  documentNumber: '1500003001',
  poNumber: '4500001001',
  postingDate: '2026-03-20',
  amount: { amount: '15800.00', currency: 'GBP' },
  amountDisplay: usd('20000.00'),
  vendor: NORTHLIGHT,
  description: 'Venue hire Berlin',
  lineItemId: 'L-1003',
});

export const FT_1009 = row('FT-1009', {
  type: 'payment',
  name: 'Payment 1500003002',
  documentNumber: '1500003002',
  postingDate: '2026-04-01',
  amount: { amount: '9800.00', currency: 'EUR' },
  amountDisplay: usd('10652.17'),
  vendor: FRAMES,
  description: 'Stand build',
  swbsCode: 'GB01-M26-0006',
  lineItemId: 'L-1006',
});
FT_1009.link.matchedBy = 'swbsCode';

export const FT_1010 = row('FT-1010', {
  type: 'accrual',
  name: 'Accrual 9100004001',
  documentNumber: '9100004001',
  poNumber: '4500001002',
  postingDate: '2026-08-31',
  accrualMonth: '2026-08',
  amount: usd('5000.00'),
  amountDisplay: usd('5000.00'),
  vendor: KEYNOTE,
  description: 'Keynote speakers accrual',
  lineItemId: 'L-1004',
});

export const FT_1011 = row('FT-1011', {
  type: 'accrual',
  name: 'Accrual 9100004002',
  documentNumber: '9100004002',
  poNumber: '4500001002',
  postingDate: '2026-09-30',
  accrualMonth: '2026-09',
  amount: usd('2500.00'),
  amountDisplay: usd('2500.00'),
  vendor: KEYNOTE,
  description: 'Keynote speakers accrual',
  lineItemId: 'L-1004',
});

/** The 11 transactions, newest first (the default sort). */
export const ALL_ROWS: FinancialTransactionRow[] = [
  FT_1011,
  FT_1004,
  FT_1006,
  FT_1010,
  FT_1002,
  FT_1009,
  FT_1008,
  FT_1007,
  FT_1005,
  FT_1001,
  FT_1003,
];

const DEMO_SOURCE = {
  dataAsOf: '2026-10-19T02:00:00Z',
  feedName: 'Demo data (simulated GSAP)',
  fileName: null,
  accrualMonth: '2026-09',
};

/** The Financials page for 2026, All (US3 scenario 1); pass what differs. */
export function transactionPage(
  overrides: Partial<FinancialTransactionPage> = {},
): FinancialTransactionPage {
  return {
    items: ALL_ROWS,
    total: 11,
    page: 1,
    pageSize: 50,
    year: 2026,
    counts: { all: 11, purchaseOrder: 4, invoice: 3, payment: 2, accrual: 2, notLinked: 1 },
    totals: {
      committed: usd('47652.17'),
      invoiced: usd('33152.17'),
      paid: usd('30652.17'),
      accrued: usd('2500.00'),
    },
    recordFilter: null,
    choices: {
      years: [2027, 2026],
      currentYear: 2026,
      plans: [
        { id: 'P-1001', name: 'FY2026 Lubricants Europe', year: 2026 },
        { id: 'P-1002', name: 'FY2026 Mobility China', year: 2026 },
      ],
    },
    source: DEMO_SOURCE,
    ...overrides,
  };
}

const SYSTEM = { id: 'system', name: 'MAPP' };

export const DEMO_IMPORT: FinancialImport = {
  id: 'FIM-1001',
  source: 'demo',
  feedName: 'Demo data (simulated GSAP)',
  fileName: '',
  importedAt: '2026-10-19T02:00:00Z',
  importedBy: SYSTEM,
  counts: { rows: 11, new: 11, updated: 0, unchanged: 0, refused: 0, notLinked: 1, relinked: 0 },
};

/** A transaction with the panel's fields, from one of the rows above. */
export function transactionOf(
  source: FinancialTransactionRow,
  overrides: Partial<FinancialTransaction> = {},
): FinancialTransaction {
  const notLinked = source.link.state === 'notLinked';
  return {
    ...source,
    amountUsd: source.amountDisplay,
    trackingKey:
      source.type === 'purchaseOrder'
        ? `HX${source.id === 'FT-1004' ? '1099' : (source.link.lineItemId ?? '').slice(2)}`
        : '',
    sentLineItemId: '',
    import: {
      id: DEMO_IMPORT.id,
      feedName: DEMO_IMPORT.feedName,
      fileName: '',
      importedAt: DEMO_IMPORT.importedAt,
      importedBy: SYSTEM,
    },
    canLink: notLinked,
    canUnlink: false,
    version: 1,
    createdAt: '2026-10-19T02:00:00Z',
    updatedAt: '2026-10-19T02:00:00Z',
    updatedBy: SYSTEM,
    ...overrides,
  };
}

/** FT-1004 after Alex Morgan linked it to L-1007 by hand (US7 scenario 1). */
export function manuallyLinked(): FinancialTransaction {
  return transactionOf(FT_1004, {
    link: {
      state: 'linked',
      matchedBy: 'manual',
      reason: null,
      lineItemId: 'L-1007',
      path: [
        { type: 'plan', id: 'P-1002', name: 'FY2026 Mobility China' },
        { type: 'campaign', id: 'C-1003', name: 'Retail growth' },
        { type: 'initiative', id: 'I-1005', name: 'Station promotions' },
        { type: 'activity', id: 'A-1008', name: 'Grand opening Shanghai' },
        { type: 'lineItem', id: 'L-1007', name: 'Station signage' },
      ],
      manualBy: { id: 'U-1001', name: 'Alex Morgan' },
      manualAt: '2026-10-20T09:00:00Z',
    },
    canLink: false,
    canUnlink: true,
    version: 2,
  });
}

/** What the check of a 4-row file says (US4 scenario 2). */
export function checkResult(overrides: Partial<FinancialImportCheck> = {}): FinancialImportCheck {
  return {
    fileName: 'gsap-2026-10-19.csv',
    counts: { rows: 4, new: 2, updated: 1, unchanged: 1, refused: 0, notLinked: 1, relinked: 0 },
    problems: [],
    problemsTotal: 0,
    notLinked: [
      {
        row: 5,
        name: 'PO 4500001009 / 10',
        reason: 'No line item for HX key HX1099 — check the tracking key in GSAP.',
      },
    ],
    notLinkedTotal: 1,
    unusedColumns: [],
    ...overrides,
  };
}

/** GET /api/v1/line-items/L-1007 for the link dialog's preview. */
export const STATION_SIGNAGE = {
  lineItem: { id: 'L-1007', activityId: 'A-1008', name: 'Station signage' },
  activity: {
    id: 'A-1008',
    name: 'Grand opening Shanghai',
    path: [
      { type: 'plan', id: 'P-1002', name: 'FY2026 Mobility China' },
      { type: 'campaign', id: 'C-1003', name: 'Retail growth' },
      { type: 'initiative', id: 'I-1005', name: 'Station promotions' },
    ],
  },
} as unknown as LineItemPreview;
