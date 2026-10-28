import type { FilterChip } from '@/ui/FilterChips/FilterChips';
import { FILTER_LABELS, NOT_COMPLETED } from './list-text';
import { STATUS_WORDS } from './purchase-request-text';
import type { ListSearch } from './schemas';

/** A choice of a filter menu: the value the address holds and the words people read. */
export interface Choice {
  value: string;
  label: string;
}

/** The names of the chosen values the page knows: PR type options and people. */
export interface ChoiceNames {
  types: Choice[];
  owners: Choice[];
  formatDate: (day: string) => string;
}

const NAMED_KEYS = ['type', 'vendor', 'owner'] as const;

function nameOf(choices: Choice[], value: string): string {
  return choices.find((choice) => choice.value === value)?.label ?? value;
}

/** Every filter in use as a chip, keyed "<filter>:<value>" ("Status: Approved"). */
export function listChips(search: ListSearch, names: ChoiceNames): FilterChip[] {
  const choices = { type: names.types, owner: names.owners, vendor: [] };
  const chips: FilterChip[] = [
    ...search.status.map((status) => ({
      key: `status:${status}`,
      label: `${FILTER_LABELS.status}: ${STATUS_WORDS[status]}`,
    })),
    ...NAMED_KEYS.flatMap((key) =>
      search[key].map((value) => ({
        key: `${key}:${value}`,
        label: `${FILTER_LABELS[key]}: ${nameOf(choices[key], value)}`,
      })),
    ),
  ];
  if (search.typeNotCompleted) {
    chips.push({ key: 'typeNotCompleted', label: `${FILTER_LABELS.type}: ${NOT_COMPLETED}` });
  }
  if (search.vendorNotCompleted) {
    chips.push({ key: 'vendorNotCompleted', label: `${FILTER_LABELS.vendor}: ${NOT_COMPLETED}` });
  }
  if (search.from) {
    chips.push({ key: 'from', label: `${FILTER_LABELS.from}: ${names.formatDate(search.from)}` });
  }
  if (search.to) {
    chips.push({ key: 'to', label: `${FILTER_LABELS.to}: ${names.formatDate(search.to)}` });
  }
  return chips;
}

/** The search without the filter of one chip. */
export function withoutChip(search: ListSearch, key: string): Partial<ListSearch> {
  const [filter = '', ...rest] = key.split(':');
  const value = rest.join(':');
  switch (filter) {
    case 'status':
      return { status: search.status.filter((chosen) => chosen !== value) };
    case 'type':
    case 'vendor':
    case 'owner':
      return { [filter]: search[filter].filter((chosen) => chosen !== value) };
    case 'typeNotCompleted':
    case 'vendorNotCompleted':
      return { [filter]: false };
    case 'from':
    case 'to':
      return { [filter]: '' };
    default:
      return {};
  }
}
