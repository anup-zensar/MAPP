import { describe, expect, it } from 'vitest';
import { completenessText, customTotalText } from './line-item-text';

describe('line-item texts', () => {
  it('counts the fields needed (FR-028)', () => {
    expect(completenessText(0)).toBe('All filled in');
    expect(completenessText(1)).toBe('1 field needed');
    expect(completenessText(3)).toBe('3 fields needed');
  });

  it('writes the custom total (FR-018)', () => {
    expect(customTotalText('12,000.00', '10,000.00', 'EUR', { sign: -1, amount: '2,000.00' })).toBe(
      '12,000.00 of 10,000.00 EUR spread — 2,000.00 EUR too much',
    );
    expect(customTotalText('9,500.00', '10,000.00', 'EUR', { sign: 1, amount: '500.00' })).toBe(
      '9,500.00 of 10,000.00 EUR spread — 500.00 EUR left',
    );
    expect(customTotalText('10,000.00', '10,000.00', 'EUR', { sign: 0, amount: '0.00' })).toBe(
      '10,000.00 of 10,000.00 EUR spread — adds up',
    );
  });
});
