import { describe, expect, it } from 'vitest';
import {
  balanceOf,
  balanceText,
  monthKeys,
  monthLabel,
  overBy,
  quarterTotals,
  readAmount,
  splitEvenly,
  sum,
  withQuarter,
} from './phasing-math';

const cents = (amount: number) => BigInt(Math.round(amount * 100));

describe('phasingMath (016 data-model §3.3)', () => {
  it('splits a budget evenly over 12 months, the rest in December', () => {
    expect(splitEvenly(cents(600000), 12)).toEqual(Array(12).fill(cents(50000)));
    const split = splitEvenly(cents(100000), 12);
    expect(split.slice(0, 11)).toEqual(Array(11).fill(cents(8333.33)));
    expect(split[11]).toBe(cents(8333.37));
    expect(sum(split)).toBe(cents(100000));
  });

  it('spreads a quarter over its three months by the same rule', () => {
    expect(splitEvenly(cents(100000), 3)).toEqual([
      cents(33333.33),
      cents(33333.33),
      cents(33333.34),
    ]);
    const months = withQuarter(Array(12).fill(0n), 1, cents(100000));
    expect(months.slice(3, 6)).toEqual([cents(33333.33), cents(33333.33), cents(33333.34)]);
    expect(quarterTotals(months)).toEqual([0n, cents(100000), 0n, 0n]);
  });

  it('names the months of the plan year', () => {
    expect(monthKeys(2027)[0]).toBe('2027-01');
    expect(monthKeys(2027)[11]).toBe('2027-12');
    expect(monthLabel('2027-03')).toBe('Mar 2027');
  });

  it('reads typed amounts: empty is 0.00, negatives and words are refused', () => {
    expect(readAmount('')).toBe(0n);
    expect(readAmount('12.5k')).toBe(cents(12500));
    expect(readAmount('-5')).toBeNull();
    expect(readAmount('lots')).toBeNull();
  });

  it('says how much is left or too much (FR-021)', () => {
    const months = [...Array(11).fill(cents(50000)), cents(40000)];
    expect(balanceText(balanceOf(months, cents(600000)), 'GBP')).toBe(
      '590,000.00 of 600,000.00 GBP phased — 10,000.00 left',
    );
    expect(balanceText(balanceOf(Array(12).fill(cents(50000)), cents(590000)), 'GBP')).toBe(
      '600,000.00 of 590,000.00 GBP phased — 10,000.00 too much',
    );
    expect(balanceOf(Array(12).fill(cents(50000)), cents(600000)).kind).toBe('exact');
  });

  it('finds by how much a period is over (FR-024)', () => {
    expect(overBy('50000.00', '60000.00', null)).toBe(cents(10000));
    expect(overBy('50000.00', '10000.00', '55000.00')).toBe(cents(5000));
    expect(overBy('50000.00', '10000.00', '20000.00')).toBe(0n);
    expect(overBy(null, '10000.00', null)).toBe(0n);
  });
});
