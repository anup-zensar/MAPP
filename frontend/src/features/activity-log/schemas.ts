import { z } from 'zod';
import { ACTION_FILTERS, DATE_RANGE_KEYS, NO_FILTERS } from './log-filters';
import type { LogFilterState } from './log-filters';

const ADDRESS_NAMES = ['person', 'type', 'plan', 'record', 'inside', 'range', 'from', 'to', 'q'];
const ACTION_VALUES = ACTION_FILTERS.map((choice) => choice.value);

const text = z.string().catch('');
const calendarDate = z
  .string()
  .regex(/^\d{4}-\d{2}-\d{2}$/)
  .catch('');

/** The activity log's address (FR-003): anything it does not know is left out. */
const logAddressSchema = z
  .object({
    person: text,
    type: text,
    action: z.array(z.string()),
    plan: text,
    record: text,
    inside: text,
    range: z.enum(DATE_RANGE_KEYS).or(z.literal('')).catch(''),
    from: calendarDate,
    to: calendarDate,
    q: text,
  })
  .transform((address): LogFilterState => ({
    person: address.person,
    type: address.type,
    actions: ACTION_VALUES.filter((value) => address.action.includes(value)),
    plan: address.plan,
    record: address.record,
    inside: address.record !== '' && address.inside === '1',
    range: address.range,
    from: address.range === 'custom' ? address.from : '',
    to: address.range === 'custom' ? address.to : '',
    q: address.q,
  }));

/** The filters in the address; anything it does not know is left out. */
export function filtersFromParams(params: URLSearchParams): LogFilterState {
  const address = Object.fromEntries(ADDRESS_NAMES.map((name) => [name, params.get(name)]));
  const parsed = logAddressSchema.safeParse({ ...address, action: params.getAll('action') });
  return parsed.success ? parsed.data : NO_FILTERS;
}
