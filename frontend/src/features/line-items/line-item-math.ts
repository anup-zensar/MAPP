import type { components } from '@/api/generated/schema';
import { formatAmount, formatCents, parseMoneyInput, toCents } from '@/format/decimal';
import { formatMonth as formatMonthLabel } from '@/format/intl';
import type { DateFormat } from '@/format/intl';

export type SpreadMethod = components['schemas']['SpreadMethod'];
type Money = components['schemas']['Money'];

export interface ActivityDates {
  startDate: string | null;
  endDate: string | null;
  ongoing: boolean;
}

export interface MonthCents {
  month: string;
  cents: bigint;
}

const MILLION = 1_000_000n;

/** "12500", "12,500.00", "12.5k" in cents; more decimals round half-up (1,234.565 → 1,234.57). */
export function parseAmount(text: string): bigint | null {
  const parsed = parseMoneyInput(text);
  return parsed ? toCents(parsed.amount) : null;
}

export function amountText(value: bigint | string, code: string): string {
  const amount = typeof value === 'bigint' ? formatCents(value) : value;
  return `${formatAmount(amount)} ${code}`;
}

export function moneyText(money: Money): string {
  return amountText(money.amount, money.currency);
}

export function centsInput(cents: bigint): string {
  return formatAmount(formatCents(cents));
}

export function formatMonth(month: string, format?: DateFormat): string {
  return formatMonthLabel(month, format);
}

export function addMonths(month: string, count: number): string {
  const index = Number(month.slice(0, 4)) * 12 + Number(month.slice(5, 7)) - 1 + count;
  const year = Math.floor(index / 12);
  return `${String(year).padStart(4, '0')}-${String((index % 12) + 1).padStart(2, '0')}`;
}

/** Local amount ÷ rate (units per 1 USD), half-up: 10,000.00 EUR at 0.92 → 10,869.57 USD. */
export function toUsd(cents: bigint, rate: string): bigint {
  return divideHalfUp(cents * MILLION, rateMicros(rate));
}

export function fromUsd(usdCents: bigint, rate: string): bigint {
  return divideHalfUp(usdCents * rateMicros(rate), MILLION);
}

export function convert(cents: bigint, fromRate: string, toRate: string): bigint {
  const from = rateMicros(fromRate);
  const to = rateMicros(toRate);
  return from === to ? cents : divideHalfUp(cents * to, from);
}

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

export function isOutsideDates(month: string, dates: ActivityDates): boolean {
  if (activityMonths(dates).length === 0 || !dates.startDate) {
    return false;
  }
  if (month < dates.startDate.slice(0, 7)) {
    return true;
  }
  return !dates.ongoing && month > (dates.endDate ?? '').slice(0, 7);
}

export interface SpreadResult {
  state: 'spread' | 'notSpread';
  months: MonthCents[];
}

export function spreadMonths(
  method: Exclude<SpreadMethod, 'custom'>,
  cents: bigint,
  dates: ActivityDates,
): SpreadResult {
  const months = activityMonths(dates);
  const first = months[0];
  const last = months.at(-1);
  if (first === undefined || last === undefined) {
    return { state: 'notSpread', months: [] };
  }
  if (dates.ongoing || method === 'firstMonth') {
    return { state: 'spread', months: [{ month: first, cents }] };
  }
  if (method === 'lastMonth') {
    return { state: 'spread', months: [{ month: last, cents }] };
  }
  return { state: 'spread', months: splitEvenly(cents, months) };
}

export function splitEvenly(cents: bigint, months: string[]): MonthCents[] {
  const count = BigInt(months.length);
  const share = cents / count;
  return months.map((month, index) => ({
    month,
    cents: index === months.length - 1 ? cents - share * (count - 1n) : share,
  }));
}

/** Each month converted half-up; the rounding difference goes to the last month with an amount. */
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
  const difference = target - usd.reduce((sum, { cents }) => sum + cents, 0n);
  return usd.map((month, index) =>
    index === lastWithAmount ? { month: month.month, cents: month.cents + difference } : month,
  );
}

export function customGridMonths(
  dates: ActivityDates,
  planYear: number,
  storedMonths: string[] = [],
): string[] {
  let months = activityMonths(dates);
  const first = months[0];
  if (first === undefined) {
    months = Array.from({ length: 12 }, (_, index) => addMonths(`${planYear}-01`, index));
  } else if (dates.ongoing) {
    const december = `${planYear}-12`;
    for (let month = addMonths(first, 1); month <= december; month = addMonths(month, 1)) {
      months.push(month);
    }
  }
  return [...new Set([...months, ...storedMonths])].sort();
}

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
