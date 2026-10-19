import { describe, expect, it } from 'vitest';
import { monthRange } from './useMonthRange';

describe('useMonthRange (feature 008, FR-028)', () => {
  it('is the local calendar month (tests run in UTC)', () => {
    expect(monthRange(new Date('2026-03-12T16:00:00Z'))).toEqual({
      monthStart: '2026-03-01T00:00:00.000Z',
      monthEnd: '2026-04-01T00:00:00.000Z',
    });
  });

  it('runs into the next year in December', () => {
    expect(monthRange(new Date('2026-12-31T23:59:00Z'))).toEqual({
      monthStart: '2026-12-01T00:00:00.000Z',
      monthEnd: '2027-01-01T00:00:00.000Z',
    });
  });
});
