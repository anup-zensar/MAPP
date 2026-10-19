/**
 * What All work asks the API (plan "State"). The view is read once per filter set and kept
 * (staleTime: Infinity, no refetch when the window regains focus): other people's changes appear
 * on the next filter or view change, or through Refresh. While a new answer loads, the previous
 * one stays on screen.
 */
import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { QueryClient } from '@tanstack/react-query';
import { useCallback, useEffect, useRef } from 'react';
import { api, call } from '@/api/client';
import { ApiError } from '@/api/errors';
import { useToast } from '@/components';
import type {
  AllWorkPreferences,
  AllWorkQuery,
  SavedView,
  SavedViewCreate,
  SavedViewPage,
  SavedViewUpdate,
} from './allWorkTypes';

export const ALL_WORK_KEYS = {
  all: ['all-work'] as const,
  view: (query: AllWorkQuery | null) => ['all-work', query] as const,
  /** null: the current year, before the first visit knows which year that is. */
  options: (year: number | null) => ['all-work-options', year] as const,
  preferences: ['all-work-preferences'] as const,
  views: ['saved-views'] as const,
};

/** The view for a filter set (null: not known yet, nothing is asked). */
export function useAllWork(query: AllWorkQuery | null) {
  return useQuery({
    queryKey: ALL_WORK_KEYS.view(query),
    queryFn: () => call(api.GET('/api/all-work', { params: { query: query ?? {} } })),
    enabled: query !== null,
    staleTime: Infinity,
    refetchOnWindowFocus: false,
    placeholderData: keepPreviousData,
  });
}

/**
 * The filter bar's choices and the column catalogue for a year; for null, those of the current
 * year, which the answer also files under its year.
 */
export function useAllWorkOptions(year: number | null, enabled = true) {
  const queryClient = useQueryClient();
  return useQuery({
    queryKey: ALL_WORK_KEYS.options(year),
    queryFn: () => fetchOptions(queryClient, year),
    enabled,
    staleTime: Infinity,
    refetchOnWindowFocus: false,
    placeholderData: keepPreviousData,
  });
}

/** The choices of a year, from the cache when they were asked before (a saved view's year). */
export function fetchOptionsOf(queryClient: QueryClient, year: number) {
  return queryClient.fetchQuery({
    queryKey: ALL_WORK_KEYS.options(year),
    queryFn: () => fetchOptions(queryClient, year),
    staleTime: Infinity,
  });
}

async function fetchOptions(queryClient: QueryClient, year: number | null) {
  const options = await call(
    api.GET('/api/all-work/options', { params: { query: year === null ? {} : { year } } }),
  );
  if (year === null) {
    queryClient.setQueryData(ALL_WORK_KEYS.options(options.year), options);
  }
  return options;
}

/** The person's remembered columns and sort (the defaults when never changed). */
export function useAllWorkPreferences() {
  return useQuery({
    queryKey: ALL_WORK_KEYS.preferences,
    queryFn: () => call(api.GET('/api/all-work/preferences')),
    staleTime: Infinity,
  });
}

/** How long column changes wait before they are remembered (plan "State"). */
const PREFERENCES_DELAY_MS = 500;

/**
 * Remembers the columns, their order and the sort (FR-021): 500 ms after the last change, and at
 * once when the page closes before that.
 */
export function usePreferencesSaver(): (preferences: AllWorkPreferences) => void {
  const queryClient = useQueryClient();
  const { show } = useToast();
  const pending = useRef<AllWorkPreferences | null>(null);
  const timer = useRef<number | undefined>(undefined);
  const { mutate } = useMutation({
    mutationFn: (body: AllWorkPreferences) => call(api.PUT('/api/all-work/preferences', { body })),
    onSuccess: (saved) => queryClient.setQueryData(ALL_WORK_KEYS.preferences, saved),
    onError: (error) => {
      if (error instanceof ApiError && error.kind !== 'signInRequired') {
        show({ message: error.message });
      }
    },
  });

  const flush = useCallback(() => {
    window.clearTimeout(timer.current);
    if (pending.current) {
      mutate(pending.current);
      pending.current = null;
    }
  }, [mutate]);

  useEffect(() => flush, [flush]);

  return useCallback(
    (preferences: AllWorkPreferences) => {
      pending.current = preferences;
      window.clearTimeout(timer.current);
      timer.current = window.setTimeout(flush, PREFERENCES_DELAY_MS);
    },
    [flush],
  );
}

/** The person's saved views, by name (FR-030). */
export function useSavedViews() {
  return useQuery({
    queryKey: ALL_WORK_KEYS.views,
    queryFn: () => call(api.GET('/api/saved-views')),
  });
}

/**
 * After a change the list is asked again — also when it failed, because a refused change may mean
 * the view was deleted in another tab. A saved view is written into the list at once, so "Modified"
 * does not flicker while the list reloads.
 */
function useViewsRefresh() {
  const queryClient = useQueryClient();
  return {
    remember: (view: SavedView) =>
      queryClient.setQueryData<SavedViewPage>(ALL_WORK_KEYS.views, (page) =>
        page
          ? {
              ...page,
              items: page.items.some((item) => item.id === view.id)
                ? page.items.map((item) => (item.id === view.id ? view : item))
                : [...page.items, view],
            }
          : page,
      ),
    refresh: () => queryClient.invalidateQueries({ queryKey: ALL_WORK_KEYS.views }),
  };
}

export function useCreateView() {
  const { remember, refresh } = useViewsRefresh();
  return useMutation({
    mutationFn: (body: SavedViewCreate) => call(api.POST('/api/saved-views', { body })),
    onSuccess: remember,
    onSettled: refresh,
  });
}

export function useUpdateView() {
  const { remember, refresh } = useViewsRefresh();
  return useMutation({
    mutationFn: ({ viewId, body }: { viewId: string; body: SavedViewUpdate }) =>
      call(api.PATCH('/api/saved-views/{viewId}', { params: { path: { viewId } }, body })),
    onSuccess: remember,
    onSettled: refresh,
  });
}

/** Deletes a view; the answer is the deleted view, so Undo can save it again (FR-031). */
export function useDeleteView() {
  const { refresh } = useViewsRefresh();
  return useMutation({
    mutationFn: (viewId: string) =>
      call(api.DELETE('/api/saved-views/{viewId}', { params: { path: { viewId } } })),
    onSettled: refresh,
  });
}

/** Whether a row's record still exists, before it opens (D-123). */
export function locateRecord(recordId: string) {
  return call(api.GET('/api/all-work/records/{recordId}', { params: { path: { recordId } } }));
}
