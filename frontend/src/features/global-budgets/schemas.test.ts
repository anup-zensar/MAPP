import fc from 'fast-check';
import { describe, expect, it } from 'vitest';
import { yearOf } from './schemas';

const CURRENT_YEAR = 2026;

describe('the year of the global budgets page (014 FR-036)', () => {
  it('never fails on any address, and gives all years, a year from 2000 to 2100 or the current one', () => {
    fc.assert(
      fc.property(fc.option(fc.string(), { nil: undefined }), (value) => {
        const params = new URLSearchParams(value === undefined ? {} : { year: value });
        const year = yearOf(params, CURRENT_YEAR);
        expect(year === null || (Number.isInteger(year) && year >= 2000 && year <= 2100)).toBe(
          true,
        );
      }),
    );
  });

  it('reads back the year it was given', () => {
    fc.assert(
      fc.property(fc.integer({ min: 2000, max: 2100 }), (year) => {
        expect(yearOf(new URLSearchParams({ year: String(year) }), CURRENT_YEAR)).toBe(year);
      }),
    );
    expect(yearOf(new URLSearchParams({ year: 'all' }), CURRENT_YEAR)).toBeNull();
  });
});
