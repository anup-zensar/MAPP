/**
 * What the Finance budget items page asks the API (feature 002, US5): a year's finance budgets with
 * their items, and every change to items and budget names. Every change refreshes the budgets of
 * every year (a new item may create a budget) and the activity log, recycle bin and Settings counts.
 */
import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { QueryKey } from '@tanstack/react-query';
import { api, call } from '@/api/client';
import type { components } from '@/api/generated/schema';
import { SETTINGS_KEYS } from '@/api/settingsQueries';

export type FinanceBudget = components['schemas']['FinanceBudget'];
export type FinanceBudgetPage = components['schemas']['FinanceBudgetPage'];
export type FinanceBudgetItem = components['schemas']['FinanceBudgetItem'];
export type FinanceBudgetItemCreate = components['schemas']['FinanceBudgetItemCreate'];
export type FinanceBudgetItemUpdate = components['schemas']['FinanceBudgetItemUpdate'];

/** The lists a change to an item or a budget shows in. */
export const FINANCE_LISTS: QueryKey[] = [
  ['finance-budgets'],
  SETTINGS_KEYS.overview,
  ['activity-log'],
  ['recycle-bin'],
];

/**
 * The finance budgets of a year with their items, and the years to choose from (the current year
 * when `year` is left out). The same query as useFinanceBudgets; while another year is on its way
 * the last one stays on screen, so the year filter keeps its place and its focus.
 */
export function useFinanceBudgetPage(year?: number) {
  return useQuery({
    queryKey: SETTINGS_KEYS.financeBudgets(year),
    queryFn: () =>
      call(api.GET('/api/v1/finance-budgets', { params: { query: year ? { year } : {} } })),
    placeholderData: keepPreviousData,
  });
}

/**
 * A finance budget as it is stored now (from its year's list: budgets have no address of their
 * own); after a conflict the rename takes its version from it. Undefined when it is gone.
 */
export async function fetchFinanceBudget(
  year: number,
  id: string,
): Promise<FinanceBudget | undefined> {
  const page = await call(api.GET('/api/v1/finance-budgets', { params: { query: { year } } }));
  return page.items.find((budget) => budget.id === id);
}

/** One item as it is stored now; after a conflict the form takes its version from it. */
export function fetchFinanceBudgetItem(id: string): Promise<FinanceBudgetItem> {
  return call(
    api.GET('/api/v1/finance-budget-items/{itemId}', { params: { path: { itemId: id } } }),
  );
}

/** POST …/status: sets an item inactive or reactivates it (FR-004). */
export function setFinanceBudgetItemStatus(id: string, body: { active: boolean; version: number }) {
  return call(
    api.POST('/api/v1/finance-budget-items/{itemId}/status', {
      params: { path: { itemId: id } },
      body,
    }),
  );
}

export function useCreateFinanceBudgetItem() {
  const refresh = useRefreshFinanceLists();
  return useMutation({
    mutationFn: (body: FinanceBudgetItemCreate) =>
      call(api.POST('/api/v1/finance-budget-items', { body })),
    onSettled: refresh,
  });
}

/** Changes an item: only the fields sent, with the version the form started from (D-203). */
export function useUpdateFinanceBudgetItem() {
  const refresh = useRefreshFinanceLists();
  return useMutation({
    mutationFn: ({ id, body }: { id: string; body: FinanceBudgetItemUpdate }) =>
      call(
        api.PATCH('/api/v1/finance-budget-items/{itemId}', {
          params: { path: { itemId: id } },
          body,
        }),
      ),
    onSettled: refresh,
  });
}

/** Deletes an unused item (to the recycle bin). */
export function useDeleteFinanceBudgetItem() {
  const refresh = useRefreshFinanceLists();
  return useMutation({
    mutationFn: (id: string) =>
      call(
        api.DELETE('/api/v1/finance-budget-items/{itemId}', { params: { path: { itemId: id } } }),
      ),
    onSettled: refresh,
  });
}

/** Renames a finance budget: all its items, and the plans using it as their default, follow (FR-015). */
export function useRenameFinanceBudget() {
  const refresh = useRefreshFinanceLists();
  return useMutation({
    mutationFn: ({ id, name, version }: { id: string; name: string; version: number }) =>
      call(
        api.PATCH('/api/v1/finance-budgets/{budgetId}', {
          params: { path: { budgetId: id } },
          body: { name, version },
        }),
      ),
    onSettled: refresh,
  });
}

function useRefreshFinanceLists() {
  const queryClient = useQueryClient();
  return () =>
    Promise.all(FINANCE_LISTS.map((queryKey) => queryClient.invalidateQueries({ queryKey })));
}
