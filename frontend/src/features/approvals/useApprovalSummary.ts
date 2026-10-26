import { useQuery } from '@tanstack/react-query';
import { api, call } from '@/api/client';
import { useMonthRange } from './useMonthRange';

/** How often the counts are asked again: within FR-036's minute without a push channel. */
const POLL_MS = 30_000;

/**
 * The four counts of the approvals page's summary cards (FR-028) — also the sidebar's "waiting for
 * you" badge (FR-027), which shares this query. Asked again every 30 seconds.
 */
export function useApprovalSummary() {
  const { monthStart, monthEnd } = useMonthRange();
  return useQuery({
    queryKey: ['approvals-summary', monthStart],
    queryFn: () =>
      call(api.GET('/api/v1/approvals/summary', { params: { query: { monthStart, monthEnd } } })),
    refetchInterval: POLL_MS,
  });
}
