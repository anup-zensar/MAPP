import { toCents, toHundredths } from '@/format/decimal';
import type { Money, SpaActivityRow } from './api';
import type { SortColumn, SpaSort } from './schemas';

type Key = bigint | string | null;

const LARGEST_FIRST: readonly SortColumn[] = [
  'plannedCost',
  'committed',
  'actualSpend',
  'variancePercent',
];

function moneyKey(value: Money | null): Key {
  return value ? toCents(value.amount) : null;
}

function keyOf(row: SpaActivityRow, column: SortColumn): Key {
  switch (column) {
    case 'plannedCost':
    case 'committed':
    case 'actualSpend':
      return moneyKey(row[column]);
    case 'variancePercent':
      return row.variancePercent === null ? null : toHundredths(row.variancePercent);
    case 'campaign':
      return row.campaign.name.toLowerCase();
    case 'name':
    case 'owner':
    case 'activityType':
    case 'startQuarter':
    case 'status':
      return row[column]?.toLowerCase() ?? null;
  }
}

/** Empty values last in either direction, as everywhere in the app. */
function compareKeys(a: Key, b: Key): number {
  if (a === null || b === null) {
    return a === b ? 0 : a === null ? 1 : -1;
  }
  if (a < b) return -1;
  return a > b ? 1 : 0;
}

/** The rows in the chosen order; the plan-tree order the API gives when there is none. */
export function sortRows(rows: SpaActivityRow[], sort: SpaSort | undefined): SpaActivityRow[] {
  if (!sort) {
    return rows;
  }
  const sign = sort.direction === 'asc' ? 1 : -1;
  return rows.toSorted((x, y) => {
    const a = keyOf(x, sort.column);
    const b = keyOf(y, sort.column);
    return a === null || b === null ? compareKeys(a, b) : sign * compareKeys(a, b);
  });
}

/** The first direction of a column: amounts largest first, words A to Z. */
export function firstDirection(column: SortColumn): 'asc' | 'desc' {
  return LARGEST_FIRST.includes(column) ? 'desc' : 'asc';
}
