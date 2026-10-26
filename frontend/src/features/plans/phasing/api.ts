/**
 * What Budget by month asks the API (016 US3): the plan's budget by month and the phasing saves of
 * the three levels. A save refreshes everything a budget's phasing shows in: the plan and its
 * lists (completeness), the campaigns, the panels and the activity log.
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api, call } from '@/api/client';
import type { components } from '@/api/generated/schema';
import { refreshPlanning } from '../campaigns/api';

export type BudgetByMonth = components['schemas']['BudgetByMonth'];
export type BudgetByMonthRow = components['schemas']['BudgetByMonthRow'];
export type BudgetByMonthPeriods = components['schemas']['BudgetByMonthPeriods'];
export type BudgetPhasing = components['schemas']['BudgetPhasing'];
export type BudgetPhasingUpdate = components['schemas']['BudgetPhasingUpdate'];
export type PhasingMethod = components['schemas']['PhasingMethod'];
export type PhasingLevel = 'plan' | 'campaign' | 'initiative';

/** Under ['plans'], so every plan change asks for it again. */
export const budgetByMonthKey = (planId: string) => ['plans', planId, 'budget-by-month'] as const;

/** The plan, its campaigns and initiatives by month and by quarter, in the plan currency. */
export function useBudgetByMonth(planId: string, enabled = true) {
  return useQuery({
    queryKey: budgetByMonthKey(planId),
    queryFn: () =>
      call(api.GET('/api/v1/plans/{planId}/budget-by-month', { params: { path: { planId } } })),
    enabled,
  });
}

/** Saves one record's phasing through its level's PUT (FR-020 – FR-023). */
export function useSetPhasing(level: PhasingLevel, recordId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: BudgetPhasingUpdate): Promise<void> => {
      switch (level) {
        case 'plan':
          await call(
            api.PUT('/api/v1/plans/{planId}/budget-phasing', {
              params: { path: { planId: recordId } },
              body,
            }),
          );
          return;
        case 'campaign':
          await call(
            api.PUT('/api/v1/campaigns/{campaignId}/budget-phasing', {
              params: { path: { campaignId: recordId } },
              body,
            }),
          );
          return;
        default:
          await call(
            api.PUT('/api/v1/initiatives/{initiativeId}/budget-phasing', {
              params: { path: { initiativeId: recordId } },
              body,
            }),
          );
          return;
      }
    },
    onSettled: () => refreshPlanning(queryClient),
  });
}
