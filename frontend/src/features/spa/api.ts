import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { api, call } from '@/api/client';
import { ApiError } from '@/api/errors';
import type { components } from '@/api/generated/schema';

export type SpaView = components['schemas']['SpaView'];
export type SpaActivityRow = components['schemas']['SpaActivityRow'];
export type SpaContribution = components['schemas']['SpaContribution'];
export type SpaPeriod = components['schemas']['SpaPeriod'];
export type SpaDimensionRow = components['schemas']['SpaDimensionRow'];
export type Money = components['schemas']['Money'];

export interface SpaQuery {
  planId: string;
  campaign: string | undefined;
  dimension: string | undefined;
}

export const SPA_KEYS = {
  view: (query: SpaQuery, displayCurrency: string) => ['spa', query, displayCurrency] as const,
};

/** The SP&A view of a plan or campaign; the previous answer stays while another scope loads. */
export function useSpaView(query: SpaQuery, displayCurrency: string) {
  return useQuery({
    queryKey: SPA_KEYS.view(query, displayCurrency),
    queryFn: ({ signal }) =>
      call(
        api.GET('/api/v1/plans/{planId}/spa', {
          params: {
            path: { planId: query.planId },
            query: {
              ...(query.campaign && { campaignId: query.campaign }),
              ...(query.dimension && { dimension: query.dimension }),
            },
          },
          signal,
        }),
      ),
    placeholderData: keepPreviousData,
    retry: (count, error) => !isFinal(error) && count < 2,
  });
}

/** A refusal or a missing plan does not change on a second try. */
function isFinal(error: Error): boolean {
  return error instanceof ApiError && ['notFound', 'forbidden', 'validation'].includes(error.kind);
}
