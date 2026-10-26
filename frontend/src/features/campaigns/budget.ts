/**
 * Budget arithmetic of the forms, exact to the cent with BigInt (FR-010, constitution VI). What it
 * computes is shown while typing; the API stays the judge of what is saved.
 */
import type { components } from '@/api/generated/schema';
import { formatAmount, formatCents, parseMoneyInput, toCents } from '@/lib/decimal';

type Money = components['schemas']['Money'];
type MoneyInput = components['schemas']['MoneyInput'];

/**
 * Budgets are in the plan currency (016 FR-011); US dollars, the default, while a plan does not
 * name one (data model §1).
 */
export const BUDGET_CURRENCY = 'USD';

/**
 * The parent's remaining budget with the typed amount in place of the saved one:
 * parentRemaining + savedOwn − typed. Null when the parent has no budget. Text that is not an
 * amount counts as nothing (the API names the problem on Save).
 */
export function remainingWhileTyping(
  parentRemaining: Money | null,
  savedOwn: Money | null,
  typed: string,
): Money | null {
  if (!parentRemaining) {
    return null;
  }
  const typedCents = toCents(parseMoneyInput(typed)?.amount ?? '0');
  const savedCents = savedOwn ? toCents(savedOwn.amount) : 0n;
  return {
    amount: formatCents(toCents(parentRemaining.amount) + savedCents - typedCents),
    currency: parentRemaining.currency,
  };
}

/** Below zero: shown in red with "Over-allocated by …". */
export function isNegative(money: Money): boolean {
  return toCents(money.amount) < 0n;
}

/** The amount without its sign, for "Over-allocated by …". */
export function withoutSign(money: Money): Money {
  return { amount: money.amount.replace(/^-/, ''), currency: money.currency };
}

/**
 * The budget as the API takes it, in the plan currency: null when empty; the amount rounded to the
 * cent (D-190); text that is not an amount is sent as typed, so the API reports it with every
 * other problem.
 */
export function budgetInput(typed: string, currency = BUDGET_CURRENCY): MoneyInput | null {
  const text = typed.trim();
  if (text === '') {
    return null;
  }
  return { amount: parseMoneyInput(text)?.amount ?? text, currency };
}

/** A stored budget as the money field shows it: "1,200,000.00"; "" without one. */
export function budgetText(budget: Money | null): string {
  return budget ? formatAmount(budget.amount) : '';
}
