/**
 * The activity log's filters (feature 011, FR-002, FR-003, plan "Filters"): kept in the address,
 * so links, Back and **Open in activity log** work. This file turns the address into the filter
 * state, the state into the API's query and the chips, and works out the local date ranges.
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
import type { components, operations } from '@/api/schema';
import type { FilterChip } from '@/components';
import { ACTION_CHIPS, capitalise, planningLabel } from './logText';

export type ActionFilter = components['schemas']['ActivityLogActionFilter'];
type ActorRef = components['schemas']['ActorRef'];
type RecordRef = components['schemas']['RecordRef'];
type RecordTypeCount = components['schemas']['RecordTypeCount'];

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

export type DateRangeKey = 'today' | 'week' | '30d' | 'year' | 'custom';

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

const DATE = /^\d{4}-\d{2}-\d{2}$/;

/** The filters in the address; anything it does not know is left out. */
export function filtersFromParams(params: URLSearchParams): LogFilterState {
  const range = params.get('range') ?? '';
  const record = params.get('record') ?? '';
  return {
    person: params.get('person') ?? '',
    type: params.get('type') ?? '',
    actions: ACTION_FILTERS.map((choice) => choice.value).filter((value) =>
      params.getAll('action').includes(value),
    ),
    plan: params.get('plan') ?? '',
    record,
    inside: record !== '' && params.get('inside') === '1',
    range: DATE_RANGES.some((choice) => choice.value === range) ? (range as DateRangeKey) : '',
    from: range === 'custom' ? dateParam(params.get('from')) : '',
    to: range === 'custom' ? dateParam(params.get('to')) : '',
    q: params.get('q') ?? '',
  };
}

function dateParam(value: string | null): string {
  return value && DATE.test(value) ? value : '';
}

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
      const [fromYear, fromMonth, fromDay] = state.from.split('-').map(Number);
      const [toYear, toMonth, toDay] = state.to.split('-').map(Number);
      return {
        from: localMidnight(fromYear, fromMonth - 1, fromDay),
        to: localMidnight(toYear, toMonth - 1, toDay + 1),
      };
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

/** What the chips need to name the filters: the summary's lists and the page's record. */
export interface ChipNames {
  people: ActorRef[];
  recordTypes: RecordTypeCount[];
  plans: RecordRef[];
  /** The record of the Record filter, as GET /api/v1/activity-log answered it. */
  record: RecordRef | null;
  /** Shows a calendar date in the person's format. */
  formatDate: (date: string) => string;
}

/** "Activity", "Line item": a record type's label for chips and selects. */
export function typeLabel(type: string, recordTypes: RecordTypeCount[]): string {
  const known = recordTypes.find((choice) => choice.type === type);
  return capitalise(known?.label ?? planningLabel(type));
}

/**
 * Every filter in use as a chip (FR-003): "Person: Daniel Okoro", "Type: Activity", "Action:
 * Deleted" (one per action), "Plan: FY2026 Lubricants Europe", "Record: Activity 'Film shoot'
 * (A-1042) and items inside", "Date: This week" or "Date: 01 Oct 2026 – 12 Oct 2026", "Search:
 * 'film'". The chip's key says which filter removeFilter() takes away.
 */
export function filterChips(state: LogFilterState, names: ChipNames): FilterChip[] {
  const chips: FilterChip[] = [];
  if (state.person) {
    const person = names.people.find((choice) => choice.id === state.person);
    chips.push({ key: 'person', label: `Person: ${person?.name ?? state.person}` });
  }
  if (state.type) {
    chips.push({ key: 'type', label: `Type: ${typeLabel(state.type, names.recordTypes)}` });
  }
  for (const action of state.actions) {
    chips.push({ key: `action:${action}`, label: `Action: ${ACTION_CHIPS[action].word}` });
  }
  if (state.plan) {
    const plan = names.plans.find((choice) => choice.id === state.plan);
    chips.push({ key: 'plan', label: `Plan: ${plan?.name ?? state.plan}` });
  }
  if (state.record) {
    chips.push({ key: 'record', label: recordChip(state, names) });
  }
  const dateChip = dateChipLabel(state, names.formatDate);
  if (dateChip) {
    chips.push({ key: 'date', label: dateChip });
  }
  if (state.q.trim()) {
    chips.push({ key: 'q', label: `Search: '${state.q.trim()}'` });
  }
  return chips;
}

function recordChip(state: LogFilterState, names: ChipNames): string {
  const record = names.record?.id === state.record ? names.record : null;
  const inside = state.inside ? ' and items inside' : '';
  if (!record) {
    return `Record: ${state.record}${inside}`;
  }
  return `Record: ${typeLabel(record.type, names.recordTypes)} '${record.name}' (${record.id})${inside}`;
}

function dateChipLabel(state: LogFilterState, formatDate: (date: string) => string): string {
  if (state.range === 'custom') {
    return state.from && state.to && !customRangeError(state)
      ? `Date: ${formatDate(state.from)} – ${formatDate(state.to)}`
      : '';
  }
  const range = DATE_RANGES.find((choice) => choice.value === state.range);
  return state.range && range ? `Date: ${range.label}` : '';
}

/** The filters without the one a chip names. */
export function removeFilter(state: LogFilterState, key: string): LogFilterState {
  if (key.startsWith('action:')) {
    return { ...state, actions: state.actions.filter((action) => `action:${action}` !== key) };
  }
  switch (key) {
    case 'person':
      return { ...state, person: '' };
    case 'type':
      return { ...state, type: '' };
    case 'plan':
      return { ...state, plan: '' };
    case 'record':
      return { ...state, record: '', inside: false };
    case 'date':
      return { ...state, range: '', from: '', to: '' };
    case 'q':
      return { ...state, q: '' };
    default:
      return state;
  }
}
