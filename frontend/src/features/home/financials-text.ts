import { plural } from '@/format/text';
import { formatMonth } from '@/format/intl';
import type { DateFormat } from '@/format/intl';

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
  meterLabel: 'Committed of planned cost',
} as const;

export function financialsTitle(year: number): string {
  return `Financials · ${year}`;
}

/** "Sep 2026" (or "09/2026") for "2026-09" or a date in it; the text otherwise. */
export function monthText(month: string, format?: DateFormat): string {
  const match = /^(\d{4})-(0[1-9]|1[0-2])/.exec(month);
  return match ? formatMonth(match[0], format) : month;
}

/** "Accrued (Sep 2026)": Accrued counts only the latest accrual month in the data. */
export function accruedLabel(accrualMonth: string | null, format?: DateFormat): string {
  return accrualMonth ? `Accrued (${monthText(accrualMonth, format)})` : 'Accrued';
}

export function accruedTip(accrualMonth: string | null, format?: DateFormat): string {
  return accrualMonth
    ? `Money owed for work received but not yet invoiced, as GSAP accrued it for ${monthText(accrualMonth, format)}.`
    : 'Money owed for work received but not yet invoiced, as GSAP accrued it.';
}

export function ofPlannedText(planned: string): string {
  return `of ${planned} planned`;
}

/** "Committed above planned cost by $2,000.00" (015 FR-013: words, not colour alone). */
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
