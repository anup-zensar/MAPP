/**
 * The Plans list's filters and layout (plan "Plans list", FR-003; 016 FR-033): read from the
 * address, else from what this person chose last time on this browser, else the defaults (All,
 * All years, every plan type, All plans, Cards). The browser's storage may be missing or full;
 * then the list simply uses the defaults. A single `year` — in an old address or in what was
 * remembered before 016 — reads as a one-year range (D-269).
 */
import { PLAN_TYPES } from './planCurrency';
import type { PlanType } from './planCurrency';
import type { PlanStatus } from './planText';

export interface PlanListFilters {
  status: 'all' | PlanStatus;
  /** The first year of the range; null with `yearTo`: All years. */
  yearFrom: number | null;
  /** The last year of the range (never before `yearFrom`); null with `yearFrom`: All years. */
  yearTo: number | null;
  /** Only plans of these types; empty: every type (016 FR-033). */
  types: PlanType[];
  scope: 'all' | 'myTeams';
  /** The search text (never remembered: it belongs to one visit). */
  q: string;
  layout: 'cards' | 'table';
}

export const DEFAULT_FILTERS: PlanListFilters = {
  status: 'all',
  yearFrom: null,
  yearTo: null,
  types: [],
  scope: 'all',
  q: '',
  layout: 'cards',
};

/** The address parameters of the list; any of them in the address means the address decides. */
const KEYS = ['status', 'yearFrom', 'yearTo', 'year', 'type', 'scope', 'q', 'layout'] as const;

type LooseValues = Partial<Record<(typeof KEYS)[number], unknown>>;

function storageKey(personId: string): string {
  return `mapp.plans.list.${personId}`;
}

/** A four-digit year from a number or text; null for anything else. */
function yearOf(value: unknown): number | null {
  if (typeof value === 'number') {
    return Number.isInteger(value) && value >= 1000 && value <= 9999 ? value : null;
  }
  return typeof value === 'string' && /^\d{4}$/.test(value) ? Number(value) : null;
}

/** The year range from loose values: from–to in order, one year when only one end is valid. */
function rangeOf({
  yearFrom,
  yearTo,
  year,
}: LooseValues): Pick<PlanListFilters, 'yearFrom' | 'yearTo'> {
  const single = yearOf(year);
  const from = yearOf(yearFrom) ?? yearOf(yearTo) ?? single;
  const to = yearOf(yearTo) ?? yearOf(yearFrom) ?? single;
  if (from === null || to === null) {
    return { yearFrom: null, yearTo: null };
  }
  return { yearFrom: Math.min(from, to), yearTo: Math.max(from, to) };
}

/** The known plan types among loose values, in the order the filter offers them. */
function typesOf(value: unknown): PlanType[] {
  const values = Array.isArray(value) ? (value as unknown[]) : [value];
  return PLAN_TYPES.filter((type) => values.includes(type));
}

/** Reads one filter set from loose values; anything invalid becomes its default. */
function parse(values: LooseValues): PlanListFilters {
  const { status, type, scope, q, layout } = values;
  return {
    status: status === 'draft' || status === 'published' ? status : 'all',
    ...rangeOf(values),
    types: typesOf(type),
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

/** The address parameters of a filter set: only the values that differ from the defaults. */
export function toSearchParams(filters: PlanListFilters): URLSearchParams {
  const params = new URLSearchParams();
  if (filters.status !== 'all') params.set('status', filters.status);
  if (filters.yearFrom !== null && filters.yearTo !== null) {
    params.set('yearFrom', String(filters.yearFrom));
    params.set('yearTo', String(filters.yearTo));
  }
  for (const type of filters.types) params.append('type', type);
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
      ? { ...parse(values as LooseValues), q: '' }
      : DEFAULT_FILTERS;
  } catch {
    return DEFAULT_FILTERS;
  }
}

/** Remembers the filters and layout (not the search) for this person; failures are ignored. */
export function remember(personId: string, filters: PlanListFilters): void {
  try {
    const { status, yearFrom, yearTo, types, scope, layout } = filters;
    window.localStorage.setItem(
      storageKey(personId),
      JSON.stringify({ status, yearFrom, yearTo, type: types, scope, layout }),
    );
  } catch {
    // Storage is full or blocked: the list keeps working, it just won't remember.
  }
}

/** "2026" for one year, "2025 – 2026" for a range; null for All years. */
export function yearRangeLabel(
  filters: Pick<PlanListFilters, 'yearFrom' | 'yearTo'>,
): string | null {
  const { yearFrom, yearTo } = filters;
  if (yearFrom === null || yearTo === null) return null;
  return yearFrom === yearTo ? String(yearFrom) : `${yearFrom} – ${yearTo}`;
}
