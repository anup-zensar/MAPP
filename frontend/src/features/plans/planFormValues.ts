/**
 * The plan form's values as typed, and the request bodies made from them (plan "Plan form").
 * Amounts stay text while typed; they are read only when sent. Text that can't be read as an amount
 * is sent as typed, so the API reports it together with every other problem (FR-010).
 */
import type { AttributeValues } from '@/components';
import { formatAmount, formatWhole, parseDecimalInput, parseMoneyInput } from '@/lib/decimal';
import { planCurrencyOf, planTypeOf } from './planCurrency';
import type { PlanType } from './planCurrency';
import type { Plan, PlanCreate, PlanDefaults, PlanUpdate } from './plansApi';
import type { GoalStage, SpreadMethod } from './planText';

export interface PlanFormValues {
  name: string;
  /** The year as chosen ("2026"); fixed after the first save. */
  year: string;
  modelId: string;
  teamIds: string[];
  ownerId: string;
  aop: boolean;
  description: string;
  /** Operational (default), Strategic or Cross-market (016 FR-032). */
  planType: PlanType;
  /** The plan currency (016 FR-011): the budget and every amount of the plan are kept in it. */
  currency: string;
  /** The plan budget in the plan currency as typed ("1,200,000.00"); "" for none. */
  budget: string;
  /** "" for no default. */
  defaultFinanceBudgetId: string;
  defaultSpread: SpreadMethod;
  bcpir: boolean;
  goalStage: GoalStage;
  /** As typed: a whole number for INQ and MQL, USD for Revenue; "" for none. */
  goalAmount: string;
  attributes: AttributeValues;
}

/** A new plan's form, filled in with the person's defaults (FR-006). */
export function valuesFromDefaults(defaults: PlanDefaults): PlanFormValues {
  return {
    name: '',
    year: String(defaults.year),
    modelId: defaults.modelId ?? '',
    teamIds: defaults.teamIds,
    ownerId: defaults.ownerId ?? '',
    aop: defaults.aop,
    description: '',
    planType: planTypeOf(defaults),
    currency: planCurrencyOf(defaults),
    budget: '',
    defaultFinanceBudgetId: '',
    defaultSpread: defaults.defaultSpread,
    bcpir: defaults.bcpir,
    goalStage: defaults.goalStage,
    goalAmount: '',
    attributes: {},
  };
}

/** A saved plan's form (the edit page copies it once, when it opens). */
export function valuesFromPlan(plan: Plan): PlanFormValues {
  return {
    name: plan.name,
    year: String(plan.year),
    modelId: plan.model.id,
    teamIds: plan.teams.map((team) => team.id),
    ownerId: plan.owner?.id ?? '',
    aop: plan.aop,
    description: plan.description,
    planType: planTypeOf(plan),
    currency: plan.budget?.currency ?? planCurrencyOf(plan),
    budget: plan.budget ? formatAmount(plan.budget.amount) : '',
    defaultFinanceBudgetId: plan.defaultFinanceBudget?.id ?? '',
    defaultSpread: plan.defaultSpread,
    bcpir: plan.bcpir,
    goalStage: plan.goalStage,
    goalAmount: goalAmountText(plan.goalStage, plan.goalAmount),
    attributes: plan.attributes,
  };
}

function goalAmountText(stage: GoalStage, amount: string | null): string {
  if (!amount) {
    return '';
  }
  return stage === 'revenue' ? formatAmount(amount) : formatWhole(amount);
}

/** What the API gets for the plan budget: the plan currency with two decimals, null when empty. */
function budgetOf(text: string, currency: string): PlanCreate['budget'] {
  const typed = text.trim();
  if (typed === '') {
    return null;
  }
  return { amount: parseMoneyInput(typed)?.amount ?? typed, currency };
}

function goalAmountOf(stage: GoalStage, text: string): string | null {
  const typed = text.trim();
  if (typed === '') {
    return null;
  }
  const read = stage === 'revenue' ? parseMoneyInput(typed)?.amount : parseDecimalInput(typed, 0);
  return read ?? typed;
}

/**
 * The body's fields. The plan type and currency (016) go only to an API that answers them
 * (`with016`): until 016's backend is merged, the API refuses fields it does not know.
 */
function common(values: PlanFormValues, with016: boolean) {
  return {
    ...(with016 && { planType: values.planType, currency: values.currency }),
    name: values.name,
    modelId: values.modelId || null,
    teamIds: values.teamIds,
    ownerId: values.ownerId || null,
    aop: values.aop,
    description: values.description,
    budget: budgetOf(values.budget, with016 ? values.currency : 'USD'),
    defaultFinanceBudgetId: values.defaultFinanceBudgetId || null,
    defaultSpread: values.defaultSpread,
    bcpir: values.bcpir,
    goalStage: values.goalStage,
    goalAmount: goalAmountOf(values.goalStage, values.goalAmount),
    attributes: values.attributes,
  };
}

export function toCreateBody(values: PlanFormValues, with016 = false): PlanCreate {
  const year = Number(values.year);
  return { ...common(values, with016), year: values.year && Number.isInteger(year) ? year : null };
}

export function toUpdateBody(values: PlanFormValues, version: number, with016 = false): PlanUpdate {
  return { version, ...common(values, with016) };
}

/** Whether the API answers 016's plan type and currency (it sends them on every plan and default). */
export function answers016(source: { planType?: unknown } | undefined): boolean {
  return source?.planType !== undefined;
}

/** Whether two forms hold the same values (the unsaved-changes guard). */
export function sameValues(a: PlanFormValues, b: PlanFormValues): boolean {
  return JSON.stringify(a) === JSON.stringify(b);
}
