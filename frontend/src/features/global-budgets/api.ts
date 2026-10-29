import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api, call } from '@/api/client';
import type { OverspendMode } from '@/features/overspend';
import type { components } from '@/api/generated/schema';
import { invalidateAllButSession } from '@/features/approvals';

export type GlobalBudget = components['schemas']['GlobalBudget'];
export type GlobalBudgetSummary = components['schemas']['GlobalBudgetSummary'];
export type GlobalBudgetPage = components['schemas']['GlobalBudgetPage'];
export type GlobalBudgetFigures = components['schemas']['GlobalBudgetFigures'];
export type GlobalBudgetCreate = components['schemas']['GlobalBudgetCreate'];
export type GlobalBudgetUpdate = components['schemas']['GlobalBudgetUpdate'];
export type BudgetAllocationRow = components['schemas']['BudgetAllocationRow'];
export type OtherMarketsRow = components['schemas']['OtherMarketsRow'];
export type AllocationOption = components['schemas']['AllocationOption'];
export type AllocationPlanRef = components['schemas']['AllocationPlanRef'];
export type MoneyInput = components['schemas']['MoneyInput'];

export const GLOBAL_BUDGETS_KEY = ['global-budgets'];

export function useGlobalBudgets(year: number | null) {
  return useQuery({
    queryKey: [...GLOBAL_BUDGETS_KEY, 'list', year],
    queryFn: () =>
      call(api.GET('/api/v1/global-budgets', { params: { query: year === null ? {} : { year } } })),
  });
}

export function useGlobalBudget(globalBudgetId: string) {
  return useQuery({
    queryKey: [...GLOBAL_BUDGETS_KEY, 'one', globalBudgetId],
    queryFn: () => fetchGlobalBudget(globalBudgetId),
  });
}

export function fetchGlobalBudget(globalBudgetId: string): Promise<GlobalBudget> {
  return call(
    api.GET('/api/v1/global-budgets/{globalBudgetId}', { params: { path: { globalBudgetId } } }),
  );
}

export function useAllocationOptions(globalBudgetId: string, enabled = true) {
  return useQuery({
    queryKey: [...GLOBAL_BUDGETS_KEY, 'options', globalBudgetId],
    queryFn: () =>
      call(
        api.GET('/api/v1/global-budgets/{globalBudgetId}/allocation-options', {
          params: { path: { globalBudgetId } },
        }),
      ),
    enabled,
  });
}

export function useCreateGlobalBudget() {
  const refresh = useRefresh();
  return useMutation({
    mutationFn: (body: GlobalBudgetCreate) => call(api.POST('/api/v1/global-budgets', { body })),
    onSettled: refresh,
  });
}

export function useUpdateGlobalBudget() {
  const refresh = useRefresh();
  return useMutation({
    mutationFn: ({ id, body }: { id: string; body: GlobalBudgetUpdate }) =>
      call(
        api.PATCH('/api/v1/global-budgets/{globalBudgetId}', {
          params: { path: { globalBudgetId: id } },
          body,
        }),
      ),
    onSettled: refresh,
  });
}

export function useDeleteGlobalBudget() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      call(
        api.DELETE('/api/v1/global-budgets/{globalBudgetId}', {
          params: { path: { globalBudgetId: id } },
        }),
      ),
    onSettled: (_result, error, id) =>
      queryClient.invalidateQueries({
        predicate: ({ queryKey }) =>
          queryKey[0] !== 'session' && !(error === null && isGlobalBudgetKey(queryKey, id)),
      }),
  });
}

function isGlobalBudgetKey(queryKey: readonly unknown[], id: string): boolean {
  return queryKey[0] === GLOBAL_BUDGETS_KEY[0] && queryKey[1] === 'one' && queryKey[2] === id;
}

export function useAllocate() {
  const refresh = useRefresh();
  return useMutation({
    mutationFn: ({
      globalBudgetId,
      planId,
      amount,
      overspend,
      planVersion,
    }: {
      globalBudgetId: string;
      planId: string;
      amount: MoneyInput;
      overspend: OverspendMode;
      planVersion: number;
    }) =>
      call(
        api.POST('/api/v1/global-budgets/{globalBudgetId}/allocations', {
          params: { path: { globalBudgetId } },
          body: { planId, amount, overspend, planVersion },
        }),
      ),
    onSettled: refresh,
  });
}

export function useUpdateAllocation() {
  const refresh = useRefresh();
  return useMutation({
    mutationFn: ({
      allocationId,
      amount,
      overspend,
      version,
      planVersion,
    }: {
      allocationId: string;
      amount: MoneyInput;
      overspend: OverspendMode;
      version: number;
      planVersion: number;
    }) =>
      call(
        api.PATCH('/api/v1/budget-allocations/{allocationId}', {
          params: { path: { allocationId } },
          body: { amount, overspend, version, planVersion },
        }),
      ),
    onSettled: refresh,
  });
}

export function useRemoveAllocation() {
  const refresh = useRefresh();
  return useMutation({
    mutationFn: (allocationId: string) =>
      call(
        api.DELETE('/api/v1/budget-allocations/{allocationId}', {
          params: { path: { allocationId } },
        }),
      ),
    onSettled: refresh,
  });
}

function useRefresh() {
  const queryClient = useQueryClient();
  return () => invalidateAllButSession(queryClient);
}
