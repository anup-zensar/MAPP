/**
 * The GSAP money of an activity or a line item and its words; the record view is one query
 * per record and display currency, shared by the card and the columns, so the page asks once.
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

export function useFinancialRecord(recordId: string, enabled = true) {
  const { displayCurrency } = useFormatters();
  return useQuery({
    queryKey: ['financials', 'record', recordId, displayCurrency],
    queryFn: () =>
      call(api.GET('/api/v1/financials/records/{recordId}', { params: { path: { recordId } } })),
    enabled,
  });
}

export const TRANSACTIONS_PAGE_SIZE = 50;

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

export function financialsHref(recordId: string): string {
  return `/financials?recordId=${encodeURIComponent(recordId)}`;
}

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

export function accruedTip(month: string | null): string {
  return month
    ? `Money owed for work received but not yet invoiced, as GSAP accrued it for ${month}.`
    : 'Money owed for work received but not yet invoiced, as GSAP accrued it.';
}

export function meterText(percent: number): string {
  return `${percent}% of planned cost committed`;
}

export const NO_PLANNED_COST = 'No planned cost yet';

export function aboveByText(amount: string): string {
  return `Committed above planned cost by ${amount}`;
}

export function aboveOnText(count: number): string {
  return `Committed above planned cost on ${plural(count, 'line item')}`;
}

export function emptyText(level: FinancialLevel): string {
  return `No purchase orders or invoices yet — they appear here once GSAP sends them for this ${level}'s line items.`;
}

export const NO_DATA = 'No GSAP data loaded yet.';
export const NO_ACCESS = 'You don’t have access to financial data — ask an administrator.';

export function sourceText(source: FinancialSource, dateTime: (at: string) => string): string {
  if (!source.dataAsOf) {
    return NO_DATA;
  }
  const feed = source.feedName ?? 'GSAP';
  const file = source.fileName ? ` '${source.fileName}'` : '';
  return `GSAP data as of ${dateTime(source.dataAsOf)} · ${feed}${file}`;
}

export function viewAllText(count: number): string {
  return `View all ${plural(count, 'transaction')} in Financials`;
}

export const TYPE_WORD: Record<FinancialTransactionType, string> = {
  purchaseOrder: 'Purchase order',
  invoice: 'Invoice',
  payment: 'Payment',
  accrual: 'Accrual',
};

export function linkedTransactionsText(count: number): string | null {
  if (count <= 0) {
    return null;
  }
  return count === 1
    ? '1 GSAP transaction is linked to it. It stays in SP&A One and leaves the figures until it is restored.'
    : `${plural(count, 'GSAP transaction')} are linked to it. They stay in SP&A One and leave the figures until it is restored.`;
}

export const LINKED_TRANSACTIONS_UNKNOWN =
  'Any GSAP transactions linked to it stay in SP&A One and leave the figures until it is restored.';

// A delete must not happen before the linked transactions are known, or the warning would be skipped.
export function useLinkedTransactionsNotice(recordId: string, enabled: boolean) {
  const financials = useFinancialRecord(recordId, enabled);
  const text = financials.isError
    ? LINKED_TRANSACTIONS_UNKNOWN
    : linkedTransactionsText(financials.data?.figures.transactionCount ?? 0);
  return { text, checking: enabled && financials.isPending };
}
