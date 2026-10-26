/**
 * What the Global budgets pages ask the API (014 US6, FR-029 – FR-036): the global budgets of a
 * year, one global budget with its figures and allocations, the plans it can be allocated to, and
 * every change. An allocation sets its plan's budget in the same save, so every change asks every
 * query again but the session (plans, Home and All work show those budgets).
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api, call } from '@/api/client';
import type { components } from '@/api/generated/schema';
import { invalidateAllButSession } from '@/features/approvals/approvalsApi';

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

/** The global budgets of a year (every year when null), with the years and New's availability. */
export function useGlobalBudgets(year: number | null) {
  return useQuery({
    queryKey: [...GLOBAL_BUDGETS_KEY, 'list', year],
    queryFn: () =>
      call(api.GET('/api/v1/global-budgets', { params: { query: year === null ? {} : { year } } })),
  });
}

/** One global budget with its figures and the allocations the person may see. */
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

/** The plans of the budget's year without an allocation that the person may allocate to. */
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

/** Only the changed fields, with the version the person started from (D-203). */
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

/** Moves a global budget without allocations to the recycle bin; answers the entry for Undo. */
export function useDeleteGlobalBudget() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      call(
        api.DELETE('/api/v1/global-budgets/{globalBudgetId}', {
          params: { path: { globalBudgetId: id } },
        }),
      ),
    // Everything is refreshed but the deleted global budget itself: its page is closing, and asking
    // for it again would only answer "not found". After a refusal it is refreshed too.
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

/** Allocates part of the global budget to a plan; the plan's budget becomes the amount. */
export function useAllocate() {
  const refresh = useRefresh();
  return useMutation({
    mutationFn: ({
      globalBudgetId,
      planId,
      amount,
      planVersion,
    }: {
      globalBudgetId: string;
      planId: string;
      amount: MoneyInput;
      planVersion: number;
    }) =>
      call(
        api.POST('/api/v1/global-budgets/{globalBudgetId}/allocations', {
          params: { path: { globalBudgetId } },
          body: { planId, amount, planVersion },
        }),
      ),
    onSettled: refresh,
  });
}

/** Changes an allocation's amount; the plan's budget follows in the same save. */
export function useUpdateAllocation() {
  const refresh = useRefresh();
  return useMutation({
    mutationFn: ({
      allocationId,
      amount,
      version,
      planVersion,
    }: {
      allocationId: string;
      amount: MoneyInput;
      version: number;
      planVersion: number;
    }) =>
      call(
        api.PATCH('/api/v1/budget-allocations/{allocationId}', {
          params: { path: { allocationId } },
          body: { amount, version, planVersion },
        }),
      ),
    onSettled: refresh,
  });
}

/** Removes an allocation: the plan keeps its budget, which becomes editable again (FR-034). */
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
