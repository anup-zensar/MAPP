/** Exact decimals for form fields in BigInt, never a number (constitution VI); the API judges. */

const MAX_CENTS = 99_999_999_999_999_999n;

const NUMBER = /^(-)?(\d{1,3}(?:,\d{3})+|\d+)(?:\.(\d+))?$/;

const CURRENCY = /^(-?)(?:\p{Sc}|[a-z]{3}(?![a-z]))\s*/iu;

const grouped = new Intl.NumberFormat('en-US');
const twoDecimals = new Intl.NumberFormat('en-US', {
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
});
const rateDecimals = new Intl.NumberFormat('en-US', {
  minimumFractionDigits: 2,
  maximumFractionDigits: 6,
});

export interface ParsedAmount {
  amount: string;
  /** More than two decimals were typed and rounded half-up to the cent; the form says so (D-190). */
  rounded: boolean;
}

/** Reads "12,500", "12.5k", "1.2m", "$1,234"; half-up to the cent (D-190); null past the limit. */
export function parseMoneyInput(typed: string): ParsedAmount | null {
  const text = withoutCurrency(typed.trim());
  const suffix = /\s*([km])$/i.exec(text);
  const multiplier = !suffix ? 1n : suffix[1]?.toLowerCase() === 'k' ? 1_000n : 1_000_000n;
  const number = readNumber(suffix ? text.slice(0, suffix.index) : text);
  if (!number) {
    return null;
  }
  const exact = number.digits * multiplier * 100n;
  const divisor = 10n ** BigInt(number.decimals);
  const cents = roundHalfUp(exact, divisor);
  if (cents > MAX_CENTS) {
    return null;
  }
  const signed = number.negative ? -cents : cents;
  return { amount: formatCents(signed), rounded: exact % divisor !== 0n };
}

function withoutCurrency(text: string): string {
  const currency = CURRENCY.exec(text);
  return currency ? `${currency[1] ?? ''}${text.slice(currency[0].length)}` : text;
}

/** Unlike money, never rounds: more than `decimals` decimals gives null. */
export function parseDecimalInput(text: string, decimals: 0 | 2 | 6): string | null {
  const number = readNumber(text);
  if (!number || number.decimals > decimals) {
    return null;
  }
  const match = NUMBER.exec(text.trim());
  if (!match) {
    return null;
  }
  const [, minus = '', integer = '', fraction] = match;
  const plain = `${minus}${integer.replaceAll(',', '')}`;
  return fraction === undefined ? plain : `${plain}.${fraction}`;
}

export function parseWholeInput(text: string): bigint | null {
  const value = parseDecimalInput(text, 0);
  return value === null ? null : BigInt(value);
}

export function formatWhole(value: bigint | number | string): string {
  return grouped.format(typeof value === 'string' ? exact(value) : value);
}

export function formatAmount(amount: string): string {
  return twoDecimals.format(exact(amount));
}

/** "1,234.50 EUR" even for dollars, as the API's messages write it; screens use formatMoney. */
export function formatMoneyText(money: { amount: string; currency: string }): string {
  return `${formatAmount(money.amount)} ${money.currency}`;
}

export function formatPercent(percent: string): string {
  return `${twoDecimals.format(exact(percent))} %`;
}

export function formatRate(rate: string): string {
  return rateDecimals.format(exact(rate));
}

/** 1 ÷ rate, half-up to six decimals, no trailing zeros, as the API writes rates; null for ≤ 0. */
export function inverseRate(rate: string): string | null {
  const micros = toScaled(rate, 6);
  if (micros === null || micros <= 0n) {
    return null;
  }
  return withoutTrailingZeros(fromScaled(roundHalfUp(10n ** 12n, micros), 6));
}

export function toCents(amount: string): bigint {
  const cents = toScaled(amount, 2);
  if (cents === null) {
    throw new Error(`Not an amount with at most two decimals: "${amount}"`);
  }
  return cents;
}

export function formatCents(cents: bigint): string {
  return fromScaled(cents, 2);
}

export function toHundredths(percent: string): bigint | null {
  return toScaled(percent, 2);
}

export function formatHundredths(hundredths: bigint): string {
  return fromScaled(hundredths, 2);
}

/** 3 → "33.33", "33.33", "33.34": the remainder goes last (Go's PercentSplitEvenly). */
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

function toScaled(text: string, scale: number): bigint | null {
  const number = readNumber(text);
  if (!number || number.decimals > scale) {
    return null;
  }
  const value = number.digits * 10n ** BigInt(scale - number.decimals);
  return number.negative ? -value : value;
}

function fromScaled(value: bigint, scale: number): string {
  const negative = value < 0n;
  const digits = (negative ? -value : value).toString().padStart(scale + 1, '0');
  const integer = digits.slice(0, digits.length - scale);
  const fraction = digits.slice(digits.length - scale);
  return `${negative ? '-' : ''}${integer}${scale > 0 ? `.${fraction}` : ''}`;
}

function roundHalfUp(numerator: bigint, divisor: bigint): bigint {
  return (numerator * 2n + divisor) / (divisor * 2n);
}

function withoutTrailingZeros(text: string): string {
  return text.includes('.') ? text.replace(/\.?0+$/, '') : text;
}

interface ReadNumber {
  negative: boolean;
  digits: bigint;
  decimals: number;
}

function readNumber(text: string): ReadNumber | null {
  const match = NUMBER.exec(text.trim());
  if (!match) {
    return null;
  }
  const [, minus, integer = '', fraction = ''] = match;
  const digits = BigInt(`${integer.replaceAll(',', '')}${fraction}`);
  return { negative: minus === '-' && digits !== 0n, digits, decimals: fraction.length };
}

/** A decimal string Intl.NumberFormat formats exactly, without floating point. */
export function exact(value: string): Intl.StringNumericLiteral | number {
  return isNumericLiteral(value) ? value : Number.NaN;
}

function isNumericLiteral(value: string): value is Intl.StringNumericLiteral {
  return value.trim() !== '' && !Number.isNaN(Number(value));
}
