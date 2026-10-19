/**
 * What the activity pages ask the API (006 plan, "Data"): one activity, the two forms, a plan's
 * activities as tree rows, and every change. Every change, when settled, refreshes the activity,
 * the plan's activities and the planning views above it (plans, campaigns, initiatives), the
 * recycle bin and the activity log, whatever the answer.
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { QueryClient, QueryKey } from '@tanstack/react-query';
import { api, call } from '@/api/client';
import type { components } from '@/api/schema';

export type Activity = components['schemas']['Activity'];
export type ActivitySummary = components['schemas']['ActivitySummary'];
export type ActivityFormData = components['schemas']['ActivityForm'];
export type ActivityCreate = components['schemas']['ActivityCreate'];
export type ActivityUpdate = components['schemas']['ActivityUpdate'];
export type ActivitySubmitProblem = components['schemas']['ActivitySubmitProblem'];
export type ActivityTypeOption = components['schemas']['ActivityTypeOption'];
export type ActivityWorkflowOption = components['schemas']['ActivityWorkflowOption'];
export type Money = components['schemas']['Money'];

export const activityKey = (id: string) => ['activity', id] as const;
export const newActivityFormKey = (initiativeId: string) =>
  ['activity-form', 'new', initiativeId] as const;
export const editActivityFormKey = (id: string) => ['activity-form', 'edit', id] as const;
export const planActivitiesKey = (planId: string) => ['plan-activities', planId] as const;

/**
 * The lists an activity change can show in: the activity itself and its forms, the plan's
 * activity rows, 004's plans, plan page and plan tree, 005's campaigns and initiatives (their
 * planned costs roll up), the recycle bin and the activity log.
 */
export const ACTIVITY_PLANNING_KEYS: QueryKey[] = [
  ['activity'],
  ['activity-form'],
  // Its line items' spreads and unassigned planned cost (007), and its approval (008).
  ['line-items'],
  ['approval'],
  ['plan-activities'],
  ['plans'],
  ['plan'],
  ['plan-tree'],
  ['campaigns'],
  ['initiatives'],
  ['recycle-bin'],
  ['activity-log'],
];

/** What a delete refreshes: everything but the deleted activity, its forms, line items and approval. */
const LISTS_AFTER_DELETE = ACTIVITY_PLANNING_KEYS.slice(4);

/** An activity with its status, completeness and ready-to-submit check (GET /api/activities/{id}). */
export function useActivity(id: string) {
  return useQuery({
    queryKey: activityKey(id),
    queryFn: () =>
      call(api.GET('/api/activities/{activityId}', { params: { path: { activityId: id } } })),
  });
}

/**
 * Everything the new-activity form needs, with its defaults. Asked again each time the form
 * opens, so the names in use and the Settings are current.
 */
export function useNewActivityForm(initiativeId: string) {
  return useQuery({
    queryKey: newActivityFormKey(initiativeId),
    queryFn: () =>
      call(
        api.GET('/api/initiatives/{initiativeId}/activity-form', {
          params: { path: { initiativeId } },
        }),
      ),
    staleTime: 0,
    refetchOnMount: 'always',
  });
}

/** Everything the edit form needs (no defaults); asked again each time the form opens. */
export function useEditActivityForm(id: string) {
  return useQuery({
    queryKey: editActivityFormKey(id),
    queryFn: () =>
      call(api.GET('/api/activities/{activityId}/form', { params: { path: { activityId: id } } })),
    staleTime: 0,
    refetchOnMount: 'always',
  });
}

/** Every activity of a plan, as plan-tree rows (GET /api/plans/{planId}/activities). */
export function usePlanActivities(planId: string) {
  return useQuery({
    queryKey: planActivitiesKey(planId),
    queryFn: () =>
      call(api.GET('/api/plans/{planId}/activities', { params: { path: { planId } } })),
  });
}

/**
 * Asks for the activity again and returns it as it is stored now. After a conflict, the edit
 * form takes the latest version from it.
 */
export function fetchLatestActivity(queryClient: QueryClient, id: string): Promise<Activity> {
  // staleTime 0: always ask the API, never answer from the cache.
  return queryClient.fetchQuery({
    queryKey: activityKey(id),
    queryFn: () =>
      call(api.GET('/api/activities/{activityId}', { params: { path: { activityId: id } } })),
    staleTime: 0,
  });
}

/** A new activity under an initiative. */
export interface NewActivity {
  initiativeId: string;
  body: ActivityCreate;
}

/** Creates an activity (status Created). */
export function useCreateActivity() {
  const refresh = useRefreshPlanning();
  return useMutation({
    mutationFn: ({ initiativeId, body }: NewActivity) =>
      call(
        api.POST('/api/initiatives/{initiativeId}/activities', {
          params: { path: { initiativeId } },
          body,
        }),
      ),
    onSettled: refresh,
  });
}

/** An edited activity: every editable field and the version the form opened with. */
export interface EditedActivity {
  id: string;
  body: ActivityUpdate;
}

/**
 * Saves an edited activity. The version comes from the edit form and never from the cache (001's
 * edit-form rule), so a save after someone else's change is refused with a conflict.
 */
export function useUpdateActivity() {
  const refresh = useRefreshPlanning();
  return useMutation({
    mutationFn: ({ id, body }: EditedActivity) =>
      call(
        api.PATCH('/api/activities/{activityId}', { params: { path: { activityId: id } }, body }),
      ),
    onSettled: refresh,
  });
}

/**
 * Moves an activity and its line items to the recycle bin; answers the entry for Undo. The deleted
 * activity itself is not asked again (its page is left), only the lists it was in.
 */
export function useDeleteActivity() {
  const refresh = useRefreshPlanning(LISTS_AFTER_DELETE);
  return useMutation({
    mutationFn: (id: string) =>
      call(api.DELETE('/api/activities/{activityId}', { params: { path: { activityId: id } } })),
    onSettled: refresh,
  });
}

/**
 * Refreshes everything an activity change can show in. A change waits for this before it is done,
 * so the next page already shows the new state.
 */
function useRefreshPlanning(keys: QueryKey[] = ACTIVITY_PLANNING_KEYS) {
  const queryClient = useQueryClient();
  return () => Promise.all(keys.map((queryKey) => queryClient.invalidateQueries({ queryKey })));
}
