import type { PlanType } from './plan-currency';
import type { PlanStatus } from './plan-text';
import { looseFiltersSchema } from './schemas';

export interface PlanListFilters {
  status: 'all' | PlanStatus;
  yearFrom: number | null;
  yearTo: number | null;
  types: PlanType[];
  scope: 'all' | 'myTeams';
  bcpir: 'include' | 'only' | 'exclude';
  q: string;
  layout: 'cards' | 'table';
}

export const DEFAULT_FILTERS: PlanListFilters = {
  status: 'all',
  yearFrom: null,
  yearTo: null,
  types: [],
  scope: 'all',
  bcpir: 'include',
  q: '',
  layout: 'cards',
};

const KEYS = [
  'status',
  'yearFrom',
  'yearTo',
  'year',
  'type',
  'scope',
  'bcpir',
  'q',
  'layout',
] as const;

function storageKey(personId: string): string {
  return `mapp.plans.list.${personId}`;
}

function parse(values: unknown): PlanListFilters {
  const parsed = looseFiltersSchema.safeParse(values);
  return parsed.success ? parsed.data : DEFAULT_FILTERS;
}

export function readFilters(search: URLSearchParams, personId: string): PlanListFilters {
  if (KEYS.some((key) => search.has(key))) {
    return parse(
      Object.fromEntries(
        KEYS.map((key) => [
          key,
          key === 'type' ? search.getAll(key) : (search.get(key) ?? undefined),
        ]),
      ),
    );
  }
  return { ...readRemembered(personId), q: '' };
}

export function toSearchParams(filters: PlanListFilters): URLSearchParams {
  const params = new URLSearchParams();
  if (filters.status !== 'all') params.set('status', filters.status);
  if (filters.yearFrom !== null && filters.yearTo !== null) {
    params.set('yearFrom', String(filters.yearFrom));
    params.set('yearTo', String(filters.yearTo));
  }
  for (const type of filters.types) params.append('type', type);
  if (filters.scope !== 'all') params.set('scope', filters.scope);
  if (filters.bcpir !== 'include') params.set('bcpir', filters.bcpir);
  if (filters.q.trim() !== '') params.set('q', filters.q);
  if (filters.layout !== 'cards') params.set('layout', filters.layout);
  return params;
}

export function readRemembered(personId: string): PlanListFilters {
  try {
    const stored = window.localStorage.getItem(storageKey(personId));
    return stored ? { ...parse(JSON.parse(stored)), q: '' } : DEFAULT_FILTERS;
  } catch {
    return DEFAULT_FILTERS;
  }
}

export function remember(personId: string, filters: PlanListFilters): void {
  try {
    const { status, yearFrom, yearTo, types, scope, bcpir, layout } = filters;
    window.localStorage.setItem(
      storageKey(personId),
      JSON.stringify({ status, yearFrom, yearTo, type: types, scope, bcpir, layout }),
    );
  } catch {
    // Storage is full or blocked: the list keeps working, it just won't remember.
  }
}

export function yearRangeLabel(
  filters: Pick<PlanListFilters, 'yearFrom' | 'yearTo'>,
): string | null {
  const { yearFrom, yearTo } = filters;
  if (yearFrom === null || yearTo === null) return null;
  return yearFrom === yearTo ? String(yearFrom) : `${yearFrom} – ${yearTo}`;
}
