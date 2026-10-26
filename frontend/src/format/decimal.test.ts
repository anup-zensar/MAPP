import { describe, expect, it } from 'vitest';
import {
  formatAmount,
  formatCents,
  formatHundredths,
  formatMoneyText,
  formatPercent,
  formatRate,
  formatWhole,
  inverseRate,
  parseDecimalInput,
  parseMoneyInput,
  parseWholeInput,
  percentSplitEvenly,
  toCents,
  toHundredths,
} from './decimal';

describe('parseMoneyInput', () => {
  it.each([
    ['12500', '12500.00'],
    ['12,500.00', '12500.00'],
    ['  12,500.5 ', '12500.50'],
    ['12.5k', '12500.00'],
    ['12.5 K', '12500.00'],
    ['1.2m', '1200000.00'],
    ['0', '0.00'],
    ['-250', '-250.00'],
    ['999,999,999,999,999.99', '999999999999999.99'],
  ])('reads "%s" as %s (design system §5 money input)', (typed, amount) => {
    expect(parseMoneyInput(typed)).toEqual({ amount, rounded: false });
  });

  it('rounds more than two decimals half-up to the cent and says so (D-190)', () => {
    expect(parseMoneyInput('1234.565')).toEqual({ amount: '1234.57', rounded: true });
    expect(parseMoneyInput('1234.564')).toEqual({ amount: '1234.56', rounded: true });
    expect(parseMoneyInput('-0.005')).toEqual({ amount: '-0.01', rounded: true });
    expect(parseMoneyInput('0.001')).toEqual({ amount: '0.00', rounded: true });
  });

  it.each(['', 'abc', '12.500,00', '1,25', '1,2345', '12.', '.5', '--5', '12 500', '1e5'])(
    'refuses "%s"',
    (typed) => {
      expect(parseMoneyInput(typed)).toBeNull();
    },
  );

  it('refuses amounts beyond the API limit (D-010)', () => {
    expect(parseMoneyInput('1000000000000000')).toBeNull();
    expect(parseMoneyInput('1000000000000k')).toBeNull();
  });
});

describe('parseDecimalInput and parseWholeInput', () => {
  it('returns the number without grouping, keeping what was typed', () => {
    expect(parseDecimalInput('1,234.5', 2)).toBe('1234.5');
    expect(parseDecimalInput('2', 2)).toBe('2');
    expect(parseDecimalInput('0.920000', 6)).toBe('0.920000');
    expect(parseDecimalInput('250,000', 0)).toBe('250000');
    expect(parseDecimalInput('-12', 0)).toBe('-12');
  });

  it('refuses more decimals than allowed instead of rounding', () => {
    expect(parseDecimalInput('2.005', 2)).toBeNull();
    expect(parseDecimalInput('12.0', 0)).toBeNull();
    expect(parseDecimalInput('0.1234567', 6)).toBeNull();
    expect(parseDecimalInput('two', 2)).toBeNull();
  });

  it('reads whole numbers as BigInt', () => {
    expect(parseWholeInput('250,000')).toBe(250000n);
    expect(parseWholeInput('999999999999')).toBe(999999999999n);
    expect(parseWholeInput('2.5')).toBeNull();
  });
});

describe('formatting', () => {
  it('groups whole numbers and amounts exactly', () => {
    expect(formatWhole(250000n)).toBe('250,000');
    expect(formatWhole('1217')).toBe('1,217');
    expect(formatAmount('1234.5')).toBe('1,234.50');
    expect(formatAmount('12345678901234567.89')).toBe('12,345,678,901,234,567.89');
    expect(formatAmount('-12500')).toBe('-12,500.00');
  });

  it('writes money with the code after the amount, as the API does (002 Money.Format)', () => {
    expect(formatMoneyText({ amount: '120000', currency: 'USD' })).toBe('120,000.00 USD');
    expect(formatMoneyText({ amount: '46000.00', currency: 'EUR' })).toBe('46,000.00 EUR');
  });

  it('writes percentages with two decimals and rates with two to six', () => {
    expect(formatPercent('2')).toBe('2.00 %');
    expect(formatPercent('33.34')).toBe('33.34 %');
    expect(formatRate('0.920000')).toBe('0.92');
    expect(formatRate('150')).toBe('150.00');
    expect(formatRate('1.086957')).toBe('1.086957');
  });
});

describe('inverseRate', () => {
  it('is 1 ÷ rate rounded half-up to six decimals (FR-028, spec US4)', () => {
    expect(inverseRate('0.92')).toBe('1.086957');
    expect(inverseRate('150')).toBe('0.006667');
    expect(inverseRate('0.95')).toBe('1.052632');
    expect(inverseRate('1')).toBe('1');
  });

  it('has no answer for a rate that is not above 0', () => {
    expect(inverseRate('0')).toBeNull();
    expect(inverseRate('abc')).toBeNull();
  });
});

describe('cents and hundredths', () => {
  it('turns API amounts into cents and back', () => {
    expect(toCents('1234.5')).toBe(123450n);
    expect(toCents('-12')).toBe(-1200n);
    expect(formatCents(123450n)).toBe('1234.50');
    expect(formatCents(-5n)).toBe('-0.05');
    expect(formatCents(0n)).toBe('0.00');
    expect(() => toCents('1.234')).toThrow();
  });

  it('turns percentages into hundredths and back', () => {
    expect(toHundredths('33.33')).toBe(3333n);
    expect(toHundredths('60')).toBe(6000n);
    expect(toHundredths('1.234')).toBeNull();
    expect(formatHundredths(10000n)).toBe('100.00');
  });

  it('splits 100 % evenly, the remainder in the last share (R10)', () => {
    expect(percentSplitEvenly(3)).toEqual(['33.33', '33.33', '33.34']);
    expect(percentSplitEvenly(2)).toEqual(['50.00', '50.00']);
    expect(percentSplitEvenly(1)).toEqual(['100.00']);
    expect(percentSplitEvenly(0)).toEqual([]);
  });
});
