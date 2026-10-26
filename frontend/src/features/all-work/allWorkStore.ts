/**
 * All work's session state, per person (plan "State", D-122): the view, filters, search and
 * expansion, the applied saved view, the page and the scroll position. It lives in memory while
 * the browser tab lives, so it survives moving around MAPP (FR-009) and coming back from a
 * record (FR-016). A reload, or a new sign-in, is a new visit: signing out clears the web app's
 * cached data, and with it the marker this store checks.
 */
import { useQueryClient } from '@tanstack/react-query';
import type { QueryClient } from '@tanstack/react-query';
import { useCallback, useSyncExternalStore } from 'react';
import type { ViewState } from './allWorkTypes';
import type { PageSize } from './viewState';

export interface AllWorkEntry {
  /** False until the first visit has chosen the default saved view or the standard view. */
  initialized: boolean;
  state: ViewState | null;
  /** The saved view applied last (FR-030); null for the standard view or a changed address. */
  appliedViewId: string | null;
  page: number;
  pageSize: PageSize;
  /** The page's scroll position when a record was opened (FR-016). */
  scrollY: number;
  /** The visit this entry belongs to (see visitOf). */
  visit?: object;
}

const NEW_VISIT: AllWorkEntry = {
  initialized: false,
  state: null,
  appliedViewId: null,
  page: 1,
  pageSize: 50,
  scrollY: 0,
};

const entries = new Map<string, AllWorkEntry>();
const listeners = new Set<() => void>();

function subscribe(listener: () => void): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

function notify() {
  listeners.forEach((listener) => listener());
}

/** The query-cache key of a person's visit marker. Signing out removes every cached answer. */
const VISIT_KEY = 'all-work-visit';

function visitOf(queryClient: QueryClient, personId: string): object | undefined {
  return queryClient.getQueryData<object>([VISIT_KEY, personId]);
}

/** The person's entry, or a new visit when the entry belongs to an earlier sign-in. */
function entryOf(queryClient: QueryClient, personId: string): AllWorkEntry {
  const entry = entries.get(personId);
  return entry && entry.visit !== undefined && entry.visit === visitOf(queryClient, personId)
    ? entry
    : NEW_VISIT;
}

export type EntryChange = Partial<AllWorkEntry> | ((entry: AllWorkEntry) => Partial<AllWorkEntry>);

/**
 * The person's All work entry and a function that changes it. Every page visit of the same
 * sign-in sees the same entry; another person has their own.
 */
export function useAllWorkState(personId: string): [AllWorkEntry, (change: EntryChange) => void] {
  const queryClient = useQueryClient();
  const stored = useSyncExternalStore(subscribe, () => entries.get(personId) ?? NEW_VISIT);
  const entry =
    stored.visit !== undefined && stored.visit === visitOf(queryClient, personId)
      ? stored
      : NEW_VISIT;

  const update = useCallback(
    (change: EntryChange) => {
      const current = entryOf(queryClient, personId);
      const next: AllWorkEntry = {
        ...current,
        ...(typeof change === 'function' ? change(current) : change),
      };
      if (next.visit === undefined) {
        // A new visit: its marker stays in the cache until the person signs out.
        const visit = {};
        queryClient.setQueryDefaults([VISIT_KEY], { gcTime: Infinity });
        queryClient.setQueryData([VISIT_KEY, personId], visit);
        next.visit = visit;
      }
      entries.set(personId, next);
      notify();
    },
    [queryClient, personId],
  );

  return [entry, update];
}

/** Forgets every entry (tests). */
export function resetAllWorkStore(): void {
  entries.clear();
  notify();
}
