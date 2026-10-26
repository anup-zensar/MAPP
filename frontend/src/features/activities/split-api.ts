/**
 * Split activity (feature 016, US4, FR-026 – FR-031): the dialog's data, the preview of both
 * activities and the split itself. The API works out every amount in the plan currency; the web
 * app adds up the ticked line items only to pre-fill the amount, as feedback (D-061).
 */
import {
  keepPreviousData,
  skipToken,
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query';
import { api, call } from '@/api/client';
import type { components } from '@/api/generated/schema';
import { ACTIVITY_PLANNING_KEYS } from './api';

export type ActivitySplitOptions = components['schemas']['ActivitySplitOptions'];
export type ActivitySplitRequest = components['schemas']['ActivitySplitRequest'];
export type ActivitySplitPreview = components['schemas']['ActivitySplitPreview'];
export type ActivitySplitSide = components['schemas']['ActivitySplitSide'];
export type ActivitySplitResult = components['schemas']['ActivitySplitResult'];
export type SplitLink = components['schemas']['SplitLink'];

/** The dialog's data: the line items with their plan-currency amounts and the initiatives. */
export function useSplitOptions(activityId: string) {
  return useQuery({
    queryKey: ['activity', activityId, 'split-options'],
    queryFn: () =>
      call(
        api.GET('/api/v1/activities/{activityId}/split-options', {
          params: { path: { activityId } },
        }),
      ),
    // Each opening asks afresh: line items and their amounts may have changed.
    gcTime: 0,
  });
}

/** Both activities as the split would leave them; saves nothing. Null while nothing is chosen. */
export function useSplitPreview(activityId: string, request: ActivitySplitRequest | null) {
  return useQuery({
    queryKey: ['activity', activityId, 'split-preview', request],
    queryFn:
      request === null
        ? skipToken
        : () =>
            call(
              api.POST('/api/v1/activities/{activityId}/split-preview', {
                params: { path: { activityId } },
                body: request,
              }),
            ),
    placeholderData: keepPreviousData,
    gcTime: 0,
  });
}

/** Splits the activity, all or nothing, then refreshes everything that shows either activity. */
export function useSplitActivity() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, body }: { id: string; body: ActivitySplitRequest }) =>
      call(
        api.POST('/api/v1/activities/{activityId}/split', {
          params: { path: { activityId: id } },
          body,
        }),
      ),
    onSettled: () =>
      Promise.all(
        [...ACTIVITY_PLANNING_KEYS, ['financials']].map((queryKey) =>
          queryClient.invalidateQueries({ queryKey }),
        ),
      ),
  });
}
