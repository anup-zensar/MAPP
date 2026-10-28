import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { api, call } from '@/api/client';
import type { BreakdownQuery } from './all-work-types';

/** The Breakdown of a filter set, kept like the views' answers (null: nothing is asked yet). */
export function useAllWorkBreakdown(query: BreakdownQuery | null) {
  return useQuery({
    queryKey: ['all-work-breakdown', query] as const,
    queryFn: () => call(api.GET('/api/v1/all-work/breakdown', { params: { query: query ?? {} } })),
    enabled: query !== null,
    staleTime: Infinity,
    refetchOnWindowFocus: false,
    placeholderData: keepPreviousData,
  });
}
