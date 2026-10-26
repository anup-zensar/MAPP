/**
 * Exact decimals in the web app (feature 002 plan, "Front-end rules mirrored from Go"): reading what
 * people type into money, percentage, rate and whole-number fields, and the little arithmetic forms
 * need for instant feedback. BigInt throughout, never a JavaScript number for money (constitution VI).
 * The API stays the judge: what these helpers compute is shown, never trusted.
 */

/** The largest amount the API accepts: 999,999,999,999,999.99 (data-model 002 §1). */
const MAX_CENTS = 99_999_999_999_999_999n;

/** "1234", "1,234" or "-1,234.5": an optional minus, digits (grouped by commas or not), decimals. */
const NUMBER = /^(-)?(\d{1,3}(?:,\d{3})+|\d+)(?:\.(\d+))?$/;

const grouped = new Intl.NumberFormat('en-US');
const twoDecimals = new Intl.NumberFormat('en-US', {
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
});
const rateDecimals = new Intl.NumberFormat('en-US', {
  minimumFractionDigits: 2,
  maximumFractionDigits: 6,
});

/** An amount read from a money field. */
export interface ParsedAmount {
  /** The amount as the API takes it: two decimals, no grouping ("12500.00", "-250.00"). */
  amount: string;
  /** More than two decimals were typed and rounded half-up to the cent; the form says so (D-190). */
  rounded: boolean;
}

/**
 * Reads a typed amount: "12500", "12,500.00", "12.5k" (thousand), "1.2m" (million), "-250".
 * More than two decimals are rounded half-up to the cent (half away from zero below 0) and marked
 * `rounded` (D-190). Anything else, or an amount beyond the API's limit, gives null.
 */
export function parseMoneyInput(text: string): ParsedAmount | null {
  const suffix = /\s*([km])$/i.exec(text.trim());
  const multiplier = !suffix ? 1n : suffix[1].toLowerCase() === 'k' ? 1_000n : 1_000_000n;
  const number = readNumber(suffix ? text.trim().slice(0, suffix.index) : text);
  if (!number) {
    return null;
  }
  // The typed value is digits ÷ 10^decimals; in cents that is digits × 100 ÷ 10^decimals.
  const exact = number.digits * multiplier * 100n;
  const divisor = 10n ** BigInt(number.decimals);
  const cents = roundHalfUp(exact, divisor);
  if (cents > MAX_CENTS) {
    return null;
  }
  const signed = number.negative ? -cents : cents;
  return { amount: formatCents(signed), rounded: exact % divisor !== 0n };
}

/**
 * Reads a typed decimal with at most `decimals` decimals ("1,234.5" → "1234.5"; "250,000" with 0
 * decimals → "250000"): the text the API takes, grouping removed. More decimals, or anything that
 * is not a number, give null — percentages and rates are never rounded silently.
 */
export function parseDecimalInput(text: string, decimals: 0 | 2 | 6): string | null {
  const number = readNumber(text);
  if (!number || number.decimals > decimals) {
    return null;
  }
  const match = NUMBER.exec(text.trim());
  // readNumber accepted the text, so the pattern matches.
  const [, minus = '', integer, fraction] = match as RegExpExecArray;
  const plain = `${minus}${integer.replaceAll(',', '')}`;
  return fraction === undefined ? plain : `${plain}.${fraction}`;
}

/** Reads a typed whole number ("250,000" → 250000n); null for anything else. */
export function parseWholeInput(text: string): bigint | null {
  const value = parseDecimalInput(text, 0);
  return value === null ? null : BigInt(value);
}

/** "250,000": a whole number with grouping. */
export function formatWhole(value: bigint | number | string): string {
  return grouped.format(value as Intl.StringNumericLiteral);
}

/** "1,234.50": an amount from the API with grouping and two decimals, exactly (no currency). */
export function formatAmount(amount: string): string {
  return twoDecimals.format(amount as Intl.StringNumericLiteral);
}

/**
 * "1,234.50 EUR" — also for US dollars: the code after the amount, as the API writes amounts in its
 * messages (Go's Money.Format). Screens that show "$1,234.50" use formatMoney from intl.ts.
 */
export function formatMoneyText(money: { amount: string; currency: string }): string {
  return `${formatAmount(money.amount)} ${money.currency}`;
}

/** "2.00 %": a percentage from the API (model rates, split shares). */
export function formatPercent(percent: string): string {
  return `${twoDecimals.format(percent as Intl.StringNumericLiteral)} %`;
}

/** "0.92", "150.00", "1.086957": an exchange rate with two to six decimals. */
export function formatRate(rate: string): string {
  return rateDecimals.format(rate as Intl.StringNumericLiteral);
}

/**
 * 1 ÷ rate, rounded half-up to six decimals, written without trailing zeros as the API writes rates:
 * "0.92" → "1.086957", "150" → "0.006667", "1" → "1". Null for a rate that is not above 0.
 */
export function inverseRate(rate: string): string | null {
  const micros = toScaled(rate, 6);
  if (micros === null || micros <= 0n) {
    return null;
  }
  return withoutTrailingZeros(fromScaled(roundHalfUp(10n ** 12n, micros), 6));
}

/** An API amount ("1234.5", "-12.00") in cents. Throws for anything that is not one. */
export function toCents(amount: string): bigint {
  const cents = toScaled(amount, 2);
  if (cents === null) {
    throw new Error(`Not an amount with at most two decimals: "${amount}"`);
  }
  return cents;
}

/** Cents as the API writes an amount: "1234.50", "-0.05". */
export function formatCents(cents: bigint): string {
  return fromScaled(cents, 2);
}

/** A percentage ("33.33") in hundredths of a percent (3333n); null when it is not one. */
export function toHundredths(percent: string): bigint | null {
  return toScaled(percent, 2);
}

/** Hundredths of a percent as the API writes a percentage: 3333n → "33.33". */
export function formatHundredths(hundredths: bigint): string {
  return fromScaled(hundredths, 2);
}

/**
 * 100.00 % split into `count` shares, each rounded down to the hundredth and the remainder in the
 * last one: 3 → "33.33", "33.33", "33.34" (Go's money.PercentSplitEvenly, research R10).
 */
export function percentSplitEvenly(count: number): string[] {
  if (count < 1) {
    return [];
  }
  const n = BigInt(count);
  const share = 10_000n / n;
  const last = 10_000n - share * (n - 1n);
  return Array.from({ length: count }, (_, index) =>
    formatHundredths(index === count - 1 ? last : share),
  );
}

/** A decimal text with at most `scale` decimals as a whole number of 10^-scale units. */
function toScaled(text: string, scale: number): bigint | null {
  const number = readNumber(text);
  if (!number || number.decimals > scale) {
    return null;
  }
  const value = number.digits * 10n ** BigInt(scale - number.decimals);
  return number.negative ? -value : value;
}

/** A whole number of 10^-scale units as a decimal text with exactly `scale` decimals. */
function fromScaled(value: bigint, scale: number): string {
  const negative = value < 0n;
  const digits = (negative ? -value : value).toString().padStart(scale + 1, '0');
  const integer = digits.slice(0, digits.length - scale);
  const fraction = digits.slice(digits.length - scale);
  return `${negative ? '-' : ''}${integer}${scale > 0 ? `.${fraction}` : ''}`;
}

/** numerator ÷ divisor for numerator ≥ 0, rounded half-up to a whole number. */
function roundHalfUp(numerator: bigint, divisor: bigint): bigint {
  return (numerator * 2n + divisor) / (divisor * 2n);
}

function withoutTrailingZeros(text: string): string {
  return text.includes('.') ? text.replace(/\.?0+$/, '') : text;
}

interface ReadNumber {
  negative: boolean;
  /** Every digit, integer and fraction together, without the decimal point. */
  digits: bigint;
  /** How many of the digits are decimals. */
  decimals: number;
}

function readNumber(text: string): ReadNumber | null {
  const match = NUMBER.exec(text.trim());
  if (!match) {
    return null;
  }
  const [, minus, integer, fraction = ''] = match;
  const digits = BigInt(`${integer.replaceAll(',', '')}${fraction}`);
  return { negative: minus === '-' && digits !== 0n, digits, decimals: fraction.length };
}
