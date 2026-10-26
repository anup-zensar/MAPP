import { useSearchParams } from 'react-router';
import { z } from 'zod';

const statusSchema = z.enum(['active', 'inactive', 'all']).catch('active');

/** The status filter of a Settings list: Active (the default), Inactive or All. */
export type StatusFilter = z.infer<typeof statusSchema>;

/** The status filter in the address (`?status=`); Active for anything else. */
export function statusFilterOf(raw: string | null): StatusFilter {
  return statusSchema.parse(raw);
}

export interface ListFilters {
  /** What is typed in the search box. */
  q: string;
  status: StatusFilter;
  setQ: (q: string) => void;
  setStatus: (status: StatusFilter) => void;
  /** "Clear filters": no search, every status. */
  clear: () => void;
  /** The record whose details are open (`?id=`); "" when none. */
  openId: string;
  /** Opens a record's details, or closes them with "". */
  setOpenId: (id: string) => void;
}

/**
 * The search, status filter and open record of a Settings list, kept in the page address (`q`,
 * `status`, `id`), so Back and shared links keep them (feature 002 plan, "Routes and navigation").
 */
export function useListFilters(): ListFilters {
  const [params, setParams] = useSearchParams();
  const status = statusFilterOf(params.get('status'));

  function update(changes: Record<string, string | null>, replace = true) {
    setParams(
      (previous) => {
        const next = new URLSearchParams(previous);
        for (const [key, value] of Object.entries(changes)) {
          if (value === null || value === '') {
            next.delete(key);
          } else {
            next.set(key, value);
          }
        }
        return next;
      },
      { replace },
    );
  }

  return {
    q: params.get('q') ?? '',
    status,
    setQ: (q) => update({ q }),
    setStatus: (next) => update({ status: next === 'active' ? null : next }),
    clear: () => update({ q: null, status: 'all' }),
    openId: params.get('id') ?? '',
    // Opening details is a step people can go Back from; closing them replaces it.
    setOpenId: (id) => update({ id }, id === ''),
  };
}

/** The records a Settings list shows, and how many each status choice would show. */
export interface FilteredRecords<T> {
  shown: T[];
  counts: Record<StatusFilter, number>;
}

/**
 * Applies the search (any part of `searchText`, ignoring letter case) and the status filter. The
 * counts are those of the search alone, so each status choice says what it would show.
 */
export function filterRecords<T extends { active: boolean }>(
  records: T[],
  { q, status }: Pick<ListFilters, 'q' | 'status'>,
  searchText: (record: T) => string,
): FilteredRecords<T> {
  const query = q.trim().toLowerCase();
  const found = query
    ? records.filter((record) => searchText(record).toLowerCase().includes(query))
    : records;
  const active = found.filter((record) => record.active);
  const inactive = found.filter((record) => !record.active);
  return {
    shown: status === 'active' ? active : status === 'inactive' ? inactive : found,
    counts: { active: active.length, inactive: inactive.length, all: found.length },
  };
}
