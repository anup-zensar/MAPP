import { useQuery } from '@tanstack/react-query';
import { api, call } from '@/api/client';
import { ApiError } from '@/api/errors';
import type { components } from '@/api/generated/schema';
import { editActivityFormKey } from './api';

type ModelFacts = components['schemas']['ActivityModelFacts'];
type Plan = components['schemas']['Plan'];

export interface ActivityPlanFacts {
  businessUnits: string[] | undefined;
  model: ModelFacts | undefined;
  error: unknown;
  refetch: () => void;
}

// From the edit form for people who may edit the activity, otherwise from the plan, which every
// reader of the plan may open; a named approver who cannot open it reads no business units.
export function useActivityPlanFacts(
  activityId: string,
  planId: string,
  mayEdit: boolean,
): ActivityPlanFacts {
  const form = useQuery({
    queryKey: editActivityFormKey(activityId),
    queryFn: () =>
      call(api.GET('/api/v1/activities/{activityId}/form', { params: { path: { activityId } } })),
    enabled: mayEdit,
    staleTime: 0,
    refetchOnMount: 'always',
  });
  const plan = useQuery({
    queryKey: ['plan', planId],
    queryFn: () => call(api.GET('/api/v1/plans/{planId}', { params: { path: { planId } } })),
    enabled: !mayEdit,
  });
  if (!mayEdit) {
    return { ...factsOfPlan(plan.data, plan.error), refetch: () => void plan.refetch() };
  }
  const context = form.data?.context;
  return {
    businessUnits: context?.businessUnits.map((unit) => unit.id),
    model: context?.model,
    error: form.error,
    refetch: () => void form.refetch(),
  };
}

function factsOfPlan(plan: Plan | undefined, error: unknown): Omit<ActivityPlanFacts, 'refetch'> {
  if (error instanceof ApiError && error.kind === 'notFound') {
    return { businessUnits: [], model: undefined, error: null };
  }
  return {
    businessUnits: plan?.businessUnit.map((option) => option.id),
    model: plan?.model,
    error,
  };
}
