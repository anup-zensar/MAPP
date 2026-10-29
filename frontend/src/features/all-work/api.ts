// A view is read once per filter set and kept: other people's changes appear on the next filter or
// view change, or through Refresh. While a new answer loads, the previous one stays on screen.
import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { QueryClient } from '@tanstack/react-query';
import { useCallback, useEffect, useRef } from 'react';
import { api, call } from '@/api/client';
import { downloadFromApi } from '@/api/download';
import { ApiError } from '@/api/errors';
import { useToast } from '@/ui/Toast/useToast';
import type {
  AllWorkPreferences,
  BreakdownChoice,
  AllWorkQuery,
  SavedView,
  SavedViewCreate,
  SavedViewPage,
  SavedViewUpdate,
  ViewState,
} from './all-work-types';
import { toQueryParams } from './view-state';

export const ALL_WORK_KEYS = {
  all: ['all-work'] as const,
  view: (query: AllWorkQuery | null) => ['all-work', query] as const,
  // null: the current year, before the first visit knows it; one year keeps its key from before ranges.
  options: (year: number | null, yearTo?: number) =>
    yearTo !== undefined && year !== null && yearTo > year
      ? (['all-work-options', year, yearTo] as const)
      : (['all-work-options', year] as const),
  preferences: ['all-work-preferences'] as const,
  views: ['saved-views'] as const,
};

export function useAllWork(query: AllWorkQuery | null) {
  return useQuery({
    queryKey: ALL_WORK_KEYS.view(query),
    queryFn: () => call(api.GET('/api/v1/all-work', { params: { query: query ?? {} } })),
    enabled: query !== null,
    staleTime: Infinity,
    refetchOnWindowFocus: false,
    placeholderData: keepPreviousData,
  });
}

export function useAllWorkOptions(year: number | null, enabled = true, yearTo?: number) {
  const queryClient = useQueryClient();
  return useQuery({
    queryKey: ALL_WORK_KEYS.options(year, yearTo),
    queryFn: () => fetchOptions(queryClient, year, yearTo),
    enabled,
    staleTime: Infinity,
    refetchOnWindowFocus: false,
    placeholderData: keepPreviousData,
  });
}

export function fetchOptionsOf(queryClient: QueryClient, year: number, yearTo?: number) {
  return queryClient.fetchQuery({
    queryKey: ALL_WORK_KEYS.options(year, yearTo),
    queryFn: () => fetchOptions(queryClient, year, yearTo),
    staleTime: Infinity,
  });
}

async function fetchOptions(queryClient: QueryClient, year: number | null, yearTo?: number) {
  const query =
    year === null
      ? {}
      : yearTo !== undefined && yearTo > year
        ? { yearFrom: year, yearTo }
        : { year };
  const options = await call(api.GET('/api/v1/all-work/options', { params: { query } }));
  if (year === null) {
    queryClient.setQueryData(ALL_WORK_KEYS.options(options.year), options);
  }
  return options;
}

export function useAllWorkPreferences() {
  return useQuery({
    queryKey: ALL_WORK_KEYS.preferences,
    queryFn: () => call(api.GET('/api/v1/all-work/preferences')),
    staleTime: Infinity,
  });
}

const PREFERENCES_DELAY_MS = 500;

/** Remembers the columns and sort 500 ms after the last change, or at once when the page closes. */
export function usePreferencesSaver(): (preferences: AllWorkPreferences) => void {
  const queryClient = useQueryClient();
  const { show } = useToast();
  const pending = useRef<AllWorkPreferences | null>(null);
  const timer = useRef<number | undefined>(undefined);
  const { mutate } = useMutation({
    mutationFn: (body: AllWorkPreferences) =>
      call(api.PUT('/api/v1/all-work/preferences', { body })),
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

export function useSavedViews() {
  return useQuery({
    queryKey: ALL_WORK_KEYS.views,
    queryFn: () => call(api.GET('/api/v1/saved-views')),
  });
}

// The list is asked again after a change, also a failed one (the view may be gone). A saved view is
// written into the list at once, so "Modified" does not flicker while it reloads.
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
    mutationFn: (body: SavedViewCreate) => call(api.POST('/api/v1/saved-views', { body })),
    onSuccess: remember,
    onSettled: refresh,
  });
}

export function useUpdateView() {
  const { remember, refresh } = useViewsRefresh();
  return useMutation({
    mutationFn: ({ viewId, body }: { viewId: string; body: SavedViewUpdate }) =>
      call(api.PATCH('/api/v1/saved-views/{viewId}', { params: { path: { viewId } }, body })),
    onSuccess: remember,
    onSettled: refresh,
  });
}

/** Deletes a view; the answer is the deleted view, so Undo can save it again. */
export function useDeleteView() {
  const { refresh } = useViewsRefresh();
  return useMutation({
    mutationFn: (viewId: string) =>
      call(api.DELETE('/api/v1/saved-views/{viewId}', { params: { path: { viewId } } })),
    onSettled: refresh,
  });
}

export function locateRecord(recordId: string) {
  return call(api.GET('/api/v1/all-work/records/{recordId}', { params: { path: { recordId } } }));
}

export function exportAllWork(state: ViewState, format: 'xlsx' | 'csv'): Promise<string> {
  return downloadFromApi(
    api.GET('/api/v1/all-work/export', {
      params: {
        query: {
          ...toQueryParams(state),
          ...(state.view === 'breakdown' && state.breakdown && breakdownParams(state.breakdown)),
          format,
          unassigned: state.showUnassigned,
        },
      },
      parseAs: 'blob',
    }),
    `SP&A One All work.${format}`,
  );
}

function breakdownParams(choice: BreakdownChoice) {
  return { by: choice.by, ...(choice.thenBy && { thenBy: choice.thenBy }) };
}
