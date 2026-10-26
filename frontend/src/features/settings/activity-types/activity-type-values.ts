import { formatAmount, parseMoneyInput, toCents } from '@/format/decimal';
import type { ActivityType, ActivityTypeCreate, ActivityTypeUpdate } from './api';
import type { ActivityTypeValues } from './schemas';

export function startValues(type: ActivityType | undefined): ActivityTypeValues {
  return {
    name: type?.name ?? '',
    defaultPlannedCost: type?.defaultPlannedCost
      ? formatAmount(type.defaultPlannedCost.amount)
      : '',
    modelIds: type?.models.map((model) => model.id) ?? [],
  };
}

export function createBody(values: ActivityTypeValues): ActivityTypeCreate {
  return {
    name: values.name,
    defaultPlannedCost: costInput(values.defaultPlannedCost),
    modelIds: values.modelIds,
  };
}

/** The fields people changed, with the version (D-203); null when nothing changed. */
export function changedFields(
  type: ActivityType,
  start: ActivityTypeValues,
  values: ActivityTypeValues,
  version: number,
): ActivityTypeUpdate | null {
  const body: ActivityTypeUpdate = { version };
  if (values.name !== start.name) {
    body.name = values.name;
  }
  if (!sameCost(values.defaultPlannedCost, type.defaultPlannedCost)) {
    body.defaultPlannedCost = costInput(values.defaultPlannedCost);
  }
  if (!sameIds(values.modelIds, start.modelIds)) {
    // The complete new list of models allowing the type.
    body.modelIds = values.modelIds;
  }
  return Object.keys(body).length > 1 ? body : null;
}

export function removesModels(before: string[], after: string[]): boolean {
  return before.some((id) => !after.includes(id));
}

/** "5,000" and 5000.00 USD are the same cost; empty is the same as no default. */
export function sameCost(typed: string, saved: ActivityType['defaultPlannedCost']): boolean {
  if (typed.trim() === '' || saved === null) {
    return typed.trim() === '' && saved === null;
  }
  const amount = parseMoneyInput(typed);
  return amount !== null && toCents(amount.amount) === toCents(saved.amount);
}

function sameIds(a: string[], b: string[]): boolean {
  return a.length === b.length && a.every((id) => b.includes(id));
}

function costInput(text: string): ActivityTypeCreate['defaultPlannedCost'] {
  if (text.trim() === '') {
    return null;
  }
  return { amount: parseMoneyInput(text)?.amount ?? text.trim(), currency: 'USD' };
}
