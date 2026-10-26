import type { components } from '@/api/generated/schema';

type Money = components['schemas']['Money'];
export type PlanType = components['schemas']['PlanType'];

/** The plan types of feature 016 (FR-032), in the order the form and the filter offer them. */
export const PLAN_TYPES: PlanType[] = ['operational', 'strategic', 'crossMarket'];

/** "Operational", "Strategic", "Cross-market". */
export const PLAN_TYPE_LABELS: Record<PlanType, string> = {
  operational: 'Operational',
  strategic: 'Strategic',
  crossMarket: 'Cross-market',
};

/**
 * A plan's type. Every plan has one once 016's backend is merged; until then the API leaves it
 * out and the plan counts as Operational, the documented default (FR-032).
 */
export function planTypeOf(plan: { planType?: PlanType }): PlanType {
  return plan.planType ?? 'operational';
}

/** A plan's currency: USD, the default (FR-011), while the API does not send it yet. */
export function planCurrencyOf(plan: { currency?: string }): string {
  return plan.currency ?? 'USD';
}

/** True when two amounts are in different currencies, so both are worth showing. */
export function differentCurrency(
  primary: Money | null | undefined,
  secondary: Money | null | undefined,
): boolean {
  return !!primary && !!secondary && primary.currency !== secondary.currency;
}
