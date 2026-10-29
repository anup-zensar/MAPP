import { FILTER_LABELS, MAX_YEARS } from './all-work-text';
import { attrParams, canonicalAttributes, knownAttributeFilters } from './attribute-filters';
import type {
  AllWorkOptions,
  AllWorkPreferences,
  AllWorkQuery,
  FilterValue,
  Filters,
  SavedView,
  ViewState,
} from './all-work-types';

export const NO_FILTERS: Filters = {
  plans: [],
  planStatuses: [],
  containerStatuses: [],
  activityStatuses: [],
  owners: [],
  teams: [],
  businessUnits: [],
  activityTypes: [],
  strategic: 'include',
  showLevels: 'everything',
};

export const PAGE_SIZES = [50, 100, 200] as const;
export type PageSize = (typeof PAGE_SIZES)[number];

export const ID_FILTERS = ['plans', 'owners', 'teams', 'businessUnits', 'activityTypes'] as const;
export type IdFilter = (typeof ID_FILTERS)[number];

export const VALUE_FILTERS = [
  'planStatuses',
  'containerStatuses',
  'activityStatuses',
  'planTypes',
  'planCurrencies',
  'finance',
] as const;
export type ValueFilter = (typeof VALUE_FILTERS)[number];

export function yearRange(state: Pick<ViewState, 'year' | 'yearTo'>): { from: number; to: number } {
  const to =
    state.yearTo !== undefined && state.yearTo > state.year
      ? Math.min(state.yearTo, state.year + MAX_YEARS - 1)
      : state.year;
  return { from: state.year, to };
}

export function withYears(state: ViewState, from: number, to: number): ViewState {
  const { yearFrom: _from, yearTo: _to, ...rest } = state;
  void _from;
  void _to;
  const range = to > from;
  return {
    ...rest,
    year: from,
    ...(range && { yearFrom: from, yearTo: to }),
    period: range ? 'year' : state.period === 'year' ? 'month' : state.period,
    filters: { ...state.filters, plans: [] },
  };
}

export function standardState(preferences: AllWorkPreferences, year: number): ViewState {
  return {
    view: 'grid',
    year,
    filters: NO_FILTERS,
    search: '',
    columns: [...preferences.columns],
    sort: preferences.sort,
    expansion: { mode: 'default', toggled: [] },
    showUnassigned: true,
    period: 'month',
    calendarLayout: 'timeline',
  };
}

export function toQueryParams(state: ViewState): AllWorkQuery {
  const { filters } = state;
  const { from, to } = yearRange(state);
  const query: AllWorkQuery =
    to > from ? { yearFrom: from, yearTo: to, view: state.view } : { year: from, view: state.view };
  const ids = (values: FilterValue[]) => values.map((value) => value.id);
  if (filters.plans.length > 0) query.plan = ids(filters.plans);
  if (filters.planStatuses.length > 0) query.planStatus = [...filters.planStatuses];
  if (filters.containerStatuses.length > 0) query.containerStatus = [...filters.containerStatuses];
  if (filters.activityStatuses.length > 0) query.activityStatus = [...filters.activityStatuses];
  if (filters.owners.length > 0) query.owner = ids(filters.owners);
  if (filters.teams.length > 0) query.team = ids(filters.teams);
  if (filters.businessUnits.length > 0) query.businessUnit = ids(filters.businessUnits);
  if (filters.activityTypes.length > 0) query.activityType = ids(filters.activityTypes);
  if (filters.strategic !== 'include') query.strategic = filters.strategic;
  if (filters.bcpir !== undefined && filters.bcpir !== 'include') query.bcpir = filters.bcpir;
  if (filters.showLevels !== 'everything') query.showLevels = filters.showLevels;
  if (filters.planTypes?.length) query.planType = [...filters.planTypes];
  if (filters.planCurrencies?.length) query.planCurrency = [...filters.planCurrencies];
  if (filters.finance?.length) query.finance = [...filters.finance];
  if (filters.attributes?.length) query.attr = attrParams(filters.attributes);
  const search = state.search.trim();
  if (search !== '') query.search = search;
  if (state.view === 'grid') query.column = [...state.columns];
  if (state.sort) query.sort = `${state.sort.column}:${state.sort.direction}`;
  if (state.view === 'budget') query.period = state.period;
  if (state.view === 'budget' && state.comparePhased) query.comparePhased = true;
  return query;
}

export function savedStateOf(state: ViewState): SavedView['state'] {
  const { finance: _finance, ...filters } = state.filters;
  void _finance;
  return { ...state, filters };
}

export function sameState(a: ViewState, b: ViewState): boolean {
  return canonical(a) === canonical(b);
}

function canonical(state: ViewState): string {
  const { filters } = state;
  const sortedIds = (values: FilterValue[]) => values.map((value) => value.id).sort();
  const sorted = (values: string[] | undefined) => [...(values ?? [])].sort();
  const { from, to } = yearRange(state);
  const { breakdown, ...rest } = state;
  return JSON.stringify(
    {
      ...rest,
      year: from,
      yearFrom: undefined,
      yearTo: to,
      comparePhased: Boolean(state.comparePhased),
      search: state.search.trim(),
      filters: {
        ...filters,
        plans: sortedIds(filters.plans),
        planStatuses: [...filters.planStatuses].sort(),
        containerStatuses: [...filters.containerStatuses].sort(),
        activityStatuses: [...filters.activityStatuses].sort(),
        owners: sortedIds(filters.owners),
        teams: sortedIds(filters.teams),
        businessUnits: sortedIds(filters.businessUnits),
        activityTypes: sortedIds(filters.activityTypes),
        planTypes: sorted(filters.planTypes),
        planCurrencies: sorted(filters.planCurrencies),
        finance: sorted(filters.finance),
        attributes: canonicalAttributes(filters.attributes),
      },
      breakdown: state.view === 'breakdown' ? (breakdown ?? null) : null,
      expansion: { mode: state.expansion.mode, toggled: [...state.expansion.toggled].sort() },
    },
    inKeyOrder,
  );
}

// Keys in one order, so a part present in one state and missing in the other never reorders the rest.
function inKeyOrder(_key: string, value: unknown): unknown {
  return value && typeof value === 'object' && !Array.isArray(value)
    ? Object.fromEntries(Object.entries(value).sort(([a], [b]) => a.localeCompare(b)))
    : value;
}

const PARAMETER_FILTERS = new Map<string, keyof Filters>([
  ['plan', 'plans'],
  ['planStatus', 'planStatuses'],
  ['containerStatus', 'containerStatuses'],
  ['activityStatus', 'activityStatuses'],
  ['owner', 'owners'],
  ['team', 'teams'],
  ['businessUnit', 'businessUnits'],
  ['activityType', 'activityTypes'],
  ['strategic', 'strategic'],
  ['bcpir', 'bcpir'],
  ['showLevels', 'showLevels'],
  ['planType', 'planTypes'],
  ['planCurrency', 'planCurrencies'],
  ['finance', 'finance'],
  ['attr', 'attributes'],
]);

type Reset = (state: ViewState, standard: ViewState) => ViewState;

function standardYear(state: ViewState, standard: ViewState): ViewState {
  const { yearFrom: _from, yearTo: _to, ...single } = state;
  void _from;
  void _to;
  return { ...single, year: standard.year };
}

const noBreakdown: Reset = (state) => ({ ...state, breakdown: null });

const RESETS: Record<string, Reset> = {
  year: standardYear,
  yearFrom: standardYear,
  yearTo: standardYear,
  comparePhased: (state) => ({ ...state, comparePhased: false }),
  search: (state) => ({ ...state, search: '' }),
  column: (state, standard) => ({ ...state, columns: standard.columns }),
  sort: (state) => ({ ...state, sort: null }),
  period: (state) => ({ ...state, period: 'month' }),
  view: (state) => ({ ...state, view: 'grid' }),
  by: noBreakdown,
  thenBy: noBreakdown,
};

export function withoutRefused(state: ViewState, fields: string[], standard: ViewState): ViewState {
  return fields.reduce((next, field) => {
    const filter = PARAMETER_FILTERS.get(field);
    if (filter) {
      return { ...next, filters: { ...next.filters, [filter]: NO_FILTERS[filter] } };
    }
    return RESETS[field]?.(next, standard) ?? next;
  }, state);
}

export function leftOutOf(
  state: ViewState,
  options: AllWorkOptions,
): { state: ViewState; names: string[] } {
  const names: string[] = [];
  const filters: Filters = { ...state.filters };
  for (const filter of ID_FILTERS) {
    const known = new Set(choicesOf(filter, options).map((choice) => choice.id));
    filters[filter] = state.filters[filter].filter((value) => {
      if (known.has(value.id)) return true;
      names.push(`${FILTER_LABELS[filter]}: ${value.label}`);
      return false;
    });
  }
  const columnKeys = new Set(
    options.columnGroups.flatMap((group) => group.columns.map((column) => column.key)),
  );
  const columns = state.columns.filter((key) => columnKeys.has(key));
  const droppedColumns = state.columns.length - columns.length;
  if (droppedColumns > 0) {
    names.push(droppedColumns === 1 ? '1 column' : `${droppedColumns} columns`);
  }
  let { sort } = state;
  if (sort && !columnKeys.has(sort.column)) {
    sort = null;
    names.push('the sort order');
  }
  const attributes = knownAttributeFilters(filters.attributes, options.attributes);
  names.push(...attributes.names);
  return {
    state: { ...state, filters: { ...filters, attributes: attributes.filters }, columns, sort },
    names,
  };
}

export function choicesOf(
  filter: IdFilter,
  options: AllWorkOptions | undefined,
): { id: string; label: string; inactive?: boolean }[] {
  return options ? options[filter] : [];
}
