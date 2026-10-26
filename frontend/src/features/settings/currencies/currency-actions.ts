import type { ActionsMenuItem } from '@/ui/ActionsMenu/ActionsMenu';
import { formatRate } from '@/format/decimal';
import type { Currency } from './api';

/** What people can do with a currency, from its row or its details. */
export interface CurrencyActions {
  view: (currency: Currency) => void;
  edit: (currency: Currency) => void;
  setActive: (currency: Currency, active: boolean) => void;
  remove: (currency: Currency) => void;
  /** Says why a locked action can't be done (USD's status and delete). */
  explainLock: (reason: string) => void;
}

/** "1 USD = 0.92 EUR". */
export function rateLine(code: string, rate: string): string {
  return `1 USD = ${formatRate(rate)} ${code}`;
}

/** "1 EUR = 1.086957 USD". */
export function inverseLine(code: string, inverseRate: string): string {
  return `1 ${code} = ${formatRate(inverseRate)} USD`;
}

/** Why a field or action of the currency can't change (FR-028); undefined when it can. */
export function lockOf(currency: Currency, field: string): string | undefined {
  return currency.locks.find((lock) => lock.field === field)?.reason;
}

/**
 * Set inactive or Reactivate · Delete (last, after a line). A locked action stays in the menu with
 * its reason under it, and choosing it says why (design system §6.6).
 */
export function moreActions(currency: Currency, actions: CurrencyActions): ActionsMenuItem[] {
  const statusLock = lockOf(currency, 'active');
  const deleteLock = lockOf(currency, 'delete');
  return [
    currency.active
      ? {
          label: 'Set inactive',
          hint: statusLock,
          onSelect: () =>
            statusLock ? actions.explainLock(statusLock) : actions.setActive(currency, false),
        }
      : { label: 'Reactivate', onSelect: () => actions.setActive(currency, true) },
    {
      label: 'Delete',
      tone: 'danger',
      separatorBefore: true,
      hint: deleteLock,
      onSelect: () => (deleteLock ? actions.explainLock(deleteLock) : actions.remove(currency)),
    },
  ];
}
