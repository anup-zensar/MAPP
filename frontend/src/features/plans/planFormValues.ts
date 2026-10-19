/**
 * The plan form's values as typed, and the request bodies made from them (plan "Plan form").
 * Amounts stay text while typed; they are read only when sent. Text that can't be read as an amount
 * is sent as typed, so the API reports it together with every other problem (FR-010).
 */
import type { AttributeValues } from '@/components';
import { formatAmount, formatWhole, parseDecimalInput, parseMoneyInput } from '@/lib/decimal';
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
  /** The plan budget in USD as typed ("1,200,000.00"); "" for none. */
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

/** What the API gets for the plan budget: USD with two decimals, null when empty. */
function budgetOf(text: string): PlanCreate['budget'] {
  const typed = text.trim();
  if (typed === '') {
    return null;
  }
  return { amount: parseMoneyInput(typed)?.amount ?? typed, currency: 'USD' };
}

function goalAmountOf(stage: GoalStage, text: string): string | null {
  const typed = text.trim();
  if (typed === '') {
    return null;
  }
  const read = stage === 'revenue' ? parseMoneyInput(typed)?.amount : parseDecimalInput(typed, 0);
  return read ?? typed;
}

function common(values: PlanFormValues) {
  return {
    name: values.name,
    modelId: values.modelId || null,
    teamIds: values.teamIds,
    ownerId: values.ownerId || null,
    aop: values.aop,
    description: values.description,
    budget: budgetOf(values.budget),
    defaultFinanceBudgetId: values.defaultFinanceBudgetId || null,
    defaultSpread: values.defaultSpread,
    bcpir: values.bcpir,
    goalStage: values.goalStage,
    goalAmount: goalAmountOf(values.goalStage, values.goalAmount),
    attributes: values.attributes,
  };
}

export function toCreateBody(values: PlanFormValues): PlanCreate {
  const year = Number(values.year);
  return { ...common(values), year: values.year && Number.isInteger(year) ? year : null };
}

export function toUpdateBody(values: PlanFormValues, version: number): PlanUpdate {
  return { version, ...common(values) };
}

/** Whether two forms hold the same values (the unsaved-changes guard). */
export function sameValues(a: PlanFormValues, b: PlanFormValues): boolean {
  return JSON.stringify(a) === JSON.stringify(b);
}
