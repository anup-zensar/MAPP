import { plural } from '@/format/text';

export const PAGE_DESCRIPTION = 'Money for a year, shared out to market plans.';

export const FIGURE_TIPS = {
  amount: 'The money of this global budget for the year, in USD.',
  allocated: 'The part given to plans. Each allocation is that plan’s budget.',
  unallocated: 'Global budget minus allocated: what is left to give to plans.',
  plannedCost: 'What the allocated plans plan to spend, from their activities.',
};

export function emptyText(year: number | null): string {
  return year === null ? 'No global budgets yet.' : `No global budgets for ${year} yet.`;
}

/** "Over-allocated by $100,000.00" (message 32; never blocks, D-015). */
export function overAllocatedText(amount: string): string {
  return `Over-allocated by ${amount}`;
}

/** "Other markets (2 plans)" — allocations to plans the person can't see (FR-033). */
export function otherMarketsLabel(planCount: number): string {
  return `Other markets (${plural(planCount, 'plan')})`;
}

export function withoutAllocationHeading(year: number): string {
  return `Plans of ${year} without an allocation`;
}

export function inUseText(name: string, plans: number): string {
  return plans === 1
    ? `'${name}' has an allocation to 1 plan. Remove it first.`
    : `'${name}' has allocations to ${plans} plans. Remove them first.`;
}

export function removeQuestion(plan: string): string {
  return `Remove the allocation to '${plan}'?`;
}

export function removeEffect(plan: string, amount: string): string {
  return `'${plan}' keeps its budget of ${amount}, and its owner can change the budget again.`;
}

export const toasts = {
  created: (name: string) => `Global budget '${name}' created`,
  saved: (name: string) => `Global budget '${name}' saved`,
  allocated: (plan: string) => `Allocation to '${plan}' saved`,
  removed: (plan: string) => `Allocation to '${plan}' removed`,
};

export function budgetUsedText(percent: string | null): string {
  return percent === null ? '—' : `${Math.round(Number(percent))}%`;
}

/** The years a new global budget can have: the current year −1 to +5 (FR-029). */
export function yearChoices(currentYear: number): number[] {
  return Array.from({ length: 7 }, (_, index) => currentYear - 1 + index);
}
