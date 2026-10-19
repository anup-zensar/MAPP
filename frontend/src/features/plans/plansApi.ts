/**
 * What the Plans pages ask the API (plan "Data"): the list, one plan, the defaults of a new plan,
 * and every change. Each change refreshes the lists and the plan when it is done — also when it
 * failed, because a refused change may mean someone else changed the plan meanwhile.
 */
import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { QueryClient } from '@tanstack/react-query';
import { api, call } from '@/api/client';
import type { components } from '@/api/schema';
import type { PlanListFilters } from './listPreferences';

export type Plan = components['schemas']['Plan'];
export type PlanSummary = components['schemas']['PlanSummary'];
export type PlanList = components['schemas']['PlanList'];
export type PlanDefaults = components['schemas']['PlanDefaults'];
export type PlanCreate = components['schemas']['PlanCreate'];
export type PlanUpdate = components['schemas']['PlanUpdate'];
export type PlanChangeEffects = components['schemas']['PlanChangeEffects'];
export type PlanChangeEffectsRequest = components['schemas']['PlanChangeEffectsRequest'];

/** The filters the API takes (the layout is the page's own business). */
export type PlanQuery = Omit<PlanListFilters, 'layout'>;

export const PLAN_KEYS = {
  all: ['plans'] as const,
  list: (query: PlanQuery) => ['plans', query] as const,
  plan: (planId: string) => ['plan', planId] as const,
  defaults: ['plan-defaults'] as const,
  copyOptions: (planId: string) => ['plan-copy-options', planId] as const,
};

/** The plans matching the filters, newest year first; the old answer stays while a new one loads. */
export function usePlans(query: PlanQuery) {
  return useQuery({
    queryKey: PLAN_KEYS.list(query),
    queryFn: () =>
      call(
        api.GET('/api/plans', {
          params: {
            query: {
              scope: query.scope,
              ...(query.status !== 'all' && { status: query.status }),
              ...(query.year !== null && { year: query.year }),
              ...(query.q.trim() !== '' && { q: query.q.trim() }),
            },
          },
        }),
      ),
    placeholderData: keepPreviousData,
  });
}

/** One plan with its figures, goals and completeness. */
export function usePlan(planId: string) {
  return useQuery({
    queryKey: PLAN_KEYS.plan(planId),
    queryFn: () => fetchPlan(planId),
  });
}

/** The defaults of a new plan for the signed-in person. */
export function usePlanDefaults() {
  return useQuery({
    queryKey: PLAN_KEYS.defaults,
    queryFn: () => call(api.GET('/api/plan-defaults')),
    staleTime: 0,
  });
}

/** The plan as stored now (after a conflict, the edit form takes its version from it). */
export function fetchLatestPlan(queryClient: QueryClient, planId: string): Promise<Plan> {
  return queryClient.fetchQuery({
    queryKey: PLAN_KEYS.plan(planId),
    queryFn: () => fetchPlan(planId),
    staleTime: 0,
  });
}

export function useCreatePlan() {
  const refresh = useRefresh();
  return useMutation({
    mutationFn: (body: PlanCreate) => call(api.POST('/api/plans', { body })),
    onSettled: (plan) => refresh(plan?.id),
  });
}

export function useUpdatePlan(planId: string) {
  const refresh = useRefresh();
  return useMutation({
    // The version comes from the form, never from the cache (edit-form rule).
    mutationFn: (body: PlanUpdate) =>
      call(api.PATCH('/api/plans/{planId}', { params: { path: { planId } }, body })),
    onSettled: () => refresh(planId),
  });
}

/** What a new model or business unit would change; nothing is saved. */
export function useChangeEffects(planId: string) {
  return useMutation({
    mutationFn: (body: PlanChangeEffectsRequest) =>
      call(api.POST('/api/plans/{planId}/change-effects', { params: { path: { planId } }, body })),
  });
}

export function usePublishPlan() {
  const refresh = useRefresh();
  return useMutation({
    mutationFn: (planId: string) =>
      call(api.POST('/api/plans/{planId}/publish', { params: { path: { planId } } })),
    onSettled: (_plan, _error, planId) => refresh(planId),
  });
}

export function useUnpublishPlan() {
  const refresh = useRefresh();
  return useMutation({
    mutationFn: ({ planId, reason }: { planId: string; reason: string }) =>
      call(
        api.POST('/api/plans/{planId}/unpublish', {
          params: { path: { planId } },
          body: { reason },
        }),
      ),
    onSettled: (_plan, _error, { planId }) => refresh(planId),
  });
}

/** Moves a plan and everything in it to the recycle bin; answers the entry for Undo. */
export function useDeletePlan() {
  const refresh = useRefresh();
  return useMutation({
    mutationFn: (planId: string) =>
      call(api.DELETE('/api/plans/{planId}', { params: { path: { planId } } })),
    // The plan itself is not asked again: it is gone, and its page is being left.
    onSettled: () => refresh(),
  });
}

function fetchPlan(planId: string) {
  return call(api.GET('/api/plans/{planId}', { params: { path: { planId } } }));
}

/** Asks again for every list of plans, the plan, and the recycle bin and activity log. */
function useRefresh() {
  const queryClient = useQueryClient();
  return (planId?: string) =>
    Promise.all([
      queryClient.invalidateQueries({ queryKey: PLAN_KEYS.all }),
      planId && queryClient.invalidateQueries({ queryKey: PLAN_KEYS.plan(planId) }),
      queryClient.invalidateQueries({ queryKey: ['recycle-bin'] }),
      queryClient.invalidateQueries({ queryKey: ['activity-log'] }),
    ]);
}
