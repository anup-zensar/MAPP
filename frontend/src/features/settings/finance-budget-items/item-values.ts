import type { FinanceBudgetItem, FinanceBudgetItemCreate, FinanceBudgetItemUpdate } from './api';
import type { FinanceBudgetItemValues } from './schemas';

export function startValues(
  item: FinanceBudgetItem | undefined,
  year: number,
): FinanceBudgetItemValues {
  return {
    year: String(item?.year ?? year),
    budgetName: item?.budgetName ?? '',
    name: item?.name ?? '',
    code: item?.code ?? '',
  };
}

export function createBody(values: FinanceBudgetItemValues): FinanceBudgetItemCreate {
  return {
    year: Number(values.year),
    budgetName: values.budgetName,
    name: values.name,
    code: values.code,
  };
}

/** The fields people changed, with the version (D-203); null when nothing changed. */
export function changedFields(
  values: FinanceBudgetItemValues,
  start: FinanceBudgetItemValues,
  version: number,
): FinanceBudgetItemUpdate | null {
  const body: FinanceBudgetItemUpdate = { version };
  if (values.budgetName !== start.budgetName) {
    body.budgetName = values.budgetName;
  }
  if (values.name !== start.name) {
    body.name = values.name;
  }
  if (values.code !== start.code) {
    body.code = values.code;
  }
  return Object.keys(body).length > 1 ? body : null;
}
