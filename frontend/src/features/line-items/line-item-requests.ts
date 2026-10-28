/** The line-item form's requests and the API's refusals (007). */
import type { ApiError } from '@/api/errors';
import type { components } from '@/api/generated/schema';
import { parseMoneyInput } from '@/format/decimal';
import type { LineItemCreate } from './api';
import { monthFieldId } from './line-item-form-state';
import type { LineItemFormState } from './line-item-form-state';

type AttributeValue = components['schemas']['AttributeValue'];
type AttributeValues = components['schemas']['AttributeValues'];
type ClearedValue = components['schemas']['ClearedValue'];
type SplitShare = components['schemas']['SplitShare'];

/** Amounts with at most two decimals (D-149, D-190), null when empty; months only for Custom. */
export function toCreateBody(
  state: LineItemFormState,
  attributes: AttributeValues = state.attributes,
): LineItemCreate {
  return {
    name: state.name,
    ownerId: state.ownerId || null,
    description: state.description,
    currency: state.currency,
    plannedCost: amountForApi(state.plannedCost),
    spread: {
      method: state.spreadMethod,
      months:
        state.spreadMethod === 'custom'
          ? state.customMonths
              .filter((month) => month.amount.trim() !== '')
              .map((month) => ({ month: month.month, amount: amountForApi(month.amount) ?? '' }))
          : [],
    },
    attributes,
    contractedAmount: amountForApi(state.contractedAmount),
    requestSwbs: state.requestSwbs,
    financeBudgetYear: state.financeBudgetYear,
    financeBudgetItemId: state.financeBudgetItemId || null,
  };
}

export function toUpdateBody(
  state: LineItemFormState,
  version: number,
  attributes: AttributeValues = state.attributes,
) {
  return { ...toCreateBody(state, attributes), version };
}

/** "spread.months[1].amount" goes to that month's input. */
export function formProblems(
  error: ApiError,
  body: LineItemCreate,
): { field: string; message: string }[] {
  return error.fields.map((problem) => {
    const month = /^spread\.months\[(\d+)\]/.exec(problem.field);
    const sent = month ? body.spread.months[Number(month[1])] : undefined;
    return sent ? { ...problem, field: monthFieldId(sent.month) } : problem;
  });
}

export function withoutCleared(values: AttributeValues, cleared: ClearedValue[]): AttributeValues {
  const result = { ...values };
  for (const { attributeId, optionIds } of cleared) {
    result[attributeId] = withoutOptions(result[attributeId] ?? null, optionIds);
  }
  return result;
}

function withoutOptions(value: AttributeValue, optionIds: string[]): AttributeValue {
  if (!Array.isArray(value)) {
    return null;
  }
  const items: (string | SplitShare)[] = value;
  const kept = items.filter(
    (item) => !optionIds.includes(typeof item === 'string' ? item : item.optionId),
  );
  const options = kept.filter((item) => typeof item === 'string');
  const shares = kept.filter((item) => typeof item !== 'string');
  return options.length > 0 ? options : shares.length > 0 ? shares : null;
}

/** The text itself when it is no amount: the API refuses it with its own message. */
function amountForApi(text: string): string | null {
  const trimmed = text.trim();
  if (trimmed === '') {
    return null;
  }
  return parseMoneyInput(trimmed)?.amount ?? trimmed;
}
