import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { QueryClient } from '@tanstack/react-query';
import { api, call } from '@/api/client';
import type { components } from '@/api/generated/schema';
import type { PlanListFilters } from './list-preferences';

export type Plan = components['schemas']['Plan'];
export type PlanSummary = components['schemas']['PlanSummary'];
export type PlanList = components['schemas']['PlanList'];
export type PlanDefaults = components['schemas']['PlanDefaults'];
export type PlanCreate = components['schemas']['PlanCreate'];
export type PlanUpdate = components['schemas']['PlanUpdate'];
export type PlanChangeEffects = components['schemas']['PlanChangeEffects'];
export type PlanChangeEffectsRequest = components['schemas']['PlanChangeEffectsRequest'];
export type PlanCopyOptions = components['schemas']['PlanCopyOptions'];
export type PlanCopyRequest = components['schemas']['PlanCopyRequest'];

export type PlanQuery = Omit<PlanListFilters, 'layout'>;

export const PLAN_KEYS = {
  all: ['plans'] as const,
  list: (query: PlanQuery) => ['plans', query] as const,
  plan: (planId: string) => ['plan', planId] as const,
  defaults: ['plan-defaults'] as const,
  copyOptions: (planId: string) => ['plan-copy-options', planId] as const,
};

function yearParams({ yearFrom, yearTo }: Pick<PlanQuery, 'yearFrom' | 'yearTo'>) {
  if (yearFrom === null || yearTo === null) return {};
  return yearFrom === yearTo ? { year: yearFrom } : { yearFrom, yearTo };
}

export function usePlans(query: PlanQuery) {
  return useQuery({
    queryKey: PLAN_KEYS.list(query),
    queryFn: () =>
      call(
        api.GET('/api/v1/plans', {
          params: {
            query: {
              scope: query.scope,
              ...(query.status !== 'all' && { status: query.status }),
              ...yearParams(query),
              ...(query.types.length > 0 && { type: query.types }),
              ...(query.q.trim() !== '' && { q: query.q.trim() }),
              ...(query.bcpir !== 'include' && { bcpir: query.bcpir }),
            },
          },
        }),
      ),
    placeholderData: keepPreviousData,
  });
}

export function usePlan(planId: string) {
  return useQuery({
    queryKey: PLAN_KEYS.plan(planId),
    queryFn: () => fetchPlan(planId),
  });
}

export function usePlanDefaults() {
  return useQuery({
    queryKey: PLAN_KEYS.defaults,
    queryFn: () => call(api.GET('/api/v1/plan-defaults')),
    staleTime: 0,
  });
}

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
    mutationFn: (body: PlanCreate) => call(api.POST('/api/v1/plans', { body })),
    onSettled: (plan) => refresh(plan?.id),
  });
}

export function useUpdatePlan(planId: string) {
  const refresh = useRefresh();
  return useMutation({
    mutationFn: (body: PlanUpdate) =>
      call(api.PATCH('/api/v1/plans/{planId}', { params: { path: { planId } }, body })),
    onSettled: () => refresh(planId),
  });
}

export function useChangeEffects(planId: string) {
  return useMutation({
    mutationFn: (body: PlanChangeEffectsRequest) =>
      call(
        api.POST('/api/v1/plans/{planId}/change-effects', { params: { path: { planId } }, body }),
      ),
  });
}

export function usePublishPlan() {
  const refresh = useRefresh();
  return useMutation({
    mutationFn: (planId: string) =>
      call(api.POST('/api/v1/plans/{planId}/publish', { params: { path: { planId } } })),
    onSettled: (_plan, _error, planId) => refresh(planId),
  });
}

export function useUnpublishPlan() {
  const refresh = useRefresh();
  return useMutation({
    mutationFn: ({ planId, reason }: { planId: string; reason: string }) =>
      call(
        api.POST('/api/v1/plans/{planId}/unpublish', {
          params: { path: { planId } },
          body: { reason },
        }),
      ),
    onSettled: (_plan, _error, { planId }) => refresh(planId),
  });
}

export function useCopyOptions(planId: string, enabled: boolean) {
  return useQuery({
    queryKey: PLAN_KEYS.copyOptions(planId),
    queryFn: () =>
      call(api.GET('/api/v1/plans/{planId}/copy-options', { params: { path: { planId } } })),
    enabled,
    staleTime: 0,
  });
}

export function useCopyPlan() {
  const queryClient = useQueryClient();
  const refresh = useRefresh();
  return useMutation({
    mutationFn: ({ planId, body }: { planId: string; body: PlanCopyRequest }) =>
      call(api.POST('/api/v1/plans/{planId}/copy', { params: { path: { planId } }, body })),
    onSettled: (_result, _error, { planId }) =>
      Promise.all([
        refresh(planId),
        queryClient.invalidateQueries({ queryKey: PLAN_KEYS.copyOptions(planId) }),
      ]),
  });
}

export function useDeletePlan() {
  const refresh = useRefresh();
  return useMutation({
    mutationFn: (planId: string) =>
      call(api.DELETE('/api/v1/plans/{planId}', { params: { path: { planId } } })),
    // The deleted plan's own queries are left out: asked again they would answer 404.
    onSuccess: (_result, planId) => refresh(undefined, planId),
    onError: () => refresh(),
  });
}

function fetchPlan(planId: string) {
  return call(api.GET('/api/v1/plans/{planId}', { params: { path: { planId } } }));
}

function useRefresh() {
  const queryClient = useQueryClient();
  return (planId?: string, deletedId?: string) => {
    const predicate = (query: { queryKey: readonly unknown[] }) =>
      deletedId === undefined || !query.queryKey.includes(deletedId);
    return Promise.all([
      queryClient.invalidateQueries({ queryKey: PLAN_KEYS.all, predicate }),
      planId && queryClient.invalidateQueries({ queryKey: PLAN_KEYS.plan(planId) }),
      queryClient.invalidateQueries({ queryKey: ['recycle-bin'] }),
      queryClient.invalidateQueries({ queryKey: ['activity-log'] }),
    ]);
  };
}
