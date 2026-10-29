import fc from 'fast-check';
import { describe, expect, it } from 'vitest';
import { globalBudgetSchema, yearOf } from './schemas';

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

describe('038 FR-030 (M-12): the global budget form finds every problem at the first Save', () => {
  const VALUES = {
    name: 'Global Mobility',
    year: '2026',
    amount: '10',
    ownerId: 'U-1001',
    description: '',
    overspend: 'warn' as const,
  };

  it('names the empty name, the amount and the missing owner at once', () => {
    const result = globalBudgetSchema.safeParse({
      ...VALUES,
      name: '  ',
      amount: 'abc',
      ownerId: '',
    });

    expect(result.error?.issues.map((issue) => [issue.path.join('.'), issue.message])).toEqual([
      ['name', 'Enter a name for the global budget.'],
      ['amount', 'Enter an amount of 0 or more.'],
      ['ownerId', 'Choose a person who can manage global budgets for all markets.'],
    ]);
  });

  it('refuses a name longer than 120 characters', () => {
    const result = globalBudgetSchema.safeParse({ ...VALUES, name: 'x'.repeat(121) });

    expect(result.error?.issues.map((issue) => issue.message)).toEqual([
      'Use 120 characters or fewer.',
    ]);
  });
});
