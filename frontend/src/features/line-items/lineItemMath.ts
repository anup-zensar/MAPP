/**
 * The line-item rules the browser repeats for instant feedback (007 plan, "Calculations in the
 * browser", D-142): conversion, spreads over the activity's months, quarters and what new activity
 * dates do to the spreads. Amounts are BigInt cents, never JavaScript numbers. The API recalculates
 * everything on save, and its answer is what the pages show; the Go and TypeScript tests share one
 * example table.
 */
import type { components } from '@/api/schema';
import { formatAmount, formatCents, parseMoneyInput, toCents } from '@/lib/decimal';

export type SpreadMethod = components['schemas']['SpreadMethod'];
type SpreadGrid = components['schemas']['SpreadGrid'];
type Money = components['schemas']['Money'];

/** The activity's dates as the line-item screens know them ("YYYY-MM-DD" or null). */
export interface ActivityDates {
  startDate: string | null;
  endDate: string | null;
  ongoing: boolean;
}

/** One month's amount, in cents. */
export interface MonthCents {
  /** "2026-03". */
  month: string;
  cents: bigint;
}

const MILLION = 1_000_000n;
const MONTH_NAMES = [
  'Jan',
  'Feb',
  'Mar',
  'Apr',
  'May',
  'Jun',
  'Jul',
  'Aug',
  'Sep',
  'Oct',
  'Nov',
  'Dec',
];

/**
 * Reads a typed amount in cents: "12500", "12,500.00", "12.5k"; more than two decimals are rounded
 * half-up to the cent (1,234.565 → 1,234.57). Null for anything that is not an amount.
 */
export function parseAmount(text: string): bigint | null {
  const parsed = parseMoneyInput(text);
  return parsed ? toCents(parsed.amount) : null;
}

/** "10,869.57 USD", "-500.00 USD" (D-143): cents or an API amount with the currency code after it. */
export function amountText(value: bigint | string, code: string): string {
  const amount = typeof value === 'bigint' ? formatCents(value) : value;
  return `${formatAmount(amount)} ${code}`;
}

/** "10,869.57 USD" for an API Money. */
export function moneyText(money: Money): string {
  return amountText(money.amount, money.currency);
}

/** "10,869.57": cents written as the money field shows them, without the code. */
export function centsInput(cents: bigint): string {
  return formatAmount(formatCents(cents));
}

/** "Mar 2026" for "2026-03". */
export function formatMonth(month: string): string {
  const [year, number] = month.split('-');
  return `${MONTH_NAMES[Number(number) - 1]} ${year}`;
}

/** The month `count` months after `month` (before it when negative): ("2026-12", 1) → "2027-01". */
export function addMonths(month: string, count: number): string {
  const index = Number(month.slice(0, 4)) * 12 + Number(month.slice(5, 7)) - 1 + count;
  const year = Math.floor(index / 12);
  return `${String(year).padStart(4, '0')}-${String((index % 12) + 1).padStart(2, '0')}`;
}

/**
 * Planned cost in USD = local amount ÷ rate (units of the currency per 1 USD), rounded half-up to
 * the cent: 10,000.00 EUR at 0.92 → 10,869.57 USD (domain model §4.3).
 */
export function toUsd(cents: bigint, rate: string): bigint {
  return divideHalfUp(cents * MILLION, rateMicros(rate));
}

/** USD to the currency: 20,000.00 USD at 0.92 → 18,400.00 EUR (FR-012), half-up to the cent. */
export function fromUsd(usdCents: bigint, rate: string): bigint {
  return divideHalfUp(usdCents * rateMicros(rate), MILLION);
}

/**
 * An amount from one currency into another (016 data model §3.1, `money.Convert`): amount × to ÷
 * from, both rates in units of the currency per 1 USD, rounded half-up to the cent; the same rate on
 * both sides returns it unchanged. 23,000.00 EUR at 0.92 → 19,750.00 GBP at 0.79.
 */
export function convert(cents: bigint, fromRate: string, toRate: string): bigint {
  const from = rateMicros(fromRate);
  const to = rateMicros(toRate);
  return from === to ? cents : divideHalfUp(cents * to, from);
}

/**
 * The months of the activity's dates (data model §3.2): every month from the start month to the end
 * month; only the start month for an ongoing activity; none without dates ("No dates yet").
 */
export function activityMonths(dates: ActivityDates): string[] {
  if (!dates.startDate) {
    return [];
  }
  const first = dates.startDate.slice(0, 7);
  if (dates.ongoing) {
    return [first];
  }
  if (!dates.endDate) {
    return [];
  }
  const last = dates.endDate.slice(0, 7);
  const months: string[] = [];
  for (let month = first; month <= last; month = addMonths(month, 1)) {
    months.push(month);
  }
  return months;
}

/** True when the activity has dates and the month is not one of them (FR-018). */
export function isOutsideDates(month: string, dates: ActivityDates): boolean {
  if (activityMonths(dates).length === 0 || !dates.startDate) {
    return false;
  }
  if (month < dates.startDate.slice(0, 7)) {
    return true;
  }
  return !dates.ongoing && month > (dates.endDate ?? '').slice(0, 7);
}

/** A spread worked out in the browser: its months, or not spread without activity dates. */
export interface SpreadResult {
  state: 'spread' | 'notSpread';
  months: MonthCents[];
}

/**
 * The month amounts of Evenly, All in first month and All in last month (data model §3.3): even
 * shares rounded down to the cent with the rest in the last month; an ongoing activity puts
 * everything in its first month (FR-021); without dates nothing is spread.
 */
export function spreadMonths(
  method: Exclude<SpreadMethod, 'custom'>,
  cents: bigint,
  dates: ActivityDates,
): SpreadResult {
  const months = activityMonths(dates);
  if (months.length === 0) {
    return { state: 'notSpread', months: [] };
  }
  if (dates.ongoing || method === 'firstMonth') {
    return { state: 'spread', months: [{ month: months[0], cents }] };
  }
  if (method === 'lastMonth') {
    return { state: 'spread', months: [{ month: months[months.length - 1], cents }] };
  }
  return { state: 'spread', months: splitEvenly(cents, months) };
}

/** cents over the months: each share rounded down to the cent, the rest in the last month. */
export function splitEvenly(cents: bigint, months: string[]): MonthCents[] {
  const count = BigInt(months.length);
  const share = cents / count;
  return months.map((month, index) => ({
    month,
    cents: index === months.length - 1 ? cents - share * (count - 1n) : share,
  }));
}

/**
 * Month amounts in USD (data model §3.4, FR-020): each month converted half-up, and the difference
 * to the USD value of their sum in the last month with an amount, so they add up exactly.
 */
export function usdMonths(months: MonthCents[], rate: string): MonthCents[] {
  const usd = months.map(({ month, cents }) => ({ month, cents: toUsd(cents, rate) }));
  const lastWithAmount = months.findLastIndex(({ cents }) => cents !== 0n);
  if (lastWithAmount < 0) {
    return usd;
  }
  const target = toUsd(
    months.reduce((sum, { cents }) => sum + cents, 0n),
    rate,
  );
  const converted = usd.reduce((sum, { cents }) => sum + cents, 0n);
  usd[lastWithAmount] = {
    month: usd[lastWithAmount].month,
    cents: usd[lastWithAmount].cents + target - converted,
  };
  return usd;
}

/** One column of the grid after Monthly or Quarterly. */
export interface GridColumn {
  key: string;
  /** "Mar 2026", "Q1 2026" or "No dates yet". */
  label: string;
  outsideActivityDates: boolean;
  noDates: boolean;
}

/** The grid as shown: columns, each row's cells and the totals, as API amounts or null. */
export interface ShownGrid {
  columns: GridColumn[];
  rows: { lineItemId: string; name: string; cells: (Money | null)[]; total: Money }[];
  totals: (Money | null)[];
  total: Money;
}

/** The API's grid by month, with its column labels. */
export function monthColumns(grid: SpreadGrid): ShownGrid {
  return {
    columns: grid.columns.map((column) => ({
      key: column.key,
      label: column.month ? formatMonth(column.month) : 'No dates yet',
      outsideActivityDates: column.outsideActivityDates,
      noDates: column.month === null,
    })),
    rows: grid.rows,
    totals: grid.totals,
    total: grid.total,
  };
}

/**
 * The grid by calendar quarter (data model §3.8): the month columns of each quarter added up ("Q1
 * 2026"), the No dates yet column kept; a quarter is outside the activity's dates when all of its
 * months are. Display only.
 */
export function quarterColumns(grid: SpreadGrid): ShownGrid {
  const groups: { key: string; label: string; indexes: number[]; noDates: boolean }[] = [];
  grid.columns.forEach((column, index) => {
    const key = column.month ? quarterOf(column.month) : column.key;
    const group = groups.find((candidate) => candidate.key === key);
    if (group) {
      group.indexes.push(index);
    } else {
      groups.push({
        key,
        label: column.month ? key.replace(/^(\d{4})-(Q\d)$/, '$2 $1') : 'No dates yet',
        indexes: [index],
        noDates: column.month === null,
      });
    }
  });
  const currency = grid.total.currency;
  const sum = (cells: (Money | null)[], indexes: number[]): Money | null => {
    const amounts = indexes.map((index) => cells[index]).filter((cell) => cell !== null);
    if (amounts.length === 0) {
      return null;
    }
    const cents = amounts.reduce((total, cell) => total + toCents(cell.amount), 0n);
    return { amount: formatCents(cents), currency };
  };
  return {
    columns: groups.map((group) => ({
      key: group.key,
      label: group.label,
      outsideActivityDates: group.indexes.every(
        (index) => grid.columns[index].outsideActivityDates,
      ),
      noDates: group.noDates,
    })),
    rows: grid.rows.map((row) => ({
      ...row,
      cells: groups.map((group) => sum(row.cells, group.indexes)),
    })),
    totals: groups.map((group) => sum(grid.totals, group.indexes)),
    total: grid.total,
  };
}

/**
 * The months a new custom grid starts with (D-152): the activity's months; for an ongoing activity
 * its start month to December of the plan's year; without dates the plan's twelve months — and
 * every month the line item already has, sorted.
 */
export function customGridMonths(
  dates: ActivityDates,
  planYear: number,
  storedMonths: string[] = [],
): string[] {
  let months = activityMonths(dates);
  if (months.length === 0) {
    months = Array.from({ length: 12 }, (_, index) => addMonths(`${planYear}-01`, index));
  } else if (dates.ongoing) {
    const december = `${planYear}-12`;
    for (let month = addMonths(months[0], 1); month <= december; month = addMonths(month, 1)) {
      months.push(month);
    }
  }
  return [...new Set([...months, ...storedMonths])].sort();
}

/** The line items' spreads as SpreadEffectNote needs them. */
export interface SpreadFacts {
  method: SpreadMethod;
  /** The months of a custom spread. */
  months: string[];
}

/** What new activity dates do to its line items (FR-022, D-150). */
export interface SpreadEffect {
  /** Evenly, first or last month: spread again over the new dates. */
  respread: number;
  /** Custom spreads: kept as typed. */
  customKept: number;
  /** Custom spreads with a month outside the new dates. */
  customOutside: number;
  /** Evenly, first or last month while the new dates leave the activity without dates. */
  notSpread: number;
}

/** Counts what new dates do to the given spreads (D-150). */
export function spreadEffect(items: SpreadFacts[], dates: ActivityDates): SpreadEffect {
  const hasDates = activityMonths(dates).length > 0;
  const effect: SpreadEffect = { respread: 0, customKept: 0, customOutside: 0, notSpread: 0 };
  for (const item of items) {
    if (item.method === 'custom') {
      effect.customKept += 1;
      if (item.months.some((month) => isOutsideDates(month, dates))) {
        effect.customOutside += 1;
      }
    } else if (hasDates) {
      effect.respread += 1;
    } else {
      effect.notSpread += 1;
    }
  }
  return effect;
}

/** A rate ("0.92", "83.500000") in millionths. */
function rateMicros(rate: string): bigint {
  const [whole, fraction = ''] = rate.trim().split('.');
  return BigInt(whole || '0') * MILLION + BigInt(fraction.slice(0, 6).padEnd(6, '0'));
}

/** numerator ÷ divisor rounded half away from zero (half-up for positive amounts). */
function divideHalfUp(numerator: bigint, divisor: bigint): bigint {
  const negative = numerator < 0n !== divisor < 0n;
  const n = numerator < 0n ? -numerator : numerator;
  const d = divisor < 0n ? -divisor : divisor;
  const result = (n * 2n + d) / (d * 2n);
  return negative ? -result : result;
}

/** "2026-Q1" for "2026-03". */
function quarterOf(month: string): string {
  return `${month.slice(0, 4)}-Q${Math.floor((Number(month.slice(5, 7)) - 1) / 3) + 1}`;
}
