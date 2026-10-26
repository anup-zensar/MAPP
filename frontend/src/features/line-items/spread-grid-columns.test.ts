import { describe, expect, it } from 'vitest';
import type { components } from '@/api/generated/schema';
import { nth } from '@/testing/dom';
import { quarterColumns } from './spread-grid-columns';

type SpreadGrid = components['schemas']['SpreadGrid'];

const usd = (amount: string) => ({ amount, currency: 'USD' });

describe('quarterColumns (FR-003)', () => {
  it('adds the months of each quarter and keeps the No dates yet column', () => {
    const grid: SpreadGrid = {
      columns: [
        { key: '2026-03', month: '2026-03', outsideActivityDates: true },
        { key: '2026-04', month: '2026-04', outsideActivityDates: false },
        { key: '2026-05', month: '2026-05', outsideActivityDates: false },
        { key: 'noDates', month: null, outsideActivityDates: false },
      ],
      rows: [
        {
          lineItemId: 'L-1',
          name: 'Agency fee',
          cells: [usd('100.00'), usd('200.00'), null, usd('5.00')],
          total: usd('305.00'),
        },
      ],
      totals: [usd('100.00'), usd('200.00'), null, usd('5.00')],
      total: usd('305.00'),
    };
    const quarters = quarterColumns(grid);
    expect(quarters.columns.map((c) => c.label)).toEqual(['Q1 2026', 'Q2 2026', 'No dates yet']);
    expect(quarters.columns.map((c) => c.outsideActivityDates)).toEqual([true, false, false]);
    expect(nth(quarters.rows, 0).cells).toEqual([usd('100.00'), usd('200.00'), usd('5.00')]);
    expect(quarters.totals).toEqual([usd('100.00'), usd('200.00'), usd('5.00')]);
  });
});
