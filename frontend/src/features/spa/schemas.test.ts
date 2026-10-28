import fc from 'fast-check';
import { describe, expect, it } from 'vitest';
import { spaView } from '@/testing/fixtures/spa';
import { readSpaState, SORT_COLUMNS, toSpaParams } from './schemas';
import type { SpaUrlState } from './schemas';
import { sortRows } from './sort';

const KEYS = ['campaign', 'dimension', 'period', 'sort', 'other'];

describe('013 FR-005: the address of the SP&A view', () => {
  it('never fails on any address and always reads a valid state', () => {
    fc.assert(
      fc.property(fc.array(fc.tuple(fc.constantFrom(...KEYS), fc.string())), (pairs) => {
        const state = readSpaState(new URLSearchParams(pairs));
        expect(['quarter', 'month']).toContain(state.period);
        expect(state.campaign === undefined || /^C-[0-9]+$/.test(state.campaign)).toBe(true);
      }),
    );
  });

  it('reads back every state it writes', () => {
    const anyState: fc.Arbitrary<SpaUrlState> = fc.record({
      campaign: fc.option(
        fc.integer({ min: 1, max: 99999 }).map((n) => `C-${n}`),
        { nil: undefined },
      ),
      dimension: fc.option(fc.constantFrom('activityType', 'attr:SA-1012'), { nil: undefined }),
      period: fc.constantFrom('quarter' as const, 'month' as const),
      sort: fc.option(
        fc.record({
          column: fc.constantFrom(...SORT_COLUMNS),
          direction: fc.constantFrom('asc' as const, 'desc' as const),
        }),
        { nil: undefined },
      ),
    });
    fc.assert(
      fc.property(anyState, (state) => {
        expect(readSpaState(toSpaParams(state))).toEqual(state);
      }),
    );
  });
});

describe('013 FR-019: sorting the activity table', () => {
  const rows = spaView().activities;

  it('keeps the plan-tree order without a sort', () => {
    expect(sortRows(rows, undefined)).toBe(rows);
  });

  it('orders amounts exactly and puts empty values last either way', () => {
    const names = (column: (typeof SORT_COLUMNS)[number], direction: 'asc' | 'desc') =>
      sortRows(rows, { column, direction }).map((row) => row.name);
    expect(names('actualSpend', 'desc')).toEqual(['Launch film', 'Social always-on']);
    expect(names('actualSpend', 'asc')).toEqual(['Social always-on', 'Launch film']);
    expect(names('owner', 'asc')).toEqual(['Launch film', 'Social always-on']);
    expect(names('owner', 'desc')).toEqual(['Launch film', 'Social always-on']);
    expect(names('variancePercent', 'desc')).toEqual(['Launch film', 'Social always-on']);
  });
});
