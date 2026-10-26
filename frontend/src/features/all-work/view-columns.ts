import type { Sort } from './all-work-types';

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
  return columns.toSpliced(from, 1).toSpliced(to, 0, key);
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
