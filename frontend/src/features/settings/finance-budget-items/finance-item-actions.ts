import type { ActionsMenuItem } from '@/ui/ActionsMenu/ActionsMenu';
import type { FinanceBudgetItem } from './api';

/** What people can do with a finance budget item, from its row or its details. */
export interface FinanceItemActions {
  view: (item: FinanceBudgetItem) => void;
  edit: (item: FinanceBudgetItem) => void;
  setActive: (item: FinanceBudgetItem, active: boolean) => void;
  remove: (item: FinanceBudgetItem) => void;
}

/** Set inactive or Reactivate · Delete (last, after a line — design system §6.1). */
export function moreActions(
  item: FinanceBudgetItem,
  actions: FinanceItemActions,
): ActionsMenuItem[] {
  return [
    item.active
      ? { label: 'Set inactive', onSelect: () => actions.setActive(item, false) }
      : { label: 'Reactivate', onSelect: () => actions.setActive(item, true) },
    {
      label: 'Delete',
      tone: 'danger',
      separatorBefore: true,
      onSelect: () => actions.remove(item),
    },
  ];
}

/** The year lock's reason before the API has given one. */
export const YEAR_LOCKED =
  "The year can't change once the item is saved — create a new item for another year.";

/** Why the item's year can't change (data model §8). */
export function yearLock(item: FinanceBudgetItem): string {
  return item.locks.find((lock) => lock.field === 'year')?.reason ?? YEAR_LOCKED;
}
