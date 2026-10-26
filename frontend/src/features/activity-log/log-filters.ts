/**
 * The activity log's filters (feature 011, FR-002, FR-003, plan "Filters"): kept in the address,
 * so links, Back and **Open in activity log** work. This file turns the filter state into the
 * address and the API's query, and works out the local date ranges; schemas.ts reads the address.
 *
 * | Address                         | API                    |
 * | person=U-1003                   | actorId                |
 * | type=activity                   | recordType             |
 * | action=… (repeated)             | action                 |
 * | plan=P-1001                     | planId                 |
 * | record=A-1042&inside=1          | recordId, includeInside |
 * | range=…, from=…, to=…           | from, to (UTC moments) |
 * | q=film                          | q                      |
 */
import type { components, operations } from '@/api/generated/schema';
import { ACTION_CHIPS } from './log-text';

export type ActionFilter = components['schemas']['ActivityLogActionFilter'];

/** The Action filter's choices, in the order of the menu (FR-003). */
export const ACTION_FILTERS: { value: ActionFilter; word: string }[] = (
  [
    'created',
    'changed',
    'statusChanged',
    'moved',
    'copied',
    'deleted',
    'restored',
    'deletedPermanently',
  ] as const
).map((value) => ({ value, word: ACTION_CHIPS[value].word }));

export const DATE_RANGE_KEYS = ['today', 'week', '30d', 'year', 'custom'] as const;

export type DateRangeKey = (typeof DATE_RANGE_KEYS)[number];

/** The date-range pills; '' is "Any time" (FR-003). */
export const DATE_RANGES: { value: DateRangeKey | ''; label: string }[] = [
  { value: '', label: 'Any time' },
  { value: 'today', label: 'Today' },
  { value: 'week', label: 'This week' },
  { value: '30d', label: 'Last 30 days' },
  { value: 'year', label: 'This year' },
  { value: 'custom', label: 'Custom' },
];

export const CUSTOM_RANGE_ERROR = 'The end date must be on or after the start date.';

/** Every filter of the page; '' and [] mean "not filtered". */
export interface LogFilterState {
  person: string;
  type: string;
  actions: ActionFilter[];
  plan: string;
  record: string;
  /** With `record`: also the items that are or were inside it. */
  inside: boolean;
  range: DateRangeKey | '';
  /** Custom range, as calendar dates "YYYY-MM-DD"; '' when not chosen. */
  from: string;
  to: string;
  q: string;
}

export const NO_FILTERS: LogFilterState = {
  person: '',
  type: '',
  actions: [],
  plan: '',
  record: '',
  inside: false,
  range: '',
  from: '',
  to: '',
  q: '',
};

/** What GET /api/v1/activity-log is asked, without paging. */
export type LogApiQuery = Omit<
  NonNullable<operations['listActivityLog']['parameters']['query']>,
  'beforeSeq' | 'limit'
>;

/** The address of the filters, in a fixed order; filters not in use are left out. */
export function filtersToParams(state: LogFilterState): URLSearchParams {
  const params = new URLSearchParams();
  const add = (name: string, value: string) => {
    if (value) {
      params.append(name, value);
    }
  };
  add('person', state.person);
  add('type', state.type);
  state.actions.forEach((action) => params.append('action', action));
  add('plan', state.plan);
  add('record', state.record);
  add('inside', state.record && state.inside ? '1' : '');
  add('range', state.range);
  if (state.range === 'custom') {
    add('from', state.from);
    add('to', state.to);
  }
  add('q', state.q);
  return params;
}

/** The custom range's field error, when the end date is before the start date. */
export function customRangeError(state: LogFilterState): string | undefined {
  return state.range === 'custom' && state.from && state.to && state.to < state.from
    ? CUSTOM_RANGE_ERROR
    : undefined;
}

/** Whether any filter is in use (a custom range counts once both dates are chosen). */
export function isFiltered(state: LogFilterState): boolean {
  const { from, to } = rangeBounds(state, new Date()) ?? {};
  return Boolean(
    state.person ||
    state.type ||
    state.actions.length > 0 ||
    state.plan ||
    state.record ||
    from ||
    to ||
    state.q.trim(),
  );
}

/**
 * The API's query for the filters, or null when the custom range is wrong (nothing is sent until
 * it is fixed).
 */
export function apiQuery(state: LogFilterState, now: Date = new Date()): LogApiQuery | null {
  const bounds = rangeBounds(state, now);
  if (!bounds) {
    return null;
  }
  const query: LogApiQuery = {};
  if (state.person) query.actorId = state.person;
  if (state.type) query.recordType = state.type;
  if (state.actions.length > 0) query.action = state.actions;
  if (state.plan) query.planId = state.plan;
  if (state.record) {
    query.recordId = state.record;
    if (state.inside) query.includeInside = true;
  }
  if (bounds.from) query.from = bounds.from;
  if (bounds.to) query.to = bounds.to;
  const q = state.q.trim();
  if (q) query.q = q;
  return query;
}

/**
 * The date range as UTC moments of local midnights (plan "Filters"): today from midnight; this
 * week from Monday's midnight; the last 30 days from midnight 29 days ago; this year from 1 January;
 * a custom range from the midnight of `from` to the midnight after `to` (the end is exclusive).
 * A custom range applies once both dates are chosen; null when its end is before its start.
 */
export function rangeBounds(
  state: LogFilterState,
  now: Date,
): { from?: string; to?: string } | null {
  switch (state.range) {
    case '':
      return {};
    case 'today':
      return { from: todayStart(now) };
    case 'week':
      return { from: weekStart(now) };
    case '30d':
      return { from: localMidnight(now.getFullYear(), now.getMonth(), now.getDate() - 29) };
    case 'year':
      return { from: localMidnight(now.getFullYear(), 0, 1) };
    case 'custom': {
      if (customRangeError(state)) {
        return null;
      }
      if (!state.from || !state.to) {
        return {};
      }
      return { from: midnightOf(state.from), to: midnightOf(state.to, 1) };
    }
  }
}

/** Midnight today in the person's time zone, as a UTC moment (summary `todayStart`). */
export function todayStart(now: Date = new Date()): string {
  return localMidnight(now.getFullYear(), now.getMonth(), now.getDate());
}

/** Midnight of this week's Monday in the person's time zone, as a UTC moment (`weekStart`). */
export function weekStart(now: Date = new Date()): string {
  const daysSinceMonday = (now.getDay() + 6) % 7;
  return localMidnight(now.getFullYear(), now.getMonth(), now.getDate() - daysSinceMonday);
}

function localMidnight(year: number, month: number, day: number): string {
  return new Date(year, month, day).toISOString();
}

/** The local midnight of a calendar date "YYYY-MM-DD", `daysLater` days on. */
function midnightOf(date: string, daysLater = 0): string {
  const [year = 0, month = 1, day = 1] = date.split('-').map(Number);
  return localMidnight(year, month - 1, day + daysLater);
}
