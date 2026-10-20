/**
 * What the activity log and the History sections ask the API (feature 011, plan "Query keys"):
 * the summary cards' counts with the filters' choices, and the entries, a page at a time.
 */
import { useInfiniteQuery, useQuery } from '@tanstack/react-query';
import type { QueryKey } from '@tanstack/react-query';
import { api, call } from '@/api/client';
import type { components } from '@/api/schema';
import type { LogApiQuery } from './logFilters';

type ActivityLogEntry = components['schemas']['ActivityLogEntry'];
type ActivityLogPage = components['schemas']['ActivityLogPage'];

/** How many entries the log page loads at a time (FR-009). */
export const LOG_PAGE_SIZE = 50;

/** How many entries a History section loads at a time (FR-011). */
export const HISTORY_PAGE_SIZE = 20;

/**
 * The four cards' counts and the filters' choices (FR-002), for the person's local today and
 * week. Asked again when the page regains focus, so the counts stay current.
 */
export function useLogSummary(todayStart: string, weekStart: string) {
  return useQuery({
    queryKey: ['activity-log', 'summary', todayStart, weekStart],
    queryFn: () =>
      call(api.GET('/api/activity-log/summary', { params: { query: { todayStart, weekStart } } })),
    refetchOnWindowFocus: true,
  });
}

/** The entries loaded so far, newest first, and how to load more. */
export interface LoadedLog {
  entries: ActivityLogEntry[];
  /** The first page's answer: totals and the record of the Record filter. */
  firstPage: ActivityLogPage | undefined;
  isPending: boolean;
  error: Error | null;
  refetch: () => void;
  hasMore: boolean;
  loadMore: () => void;
  isLoadingMore: boolean;
}

/**
 * The entries matching `query`, `pageSize` at a time: **Load more** asks for the next page with
 * `beforeSeq` = the last entry's `seq` (FR-009). Nothing is asked while `query` is null (a custom
 * range that ends before it starts).
 */
export function useLogEntries(
  queryKey: QueryKey,
  query: LogApiQuery | null,
  pageSize: number,
): LoadedLog {
  const result = useInfiniteQuery({
    queryKey,
    enabled: query !== null,
    initialPageParam: undefined as number | undefined,
    queryFn: ({ pageParam, signal }) =>
      call(
        api.GET('/api/activity-log', {
          params: { query: { ...query, limit: pageSize, beforeSeq: pageParam } },
          signal,
        }),
      ),
    getNextPageParam: (lastPage, pages) => {
      const loaded = pages.reduce((count, page) => count + page.items.length, 0);
      const last = lastPage.items[lastPage.items.length - 1];
      return last && loaded < pages[0].total ? last.seq : undefined;
    },
  });

  return {
    entries: result.data?.pages.flatMap((page) => page.items) ?? [],
    firstPage: result.data?.pages[0],
    isPending: query !== null && result.isPending,
    error: result.error,
    refetch: () => void result.refetch(),
    hasMore: result.hasNextPage,
    loadMore: () => void result.fetchNextPage(),
    isLoadingMore: result.isFetchingNextPage,
  };
}

/** The query key of the log page's entries for an API query (plan "Query keys"). */
export function logEntriesKey(query: LogApiQuery | null): QueryKey {
  return ['activity-log', 'entries', query];
}

/** The query key of a History section's entries. */
export function historyKey(recordId: string, includeInside: boolean): QueryKey {
  return ['history', recordId, includeInside];
}
