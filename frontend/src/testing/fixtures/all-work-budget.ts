import type { AllWorkResult, Money } from '@/features/all-work';
import { gridResult } from './all-work';
import { usd } from './all-work-rows';
import { seedRows } from './all-work-seed';

const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];

/** The amounts of a row by month (1–12), in USD; the other months are empty. */
type ByMonth = Record<number, string>;

/** Twelve months, or four quarters summed from them, as the API answers them (null = "—"). */
function periodsOf(byMonth: ByMonth, period: 'month' | 'quarter'): (Money | null)[] {
  const cents = (month: number) => Math.round(Number(byMonth[month] ?? 0) * 100);
  const amount = (value: number, any: boolean) => (any ? usd((value / 100).toFixed(2)) : null);
  if (period === 'month') {
    return MONTHS.map((_, index) => {
      const value = byMonth[index + 1];
      return value ? usd(value) : null;
    });
  }
  return [0, 1, 2, 3].map((quarter) => {
    const months = [1, 2, 3].map((offset) => quarter * 3 + offset);
    return amount(
      months.reduce((sum, month) => sum + cents(month), 0),
      months.some((month) => byMonth[month] !== undefined),
    );
  });
}

function totalOf(byMonth: ByMonth): Money | null {
  const values = Object.values(byMonth);
  if (values.length === 0) return null;
  return usd(
    (values.reduce((sum, value) => sum + Math.round(Number(value) * 100), 0) / 100).toFixed(2),
  );
}

/**
 * GET /api/v1/all-work for Budget & cost on the seed story (FR-022 – FR-024): Helix launch event runs
 * March to May with Venue hire spread evenly and 3,000.00 unassigned; Helix social ads runs April
 * to June with line items 1,000.00 above its planned cost, which is not spread.
 */
export function budgetResult(period: 'month' | 'quarter' = 'month'): AllWorkResult {
  const helixEvent: ByMonth = { 3: '4000.00', 4: '4000.00', 5: '4000.00' };
  const venue: ByMonth = { 3: '3000.00', 4: '3000.00', 5: '3000.00' };
  const unassigned: ByMonth = { 3: '1000.00', 4: '1000.00', 5: '1000.00' };
  const social: ByMonth = { 4: '2000.00', 5: '2000.00', 6: '2000.00' };
  const helix: ByMonth = { 3: '4000.00', 4: '6000.00', 5: '6000.00', 6: '2000.00' };
  const byKey: Record<string, ByMonth> = {
    'P-1001': helix,
    'C-1001': helix,
    'I-1001': helix,
    'A-1001': helixEvent,
    'L-1001': venue,
    'A-1001:unassigned': unassigned,
    'A-1002': social,
    'L-1002': social,
  };
  const rows = seedRows([]).map((candidate) => {
    const byMonth = byKey[candidate.key] ?? {};
    return {
      ...candidate,
      periods: periodsOf(byMonth, period),
      total: totalOf(byMonth),
    };
  });
  return gridResult({
    view: 'budget',
    columns: [],
    periods:
      period === 'month'
        ? MONTHS.map((label, index) => ({
            key: `2026-${String(index + 1).padStart(2, '0')}`,
            label,
          }))
        : ['Q1', 'Q2', 'Q3', 'Q4'].map((label) => ({ key: label, label })),
    rows,
    totals: {
      activities: 3,
      cells: [],
      plannedCost: usd('17000.00'),
      planBudget: usd('2000000.00'),
      periods: periodsOf(helix, period),
      total: usd('18000.00'),
    },
  });
}

/**
 * GET /api/v1/all-work for the Calendar on the seed story (FR-025): the Grid's rows without line
 * items and unassigned rows, dated 18 Oct 2026.
 */
export function calendarResult(overrides: Partial<AllWorkResult> = {}): AllWorkResult {
  return gridResult({
    view: 'calendar',
    columns: [],
    rows: seedRows([]).filter(
      (candidate) => candidate.level !== 'lineItem' && candidate.level !== 'unassigned',
    ),
    totals: { ...gridResult().totals, cells: [] },
    ...overrides,
  });
}
