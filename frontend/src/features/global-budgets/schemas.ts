import { z } from 'zod';
import type { OverspendMode } from '@/features/overspend';
import { formatAmount, parseMoneyInput } from '@/format/decimal';
import type { GlobalBudget, GlobalBudgetCreate, GlobalBudgetUpdate } from './api';

const AMOUNT_PROBLEM = 'Enter an amount of 0 or more.';
const yearSchema = z.coerce.number().int().min(2000).max(2100);

/** "?year=all" lists every year (null); a missing or unknown year means the current one. */
export function yearOf(params: URLSearchParams, currentYear: number): number | null {
  const value = params.get('year');
  if (value === 'all') {
    return null;
  }
  const year = yearSchema.safeParse(value);
  return year.success ? year.data : currentYear;
}

/** The typed amount as money of 0 or more ("12,500.00" → "12500.00"); null otherwise. */
export function amountOf(text: string): string | null {
  const parsed = parseMoneyInput(text);
  return parsed && !parsed.amount.startsWith('-') ? parsed.amount : null;
}

const amountSchema = z.string().refine((text) => amountOf(text) !== null, AMOUNT_PROBLEM);

// The API's own checks, so the first Save names every problem.
export const globalBudgetSchema = z.object({
  name: z
    .string()
    .refine((name) => name.trim() !== '', 'Enter a name for the global budget.')
    .refine((name) => [...name.trim()].length <= 120, 'Use 120 characters or fewer.'),
  year: z.string(),
  amount: amountSchema,
  ownerId: z
    .string()
    .refine((id) => id !== '', 'Choose a person who can manage global budgets for all markets.'),
  description: z.string(),
  overspend: z.enum(['warn', 'refuse']),
});

export type GlobalBudgetValues = z.infer<typeof globalBudgetSchema>;

export function globalBudgetValuesOf(
  budget: GlobalBudget | undefined,
  currentYear: number,
  defaultMode: OverspendMode = 'warn',
): GlobalBudgetValues {
  return {
    name: budget?.name ?? '',
    year: String(budget?.year ?? currentYear),
    amount: budget ? formatAmount(budget.amount.amount) : '',
    ownerId: budget?.owner?.id ?? '',
    description: budget?.description ?? '',
    overspend: budget?.overspend.mode ?? defaultMode,
  };
}

export function newGlobalBudget(values: GlobalBudgetValues, amount: string): GlobalBudgetCreate {
  return {
    name: values.name.trim(),
    year: Number(values.year),
    amount: { amount, currency: 'USD' },
    ownerId: values.ownerId,
    description: values.description,
    overspend: values.overspend,
  };
}

/** The fields people changed, with the version (D-203); null when nothing changed. */
export function changedFields(
  budget: GlobalBudget,
  values: GlobalBudgetValues,
  amount: string,
  version: number,
): GlobalBudgetUpdate | null {
  const body: GlobalBudgetUpdate = { version };
  if (values.name.trim() !== budget.name) {
    body.name = values.name.trim();
  }
  if (amount !== budget.amount.amount) {
    body.amount = { amount, currency: 'USD' };
  }
  if (values.ownerId !== (budget.owner?.id ?? '')) {
    body.ownerId = values.ownerId;
  }
  if (values.description !== budget.description) {
    body.description = values.description;
  }
  if (values.overspend !== budget.overspend.mode) {
    body.overspend = values.overspend;
  }
  return Object.keys(body).length > 1 ? body : null;
}

/** A plan is needed only when the dialog allocates and none is chosen among those offered. */
export function allocationSchema(isPlanChosen: (planId: string) => boolean) {
  return z.object({
    planId: z.string().refine(isPlanChosen, 'Choose a plan.'),
    amount: amountSchema,
    overspend: z.enum(['warn', 'refuse']),
  });
}

export type AllocationValues = z.infer<ReturnType<typeof allocationSchema>>;
