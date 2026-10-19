/**
 * The Plans list's filters and layout (plan "Plans list", FR-003): read from the address, else from
 * what this person chose last time on this browser, else the defaults (All, All years, All plans,
 * Cards). The browser's storage may be missing or full; then the list simply uses the defaults.
 */
import type { PlanStatus } from './planText';

export interface PlanListFilters {
  status: 'all' | PlanStatus;
  /** null: All years. */
  year: number | null;
  scope: 'all' | 'myTeams';
  /** The search text (never remembered: it belongs to one visit). */
  q: string;
  layout: 'cards' | 'table';
}

export const DEFAULT_FILTERS: PlanListFilters = {
  status: 'all',
  year: null,
  scope: 'all',
  q: '',
  layout: 'cards',
};

/** The address parameters of the list; any of them in the address means the address decides. */
const KEYS = ['status', 'year', 'scope', 'q', 'layout'] as const;

function storageKey(personId: string): string {
  return `northstar.plans.list.${personId}`;
}

/** Reads one filter set from loose values; anything invalid becomes its default. */
function parse(values: Partial<Record<(typeof KEYS)[number], unknown>>): PlanListFilters {
  const { status, year, scope, q, layout } = values;
  const yearNumber = typeof year === 'number' ? year : Number(year);
  return {
    status: status === 'draft' || status === 'published' ? status : 'all',
    year:
      (typeof year === 'number' || (typeof year === 'string' && /^\d{4}$/.test(year))) &&
      Number.isInteger(yearNumber)
        ? yearNumber
        : null,
    scope: scope === 'myTeams' ? 'myTeams' : 'all',
    q: typeof q === 'string' ? q : '',
    layout: layout === 'table' ? 'table' : 'cards',
  };
}

/**
 * The filters to show: the address's when it names any, otherwise the remembered ones of this
 * person (without a search), otherwise the defaults.
 */
export function readFilters(search: URLSearchParams, personId: string): PlanListFilters {
  if (KEYS.some((key) => search.has(key))) {
    return parse(Object.fromEntries(KEYS.map((key) => [key, search.get(key) ?? undefined])));
  }
  return { ...readRemembered(personId), q: '' };
}

/** The address parameters of a filter set: only the values that differ from the defaults. */
export function toSearchParams(filters: PlanListFilters): URLSearchParams {
  const params = new URLSearchParams();
  if (filters.status !== 'all') params.set('status', filters.status);
  if (filters.year !== null) params.set('year', String(filters.year));
  if (filters.scope !== 'all') params.set('scope', filters.scope);
  if (filters.q.trim() !== '') params.set('q', filters.q);
  if (filters.layout !== 'cards') params.set('layout', filters.layout);
  return params;
}

/** What this person chose last time; the defaults when nothing (valid) is stored. */
export function readRemembered(personId: string): PlanListFilters {
  try {
    const stored = window.localStorage.getItem(storageKey(personId));
    const values: unknown = stored ? JSON.parse(stored) : null;
    return typeof values === 'object' && values !== null
      ? { ...parse(values), q: '' }
      : DEFAULT_FILTERS;
  } catch {
    return DEFAULT_FILTERS;
  }
}

/** Remembers the filters and layout (not the search) for this person; failures are ignored. */
export function remember(personId: string, filters: PlanListFilters): void {
  try {
    const { status, year, scope, layout } = filters;
    window.localStorage.setItem(
      storageKey(personId),
      JSON.stringify({ status, year, scope, layout }),
    );
  } catch {
    // Storage is full or blocked: the list keeps working, it just won't remember.
  }
}
