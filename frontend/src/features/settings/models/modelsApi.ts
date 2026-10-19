/**
 * What the Models page asks the API (feature 002, US1). The list comes from useModels()
 * (api/settingsQueries.ts); every change refreshes the lists that show models — activity types
 * list the models that allow them — and the activity log, recycle bin and Settings counts.
 */
import { useMutation, useQueryClient } from '@tanstack/react-query';
import type { QueryKey } from '@tanstack/react-query';
import { api, call } from '@/api/client';
import type { components } from '@/api/schema';
import { SETTINGS_KEYS } from '@/api/settingsQueries';

export type Model = components['schemas']['Model'];
export type ModelCreate = components['schemas']['ModelCreate'];
export type ModelUpdate = components['schemas']['ModelUpdate'];
export type ChangeImpact = components['schemas']['ChangeImpact'];

/** The lists a change to a model shows in. */
export const MODEL_LISTS: QueryKey[] = [
  SETTINGS_KEYS.models,
  SETTINGS_KEYS.activityTypes,
  SETTINGS_KEYS.overview,
  ['activity-log'],
  ['recycle-bin'],
];

/** One model as it is stored now; after a conflict the form takes its version from it. */
export function fetchModel(id: string): Promise<Model> {
  return call(api.GET('/api/models/{modelId}', { params: { path: { modelId: id } } }));
}

/** What a change would affect, before it is saved (FR-011, FR-012). Saves nothing. */
export function checkModelImpact(id: string, body: ModelUpdate): Promise<ChangeImpact> {
  return call(
    api.POST('/api/models/{modelId}/impact', { params: { path: { modelId: id } }, body }),
  );
}

/** POST …/status: sets a model inactive or reactivates it (FR-004). */
export function setModelStatus(id: string, body: { active: boolean; version: number }) {
  return call(
    api.POST('/api/models/{modelId}/status', { params: { path: { modelId: id } }, body }),
  );
}

/** Deletes an unused model (to the recycle bin). */
export function deleteModel(id: string) {
  return call(api.DELETE('/api/models/{modelId}', { params: { path: { modelId: id } } }));
}

export function useCreateModel() {
  const refresh = useRefreshModelLists();
  return useMutation({
    mutationFn: (body: ModelCreate) => call(api.POST('/api/models', { body })),
    onSettled: refresh,
  });
}

/** Changes a model: only the fields sent, with the version the form started from (D-203). */
export function useUpdateModel() {
  const refresh = useRefreshModelLists();
  return useMutation({
    mutationFn: ({ id, body }: { id: string; body: ModelUpdate }) =>
      call(api.PATCH('/api/models/{modelId}', { params: { path: { modelId: id } }, body })),
    onSettled: refresh,
  });
}

export function useDeleteModel() {
  const refresh = useRefreshModelLists();
  return useMutation({ mutationFn: deleteModel, onSettled: refresh });
}

function useRefreshModelLists() {
  const queryClient = useQueryClient();
  return () =>
    Promise.all(MODEL_LISTS.map((queryKey) => queryClient.invalidateQueries({ queryKey })));
}
