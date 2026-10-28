import { formatCents, formatMoneyText, toCents } from '@/format/decimal';
import type { PurchaseRequest, PurchaseRequestLine } from './api';

export const GSAP_ORDER = 'Create GSAP PR and PO (simulated)';
export const GSAP_ORDER_TITLE = 'Create GSAP PR and PO (simulated)?';
export const GSAP_TOP_UP_TITLE = 'Update the PO (simulated)?';
export const GSAP_ORDER_CREATE = 'Create';
export const GSAP_ORDER_UPDATE = 'Update PO';
export const GSAP_ORDER_LINES = 'Lines sent to the simulated GSAP';

function lines(count: number): string {
  return count === 1 ? '1 line' : `${count} lines`;
}

export function pendingLines(pr: PurchaseRequest): PurchaseRequestLine[] {
  return pr.lines.filter((line) => line.gsapPending);
}

export function orderedPO(pr: PurchaseRequest): string {
  return pr.lines.find((line) => line.poNumber !== '')?.poNumber ?? '';
}

/** "10,000.00 USD", or "9,200.00 EUR and 1,000.00 USD" for lines in several currencies. */
export function totalText(pending: PurchaseRequestLine[]): string {
  const byCurrency = new Map<string, bigint>();
  for (const line of pending) {
    const { currency, amount } = line.amount;
    byCurrency.set(currency, (byCurrency.get(currency) ?? 0n) + toCents(amount));
  }
  return [...byCurrency]
    .map(([currency, cents]) => formatMoneyText({ amount: formatCents(cents), currency }))
    .join(' and ');
}

export function createText(pr: PurchaseRequest): string {
  const pending = pendingLines(pr);
  return `The simulated GSAP will create a GSAP PR and a purchase order with ${lines(pending.length)} (${totalText(pending)}) for ${pr.id}.`;
}

export function topUpText(pr: PurchaseRequest): string {
  const pending = pendingLines(pr);
  const changed = pending.filter((line) => line.poLine !== '').length;
  const added = pending.length - changed;
  return `The simulated GSAP will update PO ${orderedPO(pr)}: ${lines(changed)} changed, ${lines(added)} added (top-up).`;
}

export function orderedToast(pr: PurchaseRequest): string {
  return `Simulated GSAP created GSAP PR ${pr.gsapPrNumber} and PO ${orderedPO(pr)}`;
}
