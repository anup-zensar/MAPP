/**
 * The GSAP money of an activity or a line item (feature 015, US1): what the activity page's
 * Financials card, the Line items section's Committed and Invoiced columns and the line-item
 * page's Financials section ask the API, and how they word it (015 plan, "User-visible
 * messages"). The record view is one query per record and display currency, shared by the card and
 * the columns (015 plan, "Query keys"), so the page asks once.
 */
import { useQuery } from '@tanstack/react-query';
import { api, call } from '@/api/client';
import type { components } from '@/api/generated/schema';
import { useFormatters } from '@/format/useFormatters';
import { plural } from '@/format/text';

export type FinancialRecordView = components['schemas']['FinancialRecordView'];
export type FinancialFiguresData = components['schemas']['FinancialFigures'];
export type FinancialTransactionRow = components['schemas']['FinancialTransactionRow'];
export type FinancialTransactionPage = components['schemas']['FinancialTransactionPage'];
export type FinancialTransactionType = components['schemas']['FinancialTransactionType'];
export type FinancialSource = components['schemas']['FinancialSource'];

/** A record's figures, breakdown, latest transactions and SWBS codes (GET /api/v1/financials/records/{id}). */
export function useFinancialRecord(recordId: string, enabled = true) {
  const { displayCurrency } = useFormatters();
  return useQuery({
    queryKey: ['financials', 'record', recordId, displayCurrency],
    queryFn: () =>
      call(api.GET('/api/v1/financials/records/{recordId}', { params: { path: { recordId } } })),
    enabled,
  });
}

/** Transactions shown per page on the line-item page (FR-017). */
export const TRANSACTIONS_PAGE_SIZE = 50;

/** One page of a record's transactions, newest first (GET /api/v1/financials/transactions). */
export function useRecordTransactions(recordId: string, page: number) {
  const { displayCurrency } = useFormatters();
  return useQuery({
    queryKey: [
      'financials',
      'transactions',
      { recordId, page, pageSize: TRANSACTIONS_PAGE_SIZE },
      displayCurrency,
    ],
    queryFn: () =>
      call(
        api.GET('/api/v1/financials/transactions', {
          params: { query: { recordId, page, pageSize: TRANSACTIONS_PAGE_SIZE } },
        }),
      ),
  });
}

/** The Financials page filtered to one record (feature 015, FR-020). */
export function financialsHref(recordId: string): string {
  return `/financials?recordId=${encodeURIComponent(recordId)}`;
}

// Texts (015 plan, "User-visible messages")
export const FINANCIALS_TITLE = 'Financials';
export type FinancialLevel = 'activity' | 'line item';

export function cardTip(level: FinancialLevel): string {
  return `Purchase orders, invoices, payments and accruals from GSAP for this ${level}'s line items.`;
}

export const FIGURE_LABEL = {
  committed: 'Committed (POs)',
  invoiced: 'Invoiced',
  paid: 'Paid',
  accrued: 'Accrued',
  notYetCommitted: 'Not yet committed',
} as const;

export const FIGURE_TIP = {
  committed: 'The value of purchase orders GSAP has issued for these line items.',
  invoiced: 'Supplier invoices GSAP has received against them, less credit notes.',
  paid: 'Payments GSAP has released — Shell’s actuals.',
  notYetCommitted: 'Planned cost minus committed.',
} as const;

/** "Money owed for work received but not yet invoiced, as GSAP accrued it for Sep 2026." */
export function accruedTip(month: string | null): string {
  return month
    ? `Money owed for work received but not yet invoiced, as GSAP accrued it for ${month}.`
    : 'Money owed for work received but not yet invoiced, as GSAP accrued it.';
}

/** "38% of planned cost committed" (design system §4.4: no space before %). */
export function meterText(percent: number): string {
  return `${percent}% of planned cost committed`;
}

export const NO_PLANNED_COST = 'No planned cost yet';

/** "Committed above planned cost by 652.17 USD". */
export function aboveByText(amount: string): string {
  return `Committed above planned cost by ${amount}`;
}

/** "Committed above planned cost on 1 line item". */
export function aboveOnText(count: number): string {
  return `Committed above planned cost on ${plural(count, 'line item')}`;
}

export function emptyText(level: FinancialLevel): string {
  return `No purchase orders or invoices yet — they appear here once GSAP sends them for this ${level}'s line items.`;
}

export const NO_DATA = 'No GSAP data loaded yet.';
export const NO_ACCESS = 'You don’t have access to financial data — ask an administrator.';

/** "GSAP data as of 19 Oct 2026, 09:00 · Demo data (simulated GSAP)". */
export function sourceText(source: FinancialSource, dateTime: (at: string) => string): string {
  if (!source.dataAsOf) {
    return NO_DATA;
  }
  const feed = source.feedName ?? 'GSAP';
  const file = source.fileName ? ` '${source.fileName}'` : '';
  return `GSAP data as of ${dateTime(source.dataAsOf)} · ${feed}${file}`;
}

/** "View all 7 transactions in Financials". */
export function viewAllText(count: number): string {
  return `View all ${plural(count, 'transaction')} in Financials`;
}

export const TYPE_WORD: Record<FinancialTransactionType, string> = {
  purchaseOrder: 'Purchase order',
  invoice: 'Invoice',
  payment: 'Payment',
  accrual: 'Accrual',
};

/** "SWBS code: DE03-M26-0003 (from GSAP data)", or "Filled by GSAP later" without one. */
export function swbsCodesText(codes: string[]): string {
  return codes.length > 0 ? `${codes.join(', ')} (from GSAP data)` : 'Filled by GSAP later';
}

/** The delete dialogs' sentence (FR-029); null when nothing is linked. */
export function linkedTransactionsText(count: number): string | null {
  if (count <= 0) {
    return null;
  }
  return count === 1
    ? '1 GSAP transaction is linked to it. It stays in MAPP and leaves the figures until it is restored.'
    : `${plural(count, 'GSAP transaction')} are linked to it. They stay in MAPP and leave the figures until it is restored.`;
}
