import { describe, expect, it } from 'vitest';
import type { components } from '@/api/generated/schema';
import {
  activityMonths,
  addMonths,
  amountText,
  convert,
  customGridMonths,
  formatMonth,
  fromUsd,
  isOutsideDates,
  parseAmount,
  quarterColumns,
  spreadEffect,
  spreadMonths,
  toUsd,
  usdMonths,
} from './line-item-math';

type SpreadGrid = components['schemas']['SpreadGrid'];

const MAR_MAY = { startDate: '2026-03-01', endDate: '2026-05-31', ongoing: false };
const ONGOING = { startDate: '2026-03-15', endDate: null, ongoing: true };
const NO_DATES = { startDate: null, endDate: null, ongoing: false };

const usd = (amount: string) => ({ amount, currency: 'USD' });

// The shared example table of the 007 plan ("Business rules and their tests"); the Go tests use
// the same rows.
describe('the shared example table (D-142, SC-002, SC-003)', () => {
  it.each([
    ['46000.00', '0.92', 5_000_000n],
    ['10000.00', '0.92', 1_086_957n],
    ['15800.00', '0.79', 2_000_000n],
    ['20000.00', '1', 2_000_000n],
  ])('converts %s at %s to USD half-up (FR-011)', (amount, rate, cents) => {
    expect(toUsd(parseAmount(amount)!, rate)).toBe(cents);
  });

  it('converts 20,000.00 USD to 18,400.00 EUR at 0.92 (FR-012)', () => {
    expect(fromUsd(2_000_000n, '0.920000')).toBe(1_840_000n);
  });

  it.each([
    ['23000.00', '0.92', '0.79', 1_975_000n],
    ['5000.00', '1', '0.79', 395_000n],
    ['10000.00', '0.92', '1', 1_086_957n],
    ['15000.00', '0.79', '0.790000', 1_500_000n],
  ])(
    'converts %s at %s into the plan currency at %s half-up (016 data model §3.1)',
    (amount, from, to, cents) => {
      expect(convert(parseAmount(amount)!, from, to)).toBe(cents);
    },
  );

  it('spreads 10,000.00 evenly over three months with the rest in the last (FR-017)', () => {
    const spread = spreadMonths('evenly', 1_000_000n, MAR_MAY);
    expect(spread).toEqual({
      state: 'spread',
      months: [
        { month: '2026-03', cents: 333_333n },
        { month: '2026-04', cents: 333_333n },
        { month: '2026-05', cents: 333_334n },
      ],
    });
  });

  it('spreads 0.00 and 0.02 over three months', () => {
    expect(spreadMonths('evenly', 0n, MAR_MAY).months.map((m) => m.cents)).toEqual([0n, 0n, 0n]);
    expect(spreadMonths('evenly', 2n, MAR_MAY).months.map((m) => m.cents)).toEqual([0n, 0n, 2n]);
  });

  it('converts 10,000.00 EUR evenly over Mar–May to 3,623.18 / 3,623.18 / 3,623.21 USD (FR-020)', () => {
    const months = spreadMonths('evenly', 1_000_000n, MAR_MAY).months;
    expect(usdMonths(months, '0.92').map((m) => m.cents)).toEqual([362_318n, 362_318n, 362_321n]);
  });

  it('puts the rounding cent in the last month with an amount', () => {
    const months = [
      { month: '2026-03', cents: 100_000n },
      { month: '2026-04', cents: 100_001n },
      { month: '2026-05', cents: 0n },
    ];
    const converted = usdMonths(months, '0.92');
    expect(converted[2].cents).toBe(0n);
    expect(converted.reduce((sum, m) => sum + m.cents, 0n)).toBe(toUsd(200_001n, '0.92'));
  });

  it.each(['0.92', '0.79', '7.10', '83.50', '1'])(
    'adds the USD months up exactly to the USD amount at %s for 1–36 months (SC-002)',
    (rate) => {
      for (let count = 1; count <= 36; count += 1) {
        const dates = {
          startDate: '2026-01-01',
          endDate: `${addMonths('2026-01', count - 1)}-28`,
          ongoing: false,
        };
        const cents = 1_234_567n + BigInt(count);
        const months = usdMonths(spreadMonths('evenly', cents, dates).months, rate);
        expect(months).toHaveLength(count);
        expect(months.reduce((sum, m) => sum + m.cents, 0n)).toBe(toUsd(cents, rate));
      }
    },
  );

  it('rounds 1,234.565 to 1,234.57 (edge case "Rounding")', () => {
    expect(parseAmount('1,234.565')).toBe(123_457n);
  });
});

describe('parseAmount and texts', () => {
  it('reads typed amounts and refuses anything else', () => {
    expect(parseAmount('12500')).toBe(1_250_000n);
    expect(parseAmount('12,500.5')).toBe(1_250_050n);
    expect(parseAmount('-50')).toBe(-5_000n);
    expect(parseAmount('')).toBeNull();
    expect(parseAmount('ten')).toBeNull();
  });

  it('writes amounts with their code and a minus sign (D-143)', () => {
    expect(amountText(1_086_957n, 'USD')).toBe('10,869.57 USD');
    expect(amountText(-50_000n, 'USD')).toBe('-500.00 USD');
    expect(amountText('46000.00', 'EUR')).toBe('46,000.00 EUR');
  });

  it('writes months as people read them', () => {
    expect(formatMonth('2026-03')).toBe('Mar 2026');
    expect(addMonths('2026-12', 1)).toBe('2027-01');
    expect(addMonths('2026-01', -1)).toBe('2025-12');
  });
});

describe('activity months and spreads (FR-017, FR-021)', () => {
  it('lists the calendar months of the dates', () => {
    expect(activityMonths(MAR_MAY)).toEqual(['2026-03', '2026-04', '2026-05']);
    expect(
      activityMonths({ startDate: '2026-12-15', endDate: '2027-01-02', ongoing: false }),
    ).toEqual(['2026-12', '2027-01']);
  });

  it('has only the start month for an ongoing activity and none without dates', () => {
    expect(activityMonths(ONGOING)).toEqual(['2026-03']);
    expect(activityMonths(NO_DATES)).toEqual([]);
    expect(activityMonths({ startDate: '2026-03-01', endDate: null, ongoing: false })).toEqual([]);
  });

  it('puts everything in the first or last month', () => {
    expect(spreadMonths('firstMonth', 1_000_000n, MAR_MAY).months).toEqual([
      { month: '2026-03', cents: 1_000_000n },
    ]);
    expect(spreadMonths('lastMonth', 1_000_000n, MAR_MAY).months).toEqual([
      { month: '2026-05', cents: 1_000_000n },
    ]);
  });

  it('spreads Evenly and All in last month like All in first month when ongoing', () => {
    expect(spreadMonths('evenly', 1_000_000n, ONGOING).months).toEqual([
      { month: '2026-03', cents: 1_000_000n },
    ]);
    expect(spreadMonths('lastMonth', 1_000_000n, ONGOING).months).toEqual([
      { month: '2026-03', cents: 1_000_000n },
    ]);
  });

  it('does not spread without dates', () => {
    expect(spreadMonths('evenly', 1_000_000n, NO_DATES)).toEqual({
      state: 'notSpread',
      months: [],
    });
  });

  it('marks months outside the dates', () => {
    expect(isOutsideDates('2026-02', MAR_MAY)).toBe(true);
    expect(isOutsideDates('2026-04', MAR_MAY)).toBe(false);
    expect(isOutsideDates('2026-06', MAR_MAY)).toBe(true);
    expect(isOutsideDates('2027-06', ONGOING)).toBe(false);
    expect(isOutsideDates('2026-02', ONGOING)).toBe(true);
    expect(isOutsideDates('2026-02', NO_DATES)).toBe(false);
  });
});

describe('customGridMonths (D-152)', () => {
  it('starts with the activity months and keeps stored months outside them', () => {
    expect(customGridMonths(MAR_MAY, 2026)).toEqual(['2026-03', '2026-04', '2026-05']);
    expect(customGridMonths(MAR_MAY, 2026, ['2026-02', '2026-04'])).toEqual([
      '2026-02',
      '2026-03',
      '2026-04',
      '2026-05',
    ]);
  });

  it('runs to December of the plan year for an ongoing activity (FR-021)', () => {
    expect(
      customGridMonths({ startDate: '2026-10-01', endDate: null, ongoing: true }, 2026),
    ).toEqual(['2026-10', '2026-11', '2026-12']);
  });

  it('offers the plan year without dates', () => {
    const months = customGridMonths(NO_DATES, 2026);
    expect(months).toHaveLength(12);
    expect(months[0]).toBe('2026-01');
    expect(months[11]).toBe('2026-12');
  });
});

describe('quarterColumns (FR-003)', () => {
  it('adds the months of each quarter and keeps the No dates yet column', () => {
    const grid: SpreadGrid = {
      columns: [
        { key: '2026-03', month: '2026-03', outsideActivityDates: true },
        { key: '2026-04', month: '2026-04', outsideActivityDates: false },
        { key: '2026-05', month: '2026-05', outsideActivityDates: false },
        { key: 'noDates', month: null, outsideActivityDates: false },
      ],
      rows: [
        {
          lineItemId: 'L-1',
          name: 'Agency fee',
          cells: [usd('100.00'), usd('200.00'), null, usd('5.00')],
          total: usd('305.00'),
        },
      ],
      totals: [usd('100.00'), usd('200.00'), null, usd('5.00')],
      total: usd('305.00'),
    };
    const quarters = quarterColumns(grid);
    expect(quarters.columns.map((c) => c.label)).toEqual(['Q1 2026', 'Q2 2026', 'No dates yet']);
    expect(quarters.columns.map((c) => c.outsideActivityDates)).toEqual([true, false, false]);
    expect(quarters.rows[0].cells).toEqual([usd('100.00'), usd('200.00'), usd('5.00')]);
    expect(quarters.totals).toEqual([usd('100.00'), usd('200.00'), usd('5.00')]);
  });
});

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
});
