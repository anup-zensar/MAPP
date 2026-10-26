import { describe, expect, it } from 'vitest';
import { dropColumn, moveColumn, nextSort } from './view-columns';

describe('sorting and column order (FR-015, FR-021)', () => {
  it('sorts ascending, then descending, then in the plan-tree order again', () => {
    const first = nextSort(null, 'plannedCost');
    expect(first).toEqual({ column: 'plannedCost', direction: 'asc' });
    const second = nextSort(first, 'plannedCost');
    expect(second).toEqual({ column: 'plannedCost', direction: 'desc' });
    expect(nextSort(second, 'plannedCost')).toBeNull();
    expect(nextSort(second, 'owner')).toEqual({ column: 'owner', direction: 'asc' });
  });

  it('moves a column one place up or down, never past the ends', () => {
    expect(moveColumn(['id', 'status', 'owner'], 'owner', -1)).toEqual(['id', 'owner', 'status']);
    expect(moveColumn(['id', 'status', 'owner'], 'id', 1)).toEqual(['status', 'id', 'owner']);
    expect(moveColumn(['id', 'status'], 'id', -1)).toEqual(['id', 'status']);
    expect(moveColumn(['id', 'status'], 'status', 1)).toEqual(['id', 'status']);
  });

  it('puts a dragged column in the place of the one it is dropped on', () => {
    const columns = ['id', 'status', 'owner', 'plannedCost'];
    expect(dropColumn(columns, 'plannedCost', 'status')).toEqual([
      'id',
      'plannedCost',
      'status',
      'owner',
    ]);
    expect(dropColumn(columns, 'id', 'owner')).toEqual(['status', 'owner', 'id', 'plannedCost']);
    expect(dropColumn(columns, 'id', 'id')).toEqual(columns);
  });
});
