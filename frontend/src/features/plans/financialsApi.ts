/** What a plan's, campaign's or initiative's Financials section asks the API (015). */
import { useQuery } from '@tanstack/react-query';
import { api, call } from '@/api/client';
import { ApiError } from '@/api/errors';

/** The record's figures, under the key 015 refreshes after an import or a link (015 plan). */
export function useRecordFinancials(recordId: string, displayCurrency: string) {
  return useQuery({
    queryKey: ['financials', 'record', recordId, displayCurrency],
    queryFn: () =>
      call(api.GET('/api/v1/financials/records/{recordId}', { params: { path: { recordId } } })),
    retry: (count, error) => !(error instanceof ApiError && error.kind === 'notFound') && count < 2,
  });
}

/** "View all n transactions in Financials": the Financials page filtered to the record (015). */
export function financialsHref(recordId: string): string {
  return `/financials?${new URLSearchParams({ record: recordId })}`;
}
