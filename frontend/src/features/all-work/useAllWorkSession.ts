import { useQueryClient } from '@tanstack/react-query';
import { useCallback, useEffect } from 'react';
import { useLocation, useNavigate } from 'react-router';
import { useCurrentUser } from '@/session/useSession';
import { useToast } from '@/ui/Toast/useToast';
import {
  fetchOptionsOf,
  useAllWorkOptions,
  useAllWorkPreferences,
  useSavedViews,
} from './allWorkApi';
import { useAllWorkState } from './allWorkStore';
import type { AllWorkEntry } from './allWorkStore';
import { leftOutMessage } from './allWorkText';
import type { SavedView, ViewState } from './allWorkTypes';
import { fromAddress, hasAddressFilters, leftOutOf, standardState, yearRange } from './viewState';
import type { PageSize } from './viewState';

/** A new state, or a change of the state as it is when the change applies. */
export type StateChange = ViewState | ((current: ViewState) => ViewState);

export interface AllWorkSession {
  /** null until the visit has chosen its view. */
  state: ViewState | null;
  entry: AllWorkEntry;
  /**
   * Shows another state, or a change of the state as it is then; the page goes back to 1 unless
   * `keepPage`.
   */
  setState: (next: StateChange, options?: { keepPage?: boolean }) => void;
  /**
   * Applies a saved view (FR-031): what no longer exists is left out and named in a message;
   * the view is then the active one (FR-030).
   */
  applyView: (view: SavedView) => Promise<void>;
  /** Marks a view as the active one without changing what is shown (after saving it). */
  setAppliedView: (viewId: string | null) => void;
  /** Back to the standard view: the current year, no filters, the remembered columns. */
  applyStandard: () => void;
  setPage: (page: number) => void;
  setPageSize: (pageSize: PageSize) => void;
  /** Remembers the scroll position before a record opens (FR-016). */
  rememberScroll: () => void;
  /** Why the visit could not start (preferences, choices or saved views failed to load). */
  startError: unknown;
  retryStart: () => void;
}

/**
 * The person's All work visit (FR-009, D-122): it starts from the person's default saved view,
 * else the standard view — the current year with the remembered columns — or from the filters in
 * the address, which it then removes; afterwards it keeps the state while the person moves around
 * MAPP.
 */
export function useAllWorkSession(): AllWorkSession {
  const user = useCurrentUser();
  const [entry, update] = useAllWorkState(user.id);
  const location = useLocation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { show } = useToast();
  const fromLink = hasAddressFilters(new URLSearchParams(location.search));
  const starting = !entry.initialized || fromLink;
  const preferences = useAllWorkPreferences();
  // The choices of the current year, which also say which year that is.
  const current = useAllWorkOptions(null, starting);
  const views = useSavedViews();

  /** A saved view's state without what no longer exists, and the names of what was left out. */
  const fitted = useCallback(
    async (view: SavedView) => {
      try {
        return leftOutOf(
          view.state,
          await fetchOptionsOf(queryClient, view.state.year, yearRange(view.state).to),
        );
      } catch {
        // The choices could not be read: the API leaves unknown values out itself.
        return { state: view.state, names: [] };
      }
    },
    [queryClient],
  );

  useEffect(() => {
    if (!starting || !preferences.data || !current.data || !views.data) {
      return;
    }
    const standard = standardState(preferences.data, current.data.currentYear);
    const params = new URLSearchParams(location.search);
    const start = { initialized: true, page: 1, scrollY: 0 };
    if (hasAddressFilters(params)) {
      update({ ...start, state: fromAddress(params, standard), appliedViewId: null });
      // The filters were applied once; the address goes back to plain /all-work (D-122).
      void navigate(location.pathname, { replace: true });
      return;
    }
    const defaultView = views.data.items.find((view) => view.isDefault);
    if (!defaultView) {
      update({ ...start, state: standard, appliedViewId: null });
      return;
    }
    // A new visit starts from the person's default view (FR-009).
    let cancelled = false;
    void fitted(defaultView).then(({ state, names }) => {
      if (cancelled) return;
      update({ ...start, state, appliedViewId: defaultView.id });
      if (names.length > 0) show({ message: leftOutMessage(names) });
    });
    return () => {
      cancelled = true;
    };
  }, [
    starting,
    preferences.data,
    current.data,
    views.data,
    location.search,
    location.pathname,
    navigate,
    update,
    fitted,
    show,
  ]);

  const setState = useCallback(
    (next: StateChange, { keepPage = false }: { keepPage?: boolean } = {}) =>
      update((entryNow) => ({
        state:
          typeof next === 'function' && entryNow.state ? next(entryNow.state) : (next as ViewState),
        ...(keepPage ? {} : { page: 1 }),
      })),
    [update],
  );

  const applyView = useCallback(
    async (view: SavedView) => {
      const { state, names } = await fitted(view);
      update({ state, appliedViewId: view.id, page: 1 });
      if (names.length > 0) show({ message: leftOutMessage(names) });
    },
    [fitted, update, show],
  );

  const startError = starting ? (preferences.error ?? current.error ?? views.error) : null;

  return {
    state: starting ? null : entry.state,
    entry,
    setState,
    applyView,
    setAppliedView: useCallback(
      (viewId: string | null) => update({ appliedViewId: viewId }),
      [update],
    ),
    applyStandard: () => {
      if (preferences.data && entry.state) {
        const year = current.data?.currentYear ?? entry.state.year;
        update({ state: standardState(preferences.data, year), appliedViewId: null, page: 1 });
      }
    },
    setPage: useCallback((page: number) => update({ page }), [update]),
    setPageSize: useCallback((pageSize: PageSize) => update({ pageSize, page: 1 }), [update]),
    rememberScroll: useCallback(() => update({ scrollY: window.scrollY }), [update]),
    startError,
    retryStart: () => {
      void preferences.refetch();
      void current.refetch();
      void views.refetch();
    },
  };
}
