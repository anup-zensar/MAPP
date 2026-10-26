import type { FilterChip } from '@/ui/FilterChips/FilterChips';
import { DEFAULT_FILTERS, yearRangeLabel } from './list-preferences';
import type { PlanListFilters } from './list-preferences';
import { PLAN_TYPE_LABELS } from './plan-currency';
import { STATUS } from './plan-text';

/** The removable chips of the filters that differ from the defaults (FR-003). */
export function filterChips(filters: PlanListFilters): FilterChip[] {
  const years = yearRangeLabel(filters);
  return [
    ...(filters.status !== 'all' ? [{ key: 'status', label: STATUS[filters.status].word }] : []),
    ...(years !== null
      ? [
          {
            key: 'years',
            label: `${filters.yearFrom === filters.yearTo ? 'Year' : 'Years'}: ${years}`,
          },
        ]
      : []),
    ...(filters.types.length > 0
      ? [
          {
            key: 'types',
            label: `Plan type: ${filters.types.map((type) => PLAN_TYPE_LABELS[type]).join(', ')}`,
          },
        ]
      : []),
    ...(filters.scope === 'myTeams' ? [{ key: 'scope', label: 'My teams’ plans' }] : []),
    ...(filters.q.trim() ? [{ key: 'q', label: `Search: ${filters.q.trim()}` }] : []),
  ];
}

/** The filters a removed chip sets back to their defaults. */
export function chipReset(key: string): Partial<PlanListFilters> {
  switch (key) {
    case 'status':
      return { status: DEFAULT_FILTERS.status };
    case 'years':
      return { yearFrom: null, yearTo: null };
    case 'types':
      return { types: DEFAULT_FILTERS.types };
    case 'scope':
      return { scope: DEFAULT_FILTERS.scope };
    case 'q':
      return { q: DEFAULT_FILTERS.q };
    default:
      return {};
  }
}
