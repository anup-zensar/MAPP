/**
 * The All work state and what follows from it, as pure functions (plan "State", data-model §1.2,
 * §4): the standard view, the API query, the address filters of other pages (D-122), "Modified"
 * (FR-030), which rows are open and on screen (FR-011, FR-012, D-115), the filter chips (FR-008)
 * and the parts of a saved view that no longer exist (FR-031).
 *
 * The web app never adds money: every amount comes from GET /api/all-work (D-114). These
 * functions only choose which of the answered rows are on screen.
 */
import {
  ACTIVITY_STATUS_LABELS,
  CONTAINER_STATUS_LABELS,
  FILTER_LABELS,
  FINANCE_LABELS,
  MAX_YEARS,
  PLAN_STATUS_LABELS,
  PLAN_TYPE_LABELS,
  SHOW_LEVELS_LABELS,
  STRATEGIC_LABELS,
} from './allWorkText';
import type {
  AllWorkOptions,
  AllWorkPreferences,
  AllWorkQuery,
  AllWorkRow,
  FilterValue,
  Filters,
  SavedView,
  Sort,
  ViewState,
} from './allWorkTypes';

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
const VALUE_FILTERS = [
  'planStatuses',
  'containerStatuses',
  'activityStatuses',
  'planTypes',
  'planCurrencies',
  'finance',
] as const;
type ValueFilter = (typeof VALUE_FILTERS)[number];

/** A plan currency in the address: three capital letters. */
const CURRENCY_CODE = /^[A-Z]{3}$/;

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
 * The query of GET /api/all-work for a state: only what is in use. Columns go with the Grid (the
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

/** The address parameters other pages may link with (the API's names, D-122). */
const ADDRESS_KEYS = [
  'year',
  'yearFrom',
  'yearTo',
  'planType',
  'planCurrency',
  'finance',
  'plan',
  'planStatus',
  'containerStatus',
  'activityStatus',
  'owner',
  'team',
  'businessUnit',
  'activityType',
  'strategic',
  'showLevels',
  'search',
] as const;

/** Whether the address names any All work filter. */
export function hasAddressFilters(params: URLSearchParams): boolean {
  return ADDRESS_KEYS.some((key) => params.has(key));
}

/**
 * The state that an address such as /all-work?owner=U-1003&activityStatus=declined opens (feature
 * 009's cards, D-122): those filters over `base`. Values that are not valid are ignored. An ID
 * keeps itself as its name until the filter bar's choices name it.
 */
export function fromAddress(params: URLSearchParams, base: ViewState): ViewState {
  const idValues = (key: string) =>
    params.getAll(key).map((id): FilterValue => ({ id, label: id }));
  const allowed = <T extends string>(key: string, values: readonly T[]) =>
    params.getAll(key).filter((value): value is T => (values as readonly string[]).includes(value));
  const one = <T extends string>(key: string, values: readonly T[], fallback: T): T =>
    allowed(key, values)[0] ?? fallback;

  const year = Number(params.get('year'));
  const isYear = (value: number) => Number.isInteger(value) && value >= 2000 && value <= 2100;
  const yearFrom = Number(params.get('yearFrom') ?? params.get('year'));
  const yearTo = Number(params.get('yearTo'));
  const planTypes = allowed('planType', keysOf(PLAN_TYPE_LABELS));
  const planCurrencies = params.getAll('planCurrency').filter((code) => CURRENCY_CODE.test(code));
  const finance = allowed('finance', keysOf(FINANCE_LABELS));
  const filters: Filters = {
    plans: idValues('plan'),
    planStatuses: allowed('planStatus', keysOf(PLAN_STATUS_LABELS)),
    containerStatuses: allowed('containerStatus', keysOf(CONTAINER_STATUS_LABELS)),
    activityStatuses: allowed('activityStatus', keysOf(ACTIVITY_STATUS_LABELS)),
    owners: idValues('owner'),
    teams: idValues('team'),
    businessUnits: idValues('businessUnit'),
    activityTypes: idValues('activityType'),
    strategic: one('strategic', keysOf(STRATEGIC_LABELS), 'include'),
    showLevels: one('showLevels', keysOf(SHOW_LEVELS_LABELS), 'everything'),
    ...(planTypes.length > 0 && { planTypes }),
    ...(planCurrencies.length > 0 && { planCurrencies }),
    ...(finance.length > 0 && { finance }),
  };
  // A range of at most five years (016 FR-034); anything else falls back to one year.
  if (isYear(yearFrom) && isYear(yearTo) && yearTo > yearFrom && yearTo - yearFrom < MAX_YEARS) {
    return withYears({ ...base, filters, search: params.get('search') ?? '' }, yearFrom, yearTo);
  }
  const { yearFrom: _from, yearTo: _to, ...single } = base;
  void _from;
  void _to;
  return {
    ...single,
    year: isYear(year) ? year : isYear(yearFrom) ? yearFrom : base.year,
    filters,
    search: params.get('search') ?? '',
  };
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

/** One row on screen, with whether it is open and how many rows it holds. */
export interface VisibleRow {
  row: AllWorkRow;
  expanded: boolean;
  /** The rows directly under it that can be shown (unassigned rows only while switched on). */
  childCount: number;
}

/** The keys of the rows that have a search hit somewhere below them (data-model §4). */
export function rowsWithHitBelow(rows: AllWorkRow[]): Set<string> {
  const parentOf = new Map(rows.map((row) => [row.key, row.parentKey]));
  const keys = new Set<string>();
  for (const row of rows) {
    if (!row.searchHit) continue;
    let parent = row.parentKey;
    while (parent && !keys.has(parent)) {
      keys.add(parent);
      parent = parentOf.get(parent) ?? null;
    }
  }
  return keys;
}

/**
 * Whether a row is open (data-model §4, FR-011): by default plans, campaigns and initiatives are
 * open and activities closed; with a search, only the rows with a hit below them are open (a
 * matching container shows its contents, collapsed); Expand all and Collapse all open or close
 * everything. A row that was opened or closed by hand is the other way round.
 */
export function isExpanded(
  row: AllWorkRow,
  state: ViewState,
  hitBelow: Set<string>,
  toggled: Set<string> = new Set(state.expansion.toggled),
): boolean {
  let open: boolean;
  switch (state.expansion.mode) {
    case 'allExpanded':
      open = true;
      break;
    case 'allCollapsed':
      open = false;
      break;
    default:
      open =
        state.search.trim() !== ''
          ? hitBelow.has(row.key)
          : row.level === 'plan' || row.level === 'campaign' || row.level === 'initiative';
  }
  return open !== toggled.has(row.key);
}

/**
 * The rows on screen, in order (data-model §4): every row whose parents are all open, without the
 * unassigned rows while the switch is off (FR-012).
 */
export function visibleRows(rows: AllWorkRow[], state: ViewState): VisibleRow[] {
  const shown = state.showUnassigned ? rows : rows.filter((row) => row.level !== 'unassigned');
  const childCount = new Map<string, number>();
  for (const row of shown) {
    if (row.parentKey) {
      childCount.set(row.parentKey, (childCount.get(row.parentKey) ?? 0) + 1);
    }
  }
  const hitBelow = rowsWithHitBelow(rows);
  const toggled = new Set(state.expansion.toggled);
  const openKeys = new Set<string>();
  const result: VisibleRow[] = [];
  for (const row of shown) {
    if (row.parentKey && !openKeys.has(row.parentKey)) {
      continue;
    }
    const count = childCount.get(row.key) ?? 0;
    const expanded = count > 0 && isExpanded(row, state, hitBelow, toggled);
    if (expanded) {
      openKeys.add(row.key);
    }
    result.push({ row, expanded, childCount: count });
  }
  return result;
}

/** Opens a closed row or closes an open one (FR-011). */
export function toggleRow(state: ViewState, key: string): ViewState {
  const toggled = state.expansion.toggled.includes(key)
    ? state.expansion.toggled.filter((other) => other !== key)
    : [...state.expansion.toggled, key];
  return { ...state, expansion: { ...state.expansion, toggled } };
}

/**
 * The sort after a click on a column heading (FR-015): ascending, then descending, then the
 * plan-tree order again. Another column starts ascending.
 */
export function nextSort(sort: Sort | null, column: string): Sort | null {
  if (!sort || sort.column !== column) {
    return { column, direction: 'asc' };
  }
  return sort.direction === 'asc' ? { column, direction: 'desc' } : null;
}

/** A column moved one place up (-1) or down (+1) in the chosen order (FR-021). */
export function moveColumn(columns: string[], key: string, by: -1 | 1): string[] {
  const from = columns.indexOf(key);
  const to = from + by;
  if (from === -1 || to < 0 || to >= columns.length) {
    return columns;
  }
  const moved = [...columns];
  [moved[from], moved[to]] = [moved[to], moved[from]];
  return moved;
}

/** A column dragged onto another one takes its place; the others move along (FR-021). */
export function dropColumn(columns: string[], dragged: string, target: string): string[] {
  if (dragged === target || !columns.includes(dragged) || !columns.includes(target)) {
    return columns;
  }
  const without = columns.filter((key) => key !== dragged);
  const at = without.indexOf(target) + (columns.indexOf(dragged) < columns.indexOf(target) ? 1 : 0);
  return [...without.slice(0, at), dragged, ...without.slice(at)];
}

/** One page of the rows on screen (D-115). */
export interface RowPage {
  rows: VisibleRow[];
  /** The page shown, 1 or more (a page beyond the last becomes the last). */
  page: number;
  pageCount: number;
  /** The position of the first and last row of the page, from 1; 0 when there are no rows. */
  from: number;
  to: number;
  total: number;
}

export function pageOf(rows: VisibleRow[], page: number, pageSize: number): RowPage {
  const pageCount = Math.max(1, Math.ceil(rows.length / pageSize));
  const current = Math.min(Math.max(1, page), pageCount);
  const start = (current - 1) * pageSize;
  const pageRows = rows.slice(start, start + pageSize);
  return {
    rows: pageRows,
    page: current,
    pageCount,
    from: pageRows.length > 0 ? start + 1 : 0,
    to: start + pageRows.length,
    total: rows.length,
  };
}

/**
 * "FY2026 Lubricants Europe › Helix launch": the rows above the first row of a page that are on
 * an earlier page, for "Continued: <path>" (D-115). Empty when the page starts with a top row.
 */
export function continuedPath(page: VisibleRow[], all: AllWorkRow[]): string {
  const first = page[0]?.row;
  if (!first?.parentKey) {
    return '';
  }
  const byKey = new Map(all.map((row) => [row.key, row]));
  const names: string[] = [];
  let parent = byKey.get(first.parentKey);
  while (parent) {
    names.unshift(parent.name);
    parent = parent.parentKey ? byKey.get(parent.parentKey) : undefined;
  }
  return names.join(' › ');
}

/** One filter value in use, as a removable chip (FR-008): "Activity status: Declined". */
export interface Chip {
  /** "<filter>:<value>", for removeChip. */
  key: string;
  label: string;
}

/**
 * The chips of every filter in use but the year (FR-008), in the filter bar's order. Names come
 * from the filter bar's choices, else from the name stored when the value was chosen.
 */
export function chipsOf(state: ViewState, options?: AllWorkOptions): Chip[] {
  const { filters } = state;
  const chips: Chip[] = [];
  const add = (filter: keyof Filters, value: string, label: string) =>
    chips.push({ key: `${filter}:${value}`, label: `${FILTER_LABELS[filter]}: ${label}` });
  const idLabel = (filter: IdFilter, value: FilterValue) =>
    optionLabel(choicesOf(filter, options), value);

  filters.plans.forEach((value) => add('plans', value.id, idLabel('plans', value)));
  filters.planStatuses.forEach((value) => add('planStatuses', value, PLAN_STATUS_LABELS[value]));
  filters.containerStatuses.forEach((value) =>
    add('containerStatuses', value, CONTAINER_STATUS_LABELS[value]),
  );
  filters.activityStatuses.forEach((value) =>
    add('activityStatuses', value, ACTIVITY_STATUS_LABELS[value]),
  );
  filters.owners.forEach((value) => add('owners', value.id, idLabel('owners', value)));
  filters.teams.forEach((value) => add('teams', value.id, idLabel('teams', value)));
  filters.businessUnits.forEach((value) =>
    add('businessUnits', value.id, idLabel('businessUnits', value)),
  );
  filters.activityTypes.forEach((value) =>
    add('activityTypes', value.id, idLabel('activityTypes', value)),
  );
  if (filters.strategic !== 'include') {
    add('strategic', filters.strategic, STRATEGIC_LABELS[filters.strategic]);
  }
  if (filters.showLevels !== 'everything') {
    add('showLevels', filters.showLevels, SHOW_LEVELS_LABELS[filters.showLevels]);
  }
  filters.planTypes?.forEach((value) => add('planTypes', value, PLAN_TYPE_LABELS[value]));
  filters.planCurrencies?.forEach((value) => add('planCurrencies', value, value));
  filters.finance?.forEach((value) => add('finance', value, FINANCE_LABELS[value]));
  return chips;
}

/** The state without the chip's value. */
export function removeChip(state: ViewState, chipKey: string): ViewState {
  const separator = chipKey.indexOf(':');
  const filter = chipKey.slice(0, separator) as keyof Filters;
  const value = chipKey.slice(separator + 1);
  const filters: Filters = { ...state.filters };
  if ((ID_FILTERS as readonly string[]).includes(filter)) {
    const key = filter as IdFilter;
    filters[key] = filters[key].filter((chosen) => chosen.id !== value);
  } else if ((VALUE_FILTERS as readonly string[]).includes(filter)) {
    const key = filter as ValueFilter;
    // The lists hold different words; filtering by text keeps each list's own type.
    (filters[key] as string[]) = ((filters[key] ?? []) as string[]).filter(
      (chosen) => chosen !== value,
    );
  } else if (filter === 'strategic') {
    filters.strategic = 'include';
  } else if (filter === 'showLevels') {
    filters.showLevels = 'everything';
  }
  return { ...state, filters };
}

/** Clear all (FR-008): every filter and the search go; the year stays. */
export function clearAll(state: ViewState): ViewState {
  return { ...state, filters: NO_FILTERS, search: '' };
}

/** The filter of each repeatable or single filter parameter of the API. */
const PARAMETER_FILTERS: Record<string, keyof Filters> = {
  plan: 'plans',
  planStatus: 'planStatuses',
  containerStatus: 'containerStatuses',
  activityStatus: 'activityStatuses',
  owner: 'owners',
  team: 'teams',
  businessUnit: 'businessUnits',
  activityType: 'activityTypes',
  strategic: 'strategic',
  showLevels: 'showLevels',
  planType: 'planTypes',
  planCurrency: 'planCurrencies',
  finance: 'finance',
};

/**
 * The state with the parts the API refused set back to the standard view's (plan "States":
 * the offending filter is reset). `fields` are the API's parameter names.
 */
export function withoutRefused(state: ViewState, fields: string[], standard: ViewState): ViewState {
  let next = state;
  for (const field of fields) {
    const filter = PARAMETER_FILTERS[field];
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

/** Whether any filter but the year, or a search, is in use. */
export function hasFilters(state: ViewState): boolean {
  return chipsOf(state).length > 0 || state.search.trim() !== '';
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

/** A value's name: from the choices (with "(inactive)"), else the name stored with it. */
function optionLabel(
  choices: { id: string; label: string; inactive?: boolean }[],
  value: FilterValue,
): string {
  const choice = choices.find((candidate) => candidate.id === value.id);
  if (!choice) {
    return value.label;
  }
  return choice.inactive ? `${choice.label} (inactive)` : choice.label;
}

function keysOf<T extends string>(labels: Record<T, string>): T[] {
  return Object.keys(labels) as T[];
}
