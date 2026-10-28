import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { QueryKey } from '@tanstack/react-query';
import { api, call } from '@/api/client';
import type { components } from '@/api/generated/schema';

export type SwbsRun = components['schemas']['SwbsRun'];
export type SwbsRunPage = components['schemas']['SwbsRunPage'];
export type SwbsRunCode = components['schemas']['SwbsRunCode'];

export const RUNS_SHOWN = 5;
const RUNS_KEY = ['simulated-gsap', 'swbs-runs'];

/** A run changes line items everywhere they show: their pages, sections, All work, Financials, Home, the log. */
const RUN_CHANGES: QueryKey[] = [
  ['simulated-gsap'],
  ['line-items'],
  ['all-work'],
  ['financials'],
  ['home'],
  ['activity-log'],
];

export function useSwbsRuns() {
  return useQuery({
    queryKey: RUNS_KEY,
    queryFn: () =>
      call(
        api.GET('/api/v1/simulated-gsap/swbs-runs', { params: { query: { limit: RUNS_SHOWN } } }),
      ),
  });
}

/** Sends every ready line item, or only lineItemIds. */
export function useSendSwbsRequests() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (lineItemIds?: string[]) =>
      call(
        api.POST('/api/v1/simulated-gsap/swbs-runs', { body: lineItemIds ? { lineItemIds } : {} }),
      ),
    onSettled: () =>
      Promise.all(RUN_CHANGES.map((queryKey) => queryClient.invalidateQueries({ queryKey }))),
  });
}
