import { formatCents, parseMoneyInput, toCents } from '@/format/decimal';
import type { Money, OverspendLimit, OverspendMode } from './api';
import {
  alreadyOverText,
  budgetBelowPlannedText,
  leftToPlanText,
  overLimitText,
} from './overspend-text';

export interface BudgetCheckInput {
  mode: OverspendMode;
  typed: string;
  stored: string | null;
  planned: Money | null;
}

export interface BudgetCheck {
  error: string | null;
  note: string | null;
}

const NOTHING: BudgetCheck = { error: null, note: null };

function centsOf(text: string): bigint | null {
  const parsed = parseMoneyInput(text.trim());
  return parsed ? toCents(parsed.amount) : null;
}

/** A Refuse budget typed below its planned cost: refused when lowered or newly set, noted when kept. */
export function budgetCheck({ mode, typed, stored, planned }: BudgetCheckInput): BudgetCheck {
  const typedCents = centsOf(typed);
  if (mode !== 'refuse' || !planned || typedCents === null) {
    return NOTHING;
  }
  const plannedCents = toCents(planned.amount);
  if (typedCents >= plannedCents) {
    return NOTHING;
  }
  if (stored === null || typedCents < toCents(stored)) {
    return { error: budgetBelowPlannedText(planned), note: null };
  }
  const over = { amount: formatCents(plannedCents - typedCents), currency: planned.currency };
  return { error: null, note: alreadyOverText(over) };
}

/** What a typed planned cost would take over each Refuse budget above the activity. */
export function overLimitMessages(
  limits: OverspendLimit[],
  before: string | null,
  typed: string,
): string[] {
  const typedCents = centsOf(typed);
  if (typedCents === null) {
    return [];
  }
  const increase = typedCents - (before === null ? 0n : toCents(before));
  return limits
    .filter((limit) => increase > 0n && increase > toCents(limit.leftToPlan.amount))
    .map((limit) => {
      const over = increase - toCents(limit.leftToPlan.amount);
      return overLimitText(
        { amount: formatCents(over), currency: limit.leftToPlan.currency },
        limit.budget.type,
        limit.budget.name,
      );
    });
}

/** The note under an activity's planned cost: what is left in each Refuse budget above it. */
export function limitNotes(limits: OverspendLimit[]): string[] {
  return limits.map((limit) =>
    leftToPlanText(limit.leftToPlan, limit.budget.type, limit.budget.name),
  );
}
