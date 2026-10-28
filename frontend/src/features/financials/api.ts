/**
 * What the Financials page asks the API (015 plan, "Query keys and invalidation"): one page of
 * transactions for the filters in the address, one transaction, the imports, the check and the
 * import of a simulated GSAP extract, the sample file, and the manual link and unlink.
 *
 * | Key                                        | Invalidated by        |
 * | ['financials', 'transactions', params]     | import, link, unlink  |
 * | ['financials', 'transaction', id]          | link, unlink          |
 * | ['financials', 'imports']                  | import                |
 *
 * An import, a link or an unlink also changes every figure on the record pages, Home and All work,
 * so those mutations invalidate ['financials'], ['all-work'] and ['home'].
 */
import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api, call } from '@/api/client';
import type { components } from '@/api/generated/schema';
import { downloadFromApi } from '@/api/download';
import type { FinancialFilters } from './financials-links';
import { toQuery } from './financials-links';

export type FinancialTransactionPage = components['schemas']['FinancialTransactionPage'];
export type FinancialTransactionRow = components['schemas']['FinancialTransactionRow'];
export type FinancialTransaction = components['schemas']['FinancialTransaction'];
export type FinancialTransactionType = components['schemas']['FinancialTransactionType'];
export type FinancialMatchedBy = components['schemas']['FinancialMatchedBy'];
export type FinancialLink = components['schemas']['FinancialLink'];
export type FinancialRecordRef = components['schemas']['FinancialRecordRef'];
export type FinancialSource = components['schemas']['FinancialSource'];
export type FinancialTotals = components['schemas']['FinancialTotals'];
export type FinancialTypeCounts = components['schemas']['FinancialTypeCounts'];
export type FinancialImport = components['schemas']['FinancialImport'];
export type FinancialImportCounts = components['schemas']['FinancialImportCounts'];
export type FinancialImportCheck = components['schemas']['FinancialImportCheck'];
export type FinancialImportResult = components['schemas']['FinancialImportResult'];
export type FinancialImportRequest = components['schemas']['FinancialImportRequest'];
export type FinancialImportProblem = components['schemas']['FinancialImportProblem'];
export type FinancialImportNotLinkedRow = components['schemas']['FinancialImportNotLinkedRow'];
export type LineItemPreview = components['schemas']['LineItemDetails'];

export const FINANCIALS_KEY = ['financials'];
const TRANSACTIONS_KEY = [...FINANCIALS_KEY, 'transactions'];
const TRANSACTION_KEY = [...FINANCIALS_KEY, 'transaction'];
export const IMPORTS_KEY = [...FINANCIALS_KEY, 'imports'];

/** The keys of the other places that show financial figures (All work 010, Home 009). */
const FIGURE_KEYS = [FINANCIALS_KEY, ['all-work'], ['home']];

/**
 * One page of transactions for the filters, with the tab counts, the totals, the choices and the
 * source line. The previous page stays on screen while the next one loads, so the table does not
 * jump while people type or page.
 */
export function useFinancialTransactions(filters: FinancialFilters) {
  const query = toQuery(filters);
  return useQuery({
    queryKey: [...TRANSACTIONS_KEY, query],
    queryFn: () => call(api.GET('/api/v1/financials/transactions', { params: { query } })),
    placeholderData: keepPreviousData,
  });
}

/** One transaction with every field, its link and its import (the details panel, FR-022). */
export function useFinancialTransaction(transactionId: string) {
  return useQuery({
    queryKey: [...TRANSACTION_KEY, transactionId],
    queryFn: () =>
      call(
        api.GET('/api/v1/financials/transactions/{transactionId}', {
          params: { path: { transactionId } },
        }),
      ),
  });
}

/** Every import, newest first (Import history, FR-008). */
export function useFinancialImports(enabled = true) {
  return useQuery({
    queryKey: IMPORTS_KEY,
    queryFn: () => call(api.GET('/api/v1/financials/imports')),
    enabled,
  });
}

/** Checks a file without saving anything (FR-004, step 1). */
export function useCheckImport() {
  return useMutation({
    mutationFn: (body: FinancialImportRequest) =>
      call(api.POST('/api/v1/financials/imports/check', { body })),
  });
}

/** Imports a file (FR-004, step 2); every figure is asked again afterwards. */
export function useImportExtract() {
  const refresh = useRefreshFigures();
  return useMutation({
    mutationFn: (body: FinancialImportRequest) =>
      call(api.POST('/api/v1/financials/imports', { body })),
    onSettled: refresh,
  });
}

/** Links a not-linked transaction to a line item by hand (FR-026). */
export function useLinkTransaction() {
  const refresh = useRefreshFigures();
  return useMutation({
    mutationFn: ({
      id,
      lineItemId,
      version,
    }: {
      id: string;
      lineItemId: string;
      version: number;
    }) =>
      call(
        api.POST('/api/v1/financials/transactions/{transactionId}/link', {
          params: { path: { transactionId: id } },
          body: { lineItemId, version },
        }),
      ),
    onSettled: refresh,
  });
}

/** Undoes a manual link (FR-026); the transaction is matched again by the rules. */
export function useUnlinkTransaction() {
  const refresh = useRefreshFigures();
  return useMutation({
    mutationFn: ({ id, version }: { id: string; version: number }) =>
      call(
        api.POST('/api/v1/financials/transactions/{transactionId}/unlink', {
          params: { path: { transactionId: id } },
          body: { version },
        }),
      ),
    onSettled: refresh,
  });
}

/** The line item a person typed, for the link dialog's preview; nothing is asked for "". */
export function useLineItemPreview(lineItemId: string) {
  return useQuery({
    queryKey: [...FINANCIALS_KEY, 'line-item-preview', lineItemId],
    queryFn: (): Promise<LineItemPreview> =>
      call(api.GET('/api/v1/line-items/{lineItemId}', { params: { path: { lineItemId } } })),
    enabled: lineItemId !== '',
    retry: false,
  });
}

/** Saves the sample file "mapp-gsap-extract-sample.csv" (FR-009). */
export function downloadSampleFile(): Promise<string> {
  return downloadFromApi(
    api.GET('/api/v1/financials/imports/template', { parseAs: 'blob' }),
    'spa-one-gsap-extract-sample.csv',
  );
}

function useRefreshFigures() {
  const queryClient = useQueryClient();
  return () =>
    Promise.all(FIGURE_KEYS.map((queryKey) => queryClient.invalidateQueries({ queryKey })));
}
