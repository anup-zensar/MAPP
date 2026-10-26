/**
 * What the Activity types page asks the API (feature 002, US5). The list comes from
 * useActivityTypes() (api/settingsQueries.ts); every change refreshes the lists that show
 * activity types — models list the types they allow, and a type's models are edited from it — and
 * the activity log, recycle bin and Settings counts.
 */
import { useMutation, useQueryClient } from '@tanstack/react-query';
import type { QueryKey } from '@tanstack/react-query';
import { api, call } from '@/api/client';
import type { components } from '@/api/schema';
import { SETTINGS_KEYS } from '@/api/settingsQueries';

export type ActivityType = components['schemas']['ActivityType'];
export type ActivityTypeCreate = components['schemas']['ActivityTypeCreate'];
export type ActivityTypeUpdate = components['schemas']['ActivityTypeUpdate'];
export type ChangeImpact = components['schemas']['ChangeImpact'];

/** The lists a change to an activity type shows in. */
export const ACTIVITY_TYPE_LISTS: QueryKey[] = [
  SETTINGS_KEYS.activityTypes,
  SETTINGS_KEYS.models,
  SETTINGS_KEYS.overview,
  ['activity-log'],
  ['recycle-bin'],
];

/** One activity type as it is stored now; after a conflict the form takes its version from it. */
export function fetchActivityType(id: string): Promise<ActivityType> {
  return call(
    api.GET('/api/v1/activity-types/{activityTypeId}', {
      params: { path: { activityTypeId: id } },
    }),
  );
}

/** What removing models from the type would affect, before saving (FR-012). Saves nothing. */
export function checkActivityTypeImpact(
  id: string,
  body: ActivityTypeUpdate,
): Promise<ChangeImpact> {
  return call(
    api.POST('/api/v1/activity-types/{activityTypeId}/impact', {
      params: { path: { activityTypeId: id } },
      body,
    }),
  );
}

/** POST …/status: sets an activity type inactive or reactivates it (FR-004). */
export function setActivityTypeStatus(id: string, body: { active: boolean; version: number }) {
  return call(
    api.POST('/api/v1/activity-types/{activityTypeId}/status', {
      params: { path: { activityTypeId: id } },
      body,
    }),
  );
}

export function useCreateActivityType() {
  const refresh = useRefreshActivityTypeLists();
  return useMutation({
    mutationFn: (body: ActivityTypeCreate) => call(api.POST('/api/v1/activity-types', { body })),
    onSettled: refresh,
  });
}

/** Changes an activity type: only the fields sent, with the version the form started from (D-203). */
export function useUpdateActivityType() {
  const refresh = useRefreshActivityTypeLists();
  return useMutation({
    mutationFn: ({ id, body }: { id: string; body: ActivityTypeUpdate }) =>
      call(
        api.PATCH('/api/v1/activity-types/{activityTypeId}', {
          params: { path: { activityTypeId: id } },
          body,
        }),
      ),
    onSettled: refresh,
  });
}

/** Deletes an unused activity type (to the recycle bin). */
export function useDeleteActivityType() {
  const refresh = useRefreshActivityTypeLists();
  return useMutation({
    mutationFn: (id: string) =>
      call(
        api.DELETE('/api/v1/activity-types/{activityTypeId}', {
          params: { path: { activityTypeId: id } },
        }),
      ),
    onSettled: refresh,
  });
}

function useRefreshActivityTypeLists() {
  const queryClient = useQueryClient();
  return () =>
    Promise.all(ACTIVITY_TYPE_LISTS.map((queryKey) => queryClient.invalidateQueries({ queryKey })));
}
