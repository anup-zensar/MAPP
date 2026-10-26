import { describe, expect, it } from 'vitest';
import { spreadEffect, spreadEffectText } from './spread-effect';

const NO_DATES = { startDate: null, endDate: null, ongoing: false };

describe('spreadEffect (FR-022)', () => {
  it('counts re-spread and kept custom spreads with their outside months', () => {
    const items = [
      { method: 'evenly' as const, months: [] },
      { method: 'lastMonth' as const, months: [] },
      { method: 'custom' as const, months: ['2026-03', '2026-04'] },
    ];
    expect(
      spreadEffect(items, { startDate: '2026-04-01', endDate: '2026-06-30', ongoing: false }),
    ).toEqual({ respread: 2, customKept: 1, customOutside: 1, notSpread: 0 });
    expect(spreadEffect(items, NO_DATES)).toEqual({
      respread: 0,
      customKept: 1,
      customOutside: 0,
      notSpread: 2,
    });
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
