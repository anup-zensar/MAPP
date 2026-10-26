/** GSAP's figures of "Launch film" (A-1001) and "Agency fee" (L-1001), feature 015. */
import type { components } from '@/api/generated/schema';
import { nth } from '../dom';
import { usd } from './line-items';

type FinancialRecordView = components['schemas']['FinancialRecordView'];
type FinancialFigures = components['schemas']['FinancialFigures'];
type FinancialTransactionRow = components['schemas']['FinancialTransactionRow'];
type FinancialTransactionPage = components['schemas']['FinancialTransactionPage'];

const SOURCE = {
  dataAsOf: '2026-03-12T09:00:00Z',
  feedName: 'Demo data (simulated GSAP)',
  fileName: null,
  accrualMonth: '2026-03',
};

/** Nothing yet unless given (015 FR-011). */
export function figures(overrides: Partial<FinancialFigures> = {}): FinancialFigures {
  return {
    plannedCost: usd('50000.00'),
    committed: usd('0.00'),
    invoiced: usd('0.00'),
    paid: usd('0.00'),
    accrued: usd('0.00'),
    notYetCommitted: usd('50000.00'),
    committedPercent: 0,
    committedAbovePlanned: false,
    lineItemsAbovePlanned: 0,
    firstLineItemAbovePlanned: null,
    transactionCount: 0,
    ...overrides,
  };
}

/** One GSAP transaction row: the purchase order of "Agency fee" unless given. */
export function transaction(
  overrides: Partial<FinancialTransactionRow> = {},
): FinancialTransactionRow {
  return {
    id: 'FT-1001',
    type: 'purchaseOrder',
    name: 'PO 4500001001 / 10',
    documentNumber: '4500001001',
    lineNumber: '10',
    poNumber: '',
    poLineNumber: '',
    postingDate: '2026-03-05',
    accrualMonth: null,
    amount: usd('30000.00'),
    amountDisplay: usd('30000.00'),
    currencyMissing: false,
    vendor: { code: 'V-100101', name: 'Brightlight Agency' },
    description: '',
    swbsCode: 'DE03-M26-0003',
    link: {
      state: 'linked',
      matchedBy: 'lineItemId',
      reason: null,
      lineItemId: 'L-1001',
      path: [],
      manualBy: null,
      manualAt: null,
    },
    ...overrides,
  };
}

/**
 * "Launch film" (A-1001) as GSAP reports it: Agency fee committed 30,000.00 and invoiced 12,000.00;
 * Film production committed 52,000.00, above its 50,000.00 planned cost (015 FR-013).
 */
export function activityRecord(overrides: Partial<FinancialRecordView> = {}): FinancialRecordView {
  return {
    record: { type: 'activity', id: 'A-1001', name: 'Launch film' },
    planId: 'P-1001',
    path: [],
    figures: figures({
      plannedCost: usd('120000.00'),
      committed: usd('82000.00'),
      invoiced: usd('12000.00'),
      paid: usd('10000.00'),
      accrued: usd('2000.00'),
      notYetCommitted: usd('38000.00'),
      committedPercent: 68,
      lineItemsAbovePlanned: 1,
      firstLineItemAbovePlanned: { type: 'lineItem', id: 'L-1002', name: 'Film production' },
      transactionCount: 3,
    }),
    breakdownLevel: 'lineItem',
    breakdown: [
      {
        record: { type: 'lineItem', id: 'L-1001', name: 'Agency fee' },
        figures: figures({
          committed: usd('30000.00'),
          invoiced: usd('12000.00'),
          paid: usd('10000.00'),
          notYetCommitted: usd('20000.00'),
          committedPercent: 60,
          transactionCount: 2,
        }),
      },
      {
        record: { type: 'lineItem', id: 'L-1002', name: 'Film production' },
        figures: figures({
          committed: usd('52000.00'),
          notYetCommitted: usd('-2000.00'),
          committedPercent: 104,
          committedAbovePlanned: true,
          transactionCount: 1,
        }),
      },
    ],
    latest: [
      transaction({
        id: 'FT-1003',
        type: 'invoice',
        name: 'Invoice 5100002003',
        postingDate: '2026-04-02',
        amount: usd('12000.00'),
        amountDisplay: usd('12000.00'),
      }),
      transaction({
        id: 'FT-1002',
        name: 'PO 4500001002 / 10',
        postingDate: '2026-03-20',
        amount: usd('52000.00'),
        amountDisplay: usd('52000.00'),
      }),
      transaction(),
    ],
    swbsCodes: [],
    source: SOURCE,
    ...overrides,
  };
}

/** "Agency fee" (L-1001) as GSAP reports it: its PO, an invoice and a payment, one SWBS code. */
export function lineItemRecord(overrides: Partial<FinancialRecordView> = {}): FinancialRecordView {
  return {
    ...activityRecord(),
    record: { type: 'lineItem', id: 'L-1001', name: 'Agency fee' },
    figures: nth(activityRecord().breakdown, 0).figures,
    breakdownLevel: null,
    breakdown: [],
    latest: [],
    swbsCodes: ['DE03-M26-0003'],
    ...overrides,
  };
}

export function transactionPage(items: FinancialTransactionRow[]): FinancialTransactionPage {
  return {
    items,
    total: items.length,
    page: 1,
    pageSize: 50,
    year: 2026,
    counts: {
      all: items.length,
      purchaseOrder: 1,
      invoice: 1,
      payment: 0,
      accrual: 0,
      notLinked: 0,
    },
    totals: {
      committed: usd('30000.00'),
      invoiced: usd('12000.00'),
      paid: usd('0.00'),
      accrued: usd('0.00'),
    },
    recordFilter: { type: 'lineItem', id: 'L-1001', name: 'Agency fee' },
    choices: { years: [2026], currentYear: 2026, plans: [] },
    source: SOURCE,
  };
}
