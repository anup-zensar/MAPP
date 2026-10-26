/**
 * The All work state and what follows from it, as pure functions (plan "State", data-model §1.2):
 * the standard view, the years, the API query, "Modified" (FR-030), the parts the API refused and
 * the parts of a saved view that no longer exist (FR-031). schemas.ts reads the address (D-122).
 */
import { FILTER_LABELS, MAX_YEARS } from './all-work-text';
import type {
  AllWorkOptions,
  AllWorkPreferences,
  AllWorkQuery,
  FilterValue,
  Filters,
  SavedView,
  ViewState,
} from './all-work-types';

/** No filter in use (data-model §1.3): empty lists, Include and Everything. */
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

/** Rows per page (D-115). */
export const PAGE_SIZES = [50, 100, 200] as const;
export type PageSize = (typeof PAGE_SIZES)[number];

/** The filters that hold IDs with the name they had when chosen. */
export const ID_FILTERS = ['plans', 'owners', 'teams', 'businessUnits', 'activityTypes'] as const;
export type IdFilter = (typeof ID_FILTERS)[number];

/** The filters that hold fixed values (statuses, plan types, currencies, Financials). */
export const VALUE_FILTERS = [
  'planStatuses',
  'containerStatuses',
  'activityStatuses',
  'planTypes',
  'planCurrencies',
  'finance',
] as const;

/**
 * The years a state shows (016 FR-034): from `year` to `yearTo`, or the one year of a state saved
 * before ranges existed (FR-043: an old year reads as a one-year range).
 */
export function yearRange(state: Pick<ViewState, 'year' | 'yearTo'>): { from: number; to: number } {
  const to =
    state.yearTo !== undefined && state.yearTo > state.year
      ? Math.min(state.yearTo, state.year + MAX_YEARS - 1)
      : state.year;
  return { from: state.year, to };
}

/**
 * The state for the years `from` to `to` (016 FR-034, FR-035): a one-year range keeps the plain
 * year (as before 016, so older servers understand it); a longer one opens Budget & cost on
 * Yearly. Chosen plans go, as with a new year: the plans of other years cannot be chosen.
 */
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

/**
 * The standard view (FR-009): the Grid of the year without filters, with the person's remembered
 * columns and sort, plans, campaigns and initiatives open, unassigned rows shown, by month.
 */
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

/**
 * The query of GET /api/v1/all-work for a state: only what is in use. Columns go with the Grid (the
 * other views have fixed columns), the period with Budget & cost.
 */
export function toQueryParams(state: ViewState): AllWorkQuery {
  const { filters } = state;
  const { from, to } = yearRange(state);
  // One year goes as `year`, as before 016; a range as yearFrom and yearTo (016 FR-034).
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
  if (filters.showLevels !== 'everything') query.showLevels = filters.showLevels;
  if (filters.planTypes?.length) query.planType = [...filters.planTypes];
  if (filters.planCurrencies?.length) query.planCurrency = [...filters.planCurrencies];
  if (filters.finance?.length) query.finance = [...filters.finance];
  const search = state.search.trim();
  if (search !== '') query.search = search;
  if (state.view === 'grid') query.column = [...state.columns];
  if (state.sort) query.sort = `${state.sort.column}:${state.sort.direction}`;
  if (state.view === 'budget') query.period = state.period;
  if (state.view === 'budget' && state.comparePhased) query.comparePhased = true;
  return query;
}

/**
 * The state a saved view keeps (data-model §1.2): the contract's filters only, so the Financials
 * filter of the session (015 FR-024, an address filter) is left out.
 */
export function savedStateOf(state: ViewState): SavedView['state'] {
  const { finance: _finance, ...filters } = state.filters;
  void _finance;
  return { ...state, filters };
}

/**
 * Whether two states show the same view (FR-030, "Modified"). The order of chosen filter values
 * and of opened rows does not matter, nor the names stored with IDs; the order of columns does.
 */
export function sameState(a: ViewState, b: ViewState): boolean {
  return canonical(a) === canonical(b);
}

function canonical(state: ViewState): string {
  const { filters } = state;
  const sortedIds = (values: FilterValue[]) => values.map((value) => value.id).sort();
  const sorted = (values: string[] | undefined) => [...(values ?? [])].sort();
  const { from, to } = yearRange(state);
  return JSON.stringify({
    ...state,
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
      attributes: filters.attributes ?? [],
    },
    expansion: { mode: state.expansion.mode, toggled: [...state.expansion.toggled].sort() },
  });
}

/** The filter of each repeatable or single filter parameter of the API. */
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
  ['showLevels', 'showLevels'],
  ['planType', 'planTypes'],
  ['planCurrency', 'planCurrencies'],
  ['finance', 'finance'],
]);

/**
 * The state with the parts the API refused set back to the standard view's (plan "States":
 * the offending filter is reset). `fields` are the API's parameter names.
 */
export function withoutRefused(state: ViewState, fields: string[], standard: ViewState): ViewState {
  let next = state;
  for (const field of fields) {
    const filter = PARAMETER_FILTERS.get(field);
    if (filter) {
      next = { ...next, filters: { ...next.filters, [filter]: NO_FILTERS[filter] } };
    } else if (field === 'year' || field === 'yearFrom' || field === 'yearTo') {
      // Back to the standard year, as one year (016 FR-034).
      const { yearFrom: _from, yearTo: _to, ...single } = next;
      void _from;
      void _to;
      next = { ...single, year: standard.year };
    } else if (field === 'comparePhased') {
      next = { ...next, comparePhased: false };
    } else if (field === 'search') {
      next = { ...next, search: '' };
    } else if (field === 'column') {
      next = { ...next, columns: standard.columns };
    } else if (field === 'sort') {
      next = { ...next, sort: null };
    } else if (field === 'period') {
      next = { ...next, period: 'month' };
    } else if (field === 'view') {
      next = { ...next, view: 'grid' };
    }
  }
  return next;
}

/**
 * A saved view's state without what no longer exists (FR-031): plans, people, teams, options and
 * activity types the filter bar no longer offers, and columns the column chooser no longer has.
 * `names` says what was left out, for "Some parts of this view no longer exist and were left out:
 * <list>."
 */
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
  return { state: { ...state, filters, columns, sort }, names };
}

/** The filter bar's choices of an ID filter. */
export function choicesOf(
  filter: IdFilter,
  options: AllWorkOptions | undefined,
): { id: string; label: string; inactive?: boolean }[] {
  return options ? options[filter] : [];
}
