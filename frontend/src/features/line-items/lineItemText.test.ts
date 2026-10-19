import { describe, expect, it } from 'vitest';
import {
  completenessText,
  customTotalText,
  exceededText,
  meterText,
  restartText,
  spreadEffectText,
  usdColumn,
} from './lineItemText';

describe('line-item texts', () => {
  it('counts the fields needed (FR-028)', () => {
    expect(completenessText(0)).toBe('All filled in');
    expect(completenessText(1)).toBe('1 field needed');
    expect(completenessText(3)).toBe('3 fields needed');
  });

  it('labels the USD column with the display currency (FR-005)', () => {
    expect(usdColumn('USD')).toBe('Planned cost (USD)');
    expect(usdColumn('EUR')).toBe('Planned cost (EUR)');
  });

  it('writes the balance texts (FR-002)', () => {
    expect(meterText('100,000.00', '120,000.00', 'USD')).toBe(
      '100,000.00 of 120,000.00 USD assigned',
    );
    expect(exceededText('500.00 USD')).toBe(
      "Line items exceed the activity's planned cost by 500.00 USD",
    );
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

  it('says what new dates do to the spreads, in singular and plural (FR-022)', () => {
    expect(spreadEffectText({ respread: 2, customKept: 1, customOutside: 0, notSpread: 0 })).toBe(
      'When you save, 2 line items are re-spread over the new dates and 1 custom spread is kept as typed.',
    );
    expect(spreadEffectText({ respread: 1, customKept: 0, customOutside: 0, notSpread: 0 })).toBe(
      'When you save, 1 line item is re-spread over the new dates.',
    );
    expect(spreadEffectText({ respread: 0, customKept: 2, customOutside: 1, notSpread: 0 })).toBe(
      "When you save, 2 custom spreads are kept as typed. 1 custom spread will have months outside the activity's dates.",
    );
    expect(spreadEffectText({ respread: 0, customKept: 0, customOutside: 0, notSpread: 3 })).toBe(
      "Without dates, 3 line items won't be spread until the activity has dates again.",
    );
    expect(spreadEffectText({ respread: 0, customKept: 0, customOutside: 0, notSpread: 0 })).toBe(
      null,
    );
  });
});
