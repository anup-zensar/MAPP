import { formatAmount, formatCents, parseMoneyInput, toCents } from '@/format/decimal';
import { formatMonth } from '@/format/intl';
import type { DateFormat } from '@/format/intl';

/** The months' short names, January first (the person's date format never changes them). */
export const MONTH_NAMES = [
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
] as const;

export function monthKeys(year: number): string[] {
  return MONTH_NAMES.map((_, index) => `${year}-${String(index + 1).padStart(2, '0')}`);
}

export function monthLabel(key: string, format?: DateFormat): string {
  return formatMonth(key, format);
}

export function quarterLabels(year: number): string[] {
  return [1, 2, 3, 4].map((quarter) => `Q${quarter} ${year}`);
}

/** As Go's Money.SplitEvenly: parts rounded down to the cent, the rest in the last part. */
export function splitEvenly(total: bigint, parts: number): bigint[] {
  const count = BigInt(parts);
  const share = total / count;
  const result = Array.from({ length: parts }, () => share);
  result[parts - 1] = total - share * (count - 1n);
  return result;
}

export function sum(amounts: bigint[]): bigint {
  return amounts.reduce((total, amount) => total + amount, 0n);
}

export function quarterTotals(months: bigint[]): bigint[] {
  return [0, 1, 2, 3].map((quarter) => sum(months.slice(quarter * 3, quarter * 3 + 3)));
}

export function withQuarter(months: bigint[], quarter: number, amount: bigint): bigint[] {
  const next = [...months];
  splitEvenly(amount, 3).forEach((part, index) => {
    next[quarter * 3 + index] = part;
  });
  return next;
}

/** "" is 0.00, "12.5k" is 12,500.00; null for text that is not an amount or is below zero. */
export function readAmount(text: string): bigint | null {
  if (text.trim() === '') {
    return 0n;
  }
  const parsed = parseMoneyInput(text);
  if (!parsed) {
    return null;
  }
  const cents = toCents(parsed.amount);
  return cents < 0n ? null : cents;
}

export function amountText(cents: bigint): string {
  return formatAmount(formatCents(cents));
}

export type PhasingBalance =
  | { kind: 'noBudget'; phased: bigint }
  | { kind: 'left'; phased: bigint; budget: bigint; difference: bigint }
  | { kind: 'tooMuch'; phased: bigint; budget: bigint; difference: bigint }
  | { kind: 'exact'; phased: bigint; budget: bigint };

export function balanceOf(months: bigint[], budget: bigint | null): PhasingBalance {
  const phased = sum(months);
  if (budget === null) {
    return { kind: 'noBudget', phased };
  }
  if (phased === budget) {
    return { kind: 'exact', phased, budget };
  }
  return phased < budget
    ? { kind: 'left', phased, budget, difference: budget - phased }
    : { kind: 'tooMuch', phased, budget, difference: phased - budget };
}

export function balanceText(balance: PhasingBalance, currency: string): string {
  const phased = amountText(balance.phased);
  switch (balance.kind) {
    case 'noBudget':
      return `${phased} ${currency} phased`;
    case 'exact':
      return `${phased} of ${amountText(balance.budget)} ${currency} phased — all of it`;
    case 'left':
      return `${phased} of ${amountText(balance.budget)} ${currency} phased — ${amountText(balance.difference)} left`;
    case 'tooMuch':
      return `${phased} of ${amountText(balance.budget)} ${currency} phased — ${amountText(balance.difference)} too much`;
  }
}

/** By how much a period is over (planned cost or the children's phased budgets above its budget). */
export function overBy(
  budget: string | null | undefined,
  plannedCost: string | null | undefined,
  allocated: string | null | undefined,
): bigint {
  if (!budget) {
    return 0n;
  }
  const limit = toCents(budget);
  const planned = plannedCost ? toCents(plannedCost) - limit : 0n;
  const given = allocated ? toCents(allocated) - limit : 0n;
  const most = planned > given ? planned : given;
  return most > 0n ? most : 0n;
}
