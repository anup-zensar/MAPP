import { describe, expect, it } from 'vitest';
import { completenessText, customTotalText, restartText } from './line-item-text';

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

  it('says why approval restarts (FR-031)', () => {
    expect(restartText('Launch film', 'approved')).toBe(
      "'Launch film' is approved. Saving sends it back to Created, and it must be submitted for approval again.",
    );
    expect(restartText('Launch film', 'submitted')).toMatch(
      /^'Launch film' is submitted for approval\./,
    );
  });
});
