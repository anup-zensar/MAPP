import { describe, expect, it } from 'vitest';
import { waterfall } from './waterfall';

const LUBRICANTS = {
  inqToMqlRate: '2.00',
  mqlToDealRate: '5.00',
  averageDealSize: { amount: '10000.00' },
};
const MOBILITY = {
  inqToMqlRate: '3.00',
  mqlToDealRate: '4.00',
  averageDealSize: { amount: '5000.00' },
};

describe('waterfall', () => {
  // The table of data-model 002 §2.1, the same rows as Go's waterfall_test.go (FR-010, SC-006).
  it.each([
    ['Lubricants 1.0', LUBRICANTS, 250_000n, 5_000n, 250n, '2500000.00'],
    ['Mobility 1.0', MOBILITY, 250_000n, 7_500n, 300n, '1500000.00'],
    ['Lubricants 1.0 (0.5 rounds up)', LUBRICANTS, 25n, 1n, 0n, '0.00'],
    ['Lubricants 1.0 (nothing)', LUBRICANTS, 0n, 0n, 0n, '0.00'],
  ])('calculates %s', (_, model, inq, mql, deals, revenue) => {
    expect(waterfall(inq, model)).toEqual({ mql, deals, revenue });
  });

  it('says when the revenue is too large to calculate (FR-010)', () => {
    const huge = { ...LUBRICANTS, averageDealSize: { amount: '100000000.00' } };
    expect(waterfall(1_000_000_000_000_000n, huge)).toBe('tooLarge');
  });
});
