import fc from 'fast-check';
import { describe, expect, it } from 'vitest';
import { pastedValues } from './spread-paste';

describe('038 FR-042 (L-7): values pasted into a month', () => {
  it('splits tab- and line-separated values, reading money as typed', () => {
    expect(pastedValues('1000\t2000\t3000')).toEqual(['1000.00', '2000.00', '3000.00']);
    expect(pastedValues('$2,500.50\r\n€10\n')).toEqual(['2500.50', '10.00']);
    expect(pastedValues('12abc\t5')).toEqual(['12abc', '5.00']);
  });

  it('leaves a single value to the field', () => {
    expect(pastedValues('1000')).toBeNull();
    expect(pastedValues('  1,000  \n')).toBeNull();
  });

  it('never fails on any clipboard text, and gives each non-empty cell once (untrusted input)', () => {
    fc.assert(
      fc.property(fc.string(), (text) => {
        const values = pastedValues(text);
        const cells = text
          .split(/[\t\r\n]+/)
          .map((cell) => cell.trim())
          .filter((cell) => cell !== '');
        expect(values === null ? cells.length <= 1 : values.length === cells.length).toBe(true);
      }),
    );
  });
});
