import { useQuery } from '@tanstack/react-query';
import { api, call } from '@/api/client';

/** The record's figures, under the key 015 refreshes after an import or a link (015 plan). */
export function useRecordFinancials(recordId: string, displayCurrency: string) {
  return useQuery({
    queryKey: ['financials', 'record', recordId, displayCurrency],
    queryFn: () =>
      call(api.GET('/api/v1/financials/records/{recordId}', { params: { path: { recordId } } })),
  });
}

export function financialsHref(recordId: string): string {
  return `/financials?${new URLSearchParams({ recordId })}`;
}
