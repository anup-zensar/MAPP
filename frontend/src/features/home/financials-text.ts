import { plural } from '@/format/text';

export const FINANCIALS_CARD = {
  open: 'Open Financials',
  committed: 'Committed',
  committedTip: 'The value of purchase orders GSAP has issued for these line items.',
  invoiced: 'Invoiced',
  invoicedTip: 'Supplier invoices GSAP has received against them, less credit notes.',
  paid: 'Paid',
  paidTip: "Payments GSAP has released — Shell's actuals.",
  accrued: 'Accrued',
  noPlannedCost: 'No planned cost yet',
  empty: 'No purchase orders or invoices for your plans yet.',
  noData: 'No GSAP data loaded yet.',
  noAccess: "You don't have access to financial data — ask an administrator.",
  meterLabel: 'Committed of planned cost',
} as const;

export function financialsTitle(year: number): string {
  return `Financials · ${year}`;
}

const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];

/** "Sep 2026" for the accrual month "2026-09" (or a date in it); the text itself otherwise. */
export function monthText(month: string): string {
  const match = /^(\d{4})-(\d{2})/.exec(month);
  const name = match ? MONTHS[Number(match[2]) - 1] : undefined;
  return match && name ? `${name} ${match[1]}` : month;
}

/** Accrued counts only the latest accrual month in the data, so the label names it. */
export function accruedLabel(accrualMonth: string | null): string {
  return accrualMonth ? `Accrued (${monthText(accrualMonth)})` : 'Accrued';
}

export function accruedTip(accrualMonth: string | null): string {
  return accrualMonth
    ? `Money owed for work received but not yet invoiced, as GSAP accrued it for ${monthText(accrualMonth)}.`
    : 'Money owed for work received but not yet invoiced, as GSAP accrued it.';
}

export function ofPlannedText(planned: string): string {
  return `of ${planned} planned`;
}

export function aboveByText(amount: string): string {
  return `Committed above planned cost by ${amount}`;
}

export function aboveThePlannedText(count: number): string {
  return `${plural(count, 'activity', 'activities')} committed above planned cost`;
}

export function notLinkedText(count: number): string {
  return `${plural(count, 'transaction')} not linked to a line item`;
}

export function sourceText(
  source: { dataAsOf: string | null; feedName: string | null; fileName: string | null },
  dateTime: (timestamp: string) => string,
): string {
  if (!source.dataAsOf) {
    return FINANCIALS_CARD.noData;
  }
  const feed = [source.feedName, source.fileName && `'${source.fileName}'`]
    .filter(Boolean)
    .join(' ');
  return `GSAP data as of ${dateTime(source.dataAsOf)}${feed ? ` · ${feed}` : ''}`;
}
