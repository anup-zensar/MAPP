import type { components } from '@/api/generated/schema';
import type { FilterChip } from '@/ui/FilterChips/FilterChips';
import { customRangeError, DATE_RANGES } from './log-filters';
import type { LogFilterState } from './log-filters';
import { ACTION_CHIPS, capitalise, planningLabel } from './log-text';

type ActorRef = components['schemas']['ActorRef'];
type RecordRef = components['schemas']['RecordRef'];
type RecordTypeCount = components['schemas']['RecordTypeCount'];

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
