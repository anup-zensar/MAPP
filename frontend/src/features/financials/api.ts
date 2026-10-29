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

/** An import, a link or an unlink changes every figure on these pages, so it invalidates them. */
const FIGURE_KEYS = [FINANCIALS_KEY, ['all-work'], ['home']];

/** The previous page stays on screen while the next one loads, so the table does not jump. */
export function useFinancialTransactions(filters: FinancialFilters) {
  const query = toQuery(filters);
  return useQuery({
    queryKey: [...TRANSACTIONS_KEY, query],
    queryFn: () => call(api.GET('/api/v1/financials/transactions', { params: { query } })),
    placeholderData: keepPreviousData,
  });
}

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

export function useFinancialImports(enabled = true) {
  return useQuery({
    queryKey: IMPORTS_KEY,
    queryFn: () => call(api.GET('/api/v1/financials/imports')),
    enabled,
  });
}

export function useCheckImport() {
  return useMutation({
    mutationFn: (body: FinancialImportRequest) =>
      call(api.POST('/api/v1/financials/imports/check', { body })),
  });
}

/** Imports a file; every figure is asked again afterwards. */
export function useImportExtract() {
  const refresh = useRefreshFigures();
  return useMutation({
    mutationFn: (body: FinancialImportRequest) =>
      call(api.POST('/api/v1/financials/imports', { body })),
    onSettled: refresh,
  });
}

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

/** Undoes a manual link; the transaction is matched again by the rules. */
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
