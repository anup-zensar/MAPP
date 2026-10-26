import fc from 'fast-check';
import { describe, expect, it } from 'vitest';
import { budgetYearOf } from './schemas';

describe('the budget year of the address (002 FR-014)', () => {
  it('never fails on any address, and gives a whole year or none', () => {
    fc.assert(
      fc.property(fc.string(), fc.string(), (query, year) => {
        const params = new URLSearchParams(query);
        params.append('year', year);
        const read = budgetYearOf(params);
        expect(read === undefined || Number.isInteger(read)).toBe(true);
      }),
    );
  });

  it('reads back the year the year filter writes', () => {
    fc.assert(
      fc.property(fc.integer(), (year) => {
        expect(budgetYearOf(new URLSearchParams({ year: String(year) }))).toBe(year);
      }),
    );
  });
});
