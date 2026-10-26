/**
 * The arithmetic of Budget by month (016 data-model §3.3) for instant feedback while someone types.
 * Amounts are cents (BigInt), never floating point. The API stays the judge: it works out the
 * Evenly months and checks every save with the same rules.
 */
import { formatAmount, formatCents, parseMoneyInput, toCents } from '@/format/decimal';

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

/** "2027-01" … "2027-12": the month keys of a plan's year. */
export function monthKeys(year: number): string[] {
  return MONTH_NAMES.map((_, index) => `${year}-${String(index + 1).padStart(2, '0')}`);
}

/** "Mar 2027" for "2027-03". */
export function monthLabel(key: string): string {
  const [year, month] = key.split('-');
  return `${MONTH_NAMES[Number(month) - 1]} ${year}`;
}

/** "Q1 2027" … "Q4 2027". */
export function quarterLabels(year: number): string[] {
  return [1, 2, 3, 4].map((quarter) => `Q${quarter} ${year}`);
}

/**
 * The Evenly rule (domain model §4.4, Go's Money.SplitEvenly): equal parts rounded down to the cent,
 * the rest in the last part. 100,000.00 in 12 → 8,333.33 × 11 and 8,333.37; in 3 → 33,333.33,
 * 33,333.33, 33,333.34.
 */
export function splitEvenly(total: bigint, parts: number): bigint[] {
  const count = BigInt(parts);
  const share = total / count;
  const result = Array.from({ length: parts }, () => share);
  result[parts - 1] = total - share * (count - 1n);
  return result;
}

/** The sum of some cents. */
export function sum(amounts: bigint[]): bigint {
  return amounts.reduce((total, amount) => total + amount, 0n);
}

/** Twelve months as four quarters (January–March first). */
export function quarterTotals(months: bigint[]): bigint[] {
  return [0, 1, 2, 3].map((quarter) => sum(months.slice(quarter * 3, quarter * 3 + 3)));
}

/** Twelve months after one quarter's amount is typed: its three months by the Evenly rule. */
export function withQuarter(months: bigint[], quarter: number, amount: bigint): bigint[] {
  const next = [...months];
  splitEvenly(amount, 3).forEach((part, index) => {
    next[quarter * 3 + index] = part;
  });
  return next;
}

/**
 * Typed text as cents: "" is 0.00, "12.5k" is 12,500.00; null for text that is not an amount or is
 * below zero (Budget by month takes 0.00 or more).
 */
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

/** Cents as typed text in a field: "50,000.00". */
export function amountText(cents: bigint): string {
  return formatAmount(formatCents(cents));
}

/** The running total under the grid (016 FR-021). */
export type PhasingBalance =
  | { kind: 'noBudget'; phased: bigint }
  | { kind: 'left'; phased: bigint; budget: bigint; difference: bigint }
  | { kind: 'tooMuch'; phased: bigint; budget: bigint; difference: bigint }
  | { kind: 'exact'; phased: bigint; budget: bigint };

/** How the typed months compare with the budget. */
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

/**
 * "590,000.00 of 600,000.00 GBP phased — 10,000.00 left", "… — 5,000.00 too much", "… — all of
 * it": the editor's running total (016 FR-021).
 */
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
