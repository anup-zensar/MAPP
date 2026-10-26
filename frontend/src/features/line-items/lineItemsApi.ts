/**
 * What the line-item screens ask the API (007 plan, "Data"). Every change refreshes the line
 * items, the activities and plans (balance, Ready to submit, completeness: FR-023), the recycle bin
 * and the activity log when it is done, also when it failed.
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { QueryKey } from '@tanstack/react-query';
import { api, call } from '@/api/client';
import type { components } from '@/api/generated/schema';

export type LineItem = components['schemas']['LineItem'];
export type LineItemSection = components['schemas']['LineItemSection'];
export type LineItemDetails = components['schemas']['LineItemDetails'];
export type LineItemFormOptions = components['schemas']['LineItemFormOptions'];
export type LineItemActivity = components['schemas']['LineItemActivity'];
export type LineItemCreate = components['schemas']['LineItemCreate'];
export type LineItemUpdate = components['schemas']['LineItemUpdate'];
export type UnassignedBalance = components['schemas']['UnassignedBalance'];
export type LineItemCurrency = components['schemas']['LineItemCurrency'];
export type FinanceBudgetItemRef = components['schemas']['FinanceBudgetItemRef'];
export type Money = components['schemas']['Money'];

/**
 * Everything a line-item change shows in; Undo refreshes them too: the line items, the activity's
 * page (its unassigned planned cost, completeness and, after a restart, status and approval —
 * features 006 and 008), the activity rows and roll-ups of the plan tree (004, 005, 006), the
 * recycle bin and the activity log.
 */
export const LINE_ITEM_CHANGE_KEYS: QueryKey[] = [
  ['line-items'],
  ['activity'],
  ['approval'],
  ['plan-activities'],
  ['plans'],
  ['plan'],
  ['campaigns'],
  ['initiatives'],
  ['recycle-bin'],
  ['activity-log'],
];

/** The Line items section of an activity: rows, balance and grid (FR-001 – FR-004). */
export function useLineItemSection(activityId: string, enabled = true) {
  return useQuery({
    queryKey: ['line-items', 'activity', activityId],
    queryFn: () =>
      call(
        api.GET('/api/v1/activities/{activityId}/line-items', { params: { path: { activityId } } }),
      ),
    enabled,
  });
}

/** Defaults and choices of a new or edited line item (FR-008, FR-011, FR-015). */
export function useLineItemFormOptions(activityId: string, enabled = true) {
  return useQuery({
    queryKey: ['line-items', 'form-options', activityId],
    queryFn: () =>
      call(
        api.GET('/api/v1/activities/{activityId}/line-items/form-options', {
          params: { path: { activityId } },
        }),
      ),
    enabled,
    // The balance and the default name change with every line item: always ask again.
    staleTime: 0,
    refetchOnMount: 'always',
  });
}

/** One line item with its activity and the activity's balance (FR-026). */
export function useLineItem(lineItemId: string) {
  return useQuery({
    queryKey: ['line-items', 'item', lineItemId],
    queryFn: () =>
      call(api.GET('/api/v1/line-items/{lineItemId}', { params: { path: { lineItemId } } })),
  });
}

/** Creates a line item in an activity. */
export function useCreateLineItem(activityId: string) {
  const refresh = useRefresh();
  return useMutation({
    mutationFn: (body: LineItemCreate) =>
      call(
        api.POST('/api/v1/activities/{activityId}/line-items', {
          params: { path: { activityId } },
          body,
        }),
      ),
    onSettled: refresh,
  });
}

/** Saves the whole form of a line item with the version it started from (D-153). */
export function useUpdateLineItem(lineItemId: string) {
  const refresh = useRefresh();
  return useMutation({
    mutationFn: (body: LineItemUpdate) =>
      call(
        api.PATCH('/api/v1/line-items/{lineItemId}', { params: { path: { lineItemId } }, body }),
      ),
    onSettled: refresh,
  });
}

/** Copies a line item in its activity; answers the copy (FR-029). */
export function useDuplicateLineItem() {
  const refresh = useRefresh();
  return useMutation({
    mutationFn: ({
      lineItemId,
      restartApproval,
    }: {
      lineItemId: string;
      restartApproval: boolean;
    }) =>
      call(
        api.POST('/api/v1/line-items/{lineItemId}/duplicate', {
          params: { path: { lineItemId } },
          body: { restartApproval },
        }),
      ),
    onSettled: refresh,
  });
}

/** Moves a line item to the recycle bin; answers the entry for Undo (FR-030). */
export function useDeleteLineItem() {
  const refresh = useRefresh();
  return useMutation({
    mutationFn: ({
      lineItemId,
      restartApproval,
    }: {
      lineItemId: string;
      restartApproval: boolean;
    }) =>
      call(
        api.DELETE('/api/v1/line-items/{lineItemId}', {
          params: {
            path: { lineItemId },
            query: restartApproval ? { restartApproval: true } : undefined,
          },
        }),
      ),
    // Not waited for: the page that deleted leaves at once, before its own record answers 404.
    onSettled: () => {
      void refresh();
    },
  });
}

/** Refreshes what a change shows in; a change waits for it, so the pages are current. */
function useRefresh() {
  const queryClient = useQueryClient();
  return () =>
    Promise.all(
      LINE_ITEM_CHANGE_KEYS.map((queryKey) => queryClient.invalidateQueries({ queryKey })),
    );
}

/** Reads a line item again, bypassing the cache: the version to save over after a conflict. */
export function useFetchLatestLineItem() {
  const queryClient = useQueryClient();
  return (lineItemId: string) =>
    queryClient.fetchQuery({
      queryKey: ['line-items', 'item', lineItemId],
      queryFn: () =>
        call(api.GET('/api/v1/line-items/{lineItemId}', { params: { path: { lineItemId } } })),
      staleTime: 0,
    });
}
