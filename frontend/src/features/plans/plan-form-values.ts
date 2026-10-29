import { formatAmount, formatWhole, parseDecimalInput, parseMoneyInput } from '@/format/decimal';
import { planCurrencyOf, planTypeOf } from './plan-currency';
import type { Plan, PlanCreate, PlanDefaults, PlanUpdate } from './api';
import type { GoalStage } from './plan-text';
import type { PlanFormValues } from './schemas';

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
    overspend: defaults.overspend,
    attributes: {},
  };
}

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
    overspend: plan.overspend.mode,
    attributes: plan.attributes,
  };
}

function goalAmountText(stage: GoalStage, amount: string | null): string {
  if (!amount) {
    return '';
  }
  return stage === 'revenue' ? formatAmount(amount) : formatWhole(amount);
}

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
    overspend: values.overspend,
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

export function answers016(source: { planType?: unknown } | undefined): boolean {
  return source?.planType !== undefined;
}

export function sameValues(a: PlanFormValues, b: PlanFormValues): boolean {
  return JSON.stringify(a) === JSON.stringify(b);
}
