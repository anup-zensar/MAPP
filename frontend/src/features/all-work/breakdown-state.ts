import { isLabelled, PLAN_TYPE_LABELS } from './all-work-text';
import type {
  BreakdownChoice,
  BreakdownQuery,
  FilterableAttribute,
  Filters,
  ViewState,
} from './all-work-types';
import { NOT_FILLED_IN, withAttributeValues } from './attribute-filters';
import { toQueryParams, withYears } from './view-state';
import type { IdFilter } from './view-state';

export const ATTRIBUTE_GROUPING = 'attr:';
const DEFAULT_ATTRIBUTE = 'Spend type';
const FALLBACK_GROUPING = 'plan';
const NOT_FILLED_IN_GROUP = 'notFilledIn';

const ID_GROUPINGS: Record<string, IdFilter> = {
  plan: 'plans',
  team: 'teams',
  businessUnit: 'businessUnits',
  activityType: 'activityTypes',
  owner: 'owners',
};

export interface GroupRef {
  key: string;
  label: string;
}

/** Spend type when Settings has it, else the first Shell attribute, else Plan. */
export function defaultGrouping(attributes: FilterableAttribute[] | undefined): string {
  const list = attributes ?? [];
  const attribute = list.find((candidate) => candidate.label === DEFAULT_ATTRIBUTE) ?? list.at(0);
  return attribute ? `${ATTRIBUTE_GROUPING}${attribute.id}` : FALLBACK_GROUPING;
}

export function breakdownChoice(
  state: ViewState,
  attributes: FilterableAttribute[] | undefined,
): BreakdownChoice {
  return state.breakdown ?? { by: defaultGrouping(attributes), thenBy: null };
}

/** The Breakdown with its groupings written out, as an export or a saved view needs it. */
export function withBreakdownChoice(
  state: ViewState,
  attributes: FilterableAttribute[] | undefined,
): ViewState {
  return state.view === 'breakdown' && !state.breakdown
    ? { ...state, breakdown: breakdownChoice(state, attributes) }
    : state;
}

/** The query of GET /api/v1/all-work/breakdown: the view's filters and search, by and then by. */
export function breakdownQuery(state: ViewState, choice: BreakdownChoice): BreakdownQuery {
  const {
    view: _view,
    column: _column,
    sort: _sort,
    period: _period,
    comparePhased: _compare,
    finance: _finance,
    ...filters
  } = toQueryParams(state);
  void [_view, _column, _sort, _period, _compare, _finance];
  return { ...filters, by: choice.by, ...(choice.thenBy && { thenBy: choice.thenBy }) };
}

// The Grid filtered to one group (and its then-by group); null when no filter can show it, such as
// Not on a line item, No owner or a built-in grouping's Not filled in.
export function gridOfGroup(
  state: ViewState,
  choice: BreakdownChoice,
  groups: GroupRef[],
): ViewState | null {
  const groupings = [choice.by, choice.thenBy ?? ''];
  return groups.reduce<ViewState | null>(
    (next, group, index) => next && filteredBy(next, groupings[index] ?? '', group),
    { ...state, view: 'grid' },
  );
}

function filteredBy(state: ViewState, grouping: string, group: GroupRef): ViewState | null {
  if (grouping.startsWith(ATTRIBUTE_GROUPING)) {
    return withAttribute(state, grouping.slice(ATTRIBUTE_GROUPING.length), group.key);
  }
  if (group.key === NOT_FILLED_IN_GROUP) {
    return null;
  }
  if (grouping === 'year') {
    const year = Number(group.key);
    return withYears(state, year, year);
  }
  const change = builtInFilter(grouping, group);
  return change ? { ...state, filters: { ...state.filters, ...change } } : null;
}

function withAttribute(state: ViewState, attributeId: string, key: string): ViewState | null {
  if (key === 'notOnLineItem') {
    return null;
  }
  const value = key === NOT_FILLED_IN_GROUP ? NOT_FILLED_IN : key;
  const attributes = withAttributeValues(state.filters.attributes, attributeId, [value]);
  return { ...state, filters: { ...state.filters, attributes } };
}

function builtInFilter(grouping: string, group: GroupRef): Partial<Filters> | null {
  const idFilter = ID_GROUPINGS[grouping];
  if (idFilter && group.key !== 'noOwner') {
    return { [idFilter]: [{ id: group.key, label: group.label }] };
  }
  if (grouping === 'planType' && isLabelled(PLAN_TYPE_LABELS)(group.key)) {
    return { planTypes: [group.key] };
  }
  return grouping === 'planCurrency' ? { planCurrencies: [group.key] } : null;
}
