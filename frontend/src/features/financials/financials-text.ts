// The words of the Financials page (015 plan, "User-visible messages"), built here so they read the same in the
// table, the panel and the dialogs.
import { formatWhole } from '@/format/decimal';
import { plural } from '@/format/text';
import { formatMonth } from '@/format/intl';
import type { DateFormat } from '@/format/intl';
import type {
  FinancialImportCounts,
  FinancialImportNotLinkedRow,
  FinancialImportProblem,
  FinancialSource,
  FinancialTransaction,
  FinancialTransactionType,
  FinancialTypeCounts,
} from './api';
import type { FinancialTab } from './financials-links';

export const TAB_LABELS: Record<FinancialTab, string> = {
  all: 'All',
  purchaseOrder: 'Purchase orders',
  invoice: 'Invoices',
  payment: 'Payments',
  accrual: 'Accruals',
  notLinked: 'Not linked',
};

export const TYPE_WORDS: Record<FinancialTransactionType, string> = {
  purchaseOrder: 'Purchase order',
  invoice: 'Invoice',
  payment: 'Payment',
  accrual: 'Accrual',
};

export const NOT_LINKED = 'Not linked';
export const LINE_ITEM_IN_BIN = 'Line item in the recycle bin';

export const NO_DATA = 'No GSAP data loaded yet.';
export const NO_ACCESS = "You don't have access to financial data — ask an administrator.";
export const NO_MATCH = 'No transactions match — clear a filter or search for something else';
export const NOTHING_TO_IMPORT = 'Nothing to import — fix the rows and choose the file again';
// Short enough for the 320 px search field (design system §5.2); the API also searches the description and the
// linked line item's and activity's name.
export const SEARCH_PLACEHOLDER = 'Search number, vendor, code or ID';
export const PAGE_SUBTITLE = 'Purchase orders, invoices, payments and accruals from GSAP';
export const TRANSACTION_GONE = "This GSAP transaction doesn't exist.";

export const TILE_TIPS = {
  committed: 'The value of purchase orders GSAP has issued for these line items.',
  invoiced: 'Supplier invoices GSAP has received against them, less credit notes.',
  paid: "Payments GSAP has released — Shell's actuals.",
  accrued: (month: string | null, format?: DateFormat) =>
    month
      ? `Money owed for work received but not yet invoiced, as GSAP accrued it for ${monthText(month, format)}.`
      : 'Money owed for work received but not yet invoiced, as GSAP accrued it.',
};

/** "Sep 2026", or the month in the person's date format ("09/2026", 038 FR-058). */
export function monthText(month: string, format?: DateFormat): string {
  return formatMonth(month, format);
}

/** "Accrued (Sep 2026)", or "Accrued" when the data has no accruals (edge case). */
export function accruedLabel(month: string | null, format?: DateFormat): string {
  return month ? `Accrued (${monthText(month, format)})` : 'Accrued';
}

export function tabLabel(tab: FinancialTab, counts: FinancialTypeCounts | undefined): string {
  const count = counts?.[tab];
  return count === undefined ? TAB_LABELS[tab] : `${TAB_LABELS[tab]} (${formatWhole(count)})`;
}

export function countLine(shown: number, all: number): string {
  return `${formatWhole(shown)} of ${formatWhole(all)} ${all === 1 ? 'transaction' : 'transactions'}`;
}

export function sourceLine(
  source: FinancialSource,
  dateTime: (timestamp: string) => string,
): string {
  if (!source.dataAsOf) {
    return NO_DATA;
  }
  const feed = source.feedName ?? 'GSAP';
  const file = source.fileName ? ` '${source.fileName}'` : '';
  return `GSAP data as of ${dateTime(source.dataAsOf)} · ${feed}${file}`;
}

type Matchable = Pick<
  FinancialTransaction,
  'link' | 'trackingKey' | 'sentLineItemId' | 'poNumber' | 'swbsCode'
>;

export function matchedByText(transaction: Matchable, date: (timestamp: string) => string) {
  const { link } = transaction;
  switch (link.matchedBy) {
    case 'lineItemId':
      return `Line item ID ${transaction.sentLineItemId || link.lineItemId}`;
    case 'trackingKey':
      return `HX key ${transaction.trackingKey}`;
    case 'poNumber':
      return `PO number ${transaction.poNumber}`;
    case 'swbsCode':
      return `SWBS code ${transaction.swbsCode}`;
    case 'manual':
      return link.manualBy && link.manualAt
        ? `Linked by ${link.manualBy.name} on ${date(link.manualAt)}`
        : 'Linked by hand';
    default:
      return link.reason;
  }
}

export function ruleLinkedNote(method: string): string {
  return `Linked by GSAP data (${method}) — correct it in GSAP.`;
}

export function unlinkQuestion(transactionName: string, lineItem: string): string {
  return `Unlink '${transactionName}' from '${lineItem}'?`;
}

export function importCountsText(counts: FinancialImportCounts): string {
  return (
    `${plural(counts.rows, 'row')}: ${formatWhole(counts.new)} new, ` +
    `${formatWhole(counts.updated)} updated, ${formatWhole(counts.unchanged)} unchanged, ` +
    `${formatWhole(counts.refused)} refused · ${formatWhole(counts.notLinked)} not linked`
  );
}

export function relinkedText(count: number): string {
  return count === 1
    ? '1 earlier transaction is now linked'
    : `${formatWhole(count)} earlier transactions are now linked`;
}

export function refusedText(count: number): string {
  return count === 1
    ? '1 row refused — fix it and import the file again.'
    : `${formatWhole(count)} rows refused — fix them and import the file again.`;
}

export function importedToast(fileName: string, counts: FinancialImportCounts): string {
  return `Imported '${fileName}': ${formatWhole(counts.new)} new, ${formatWhole(counts.updated)} updated · ${formatWhole(counts.notLinked)} not linked`;
}

export function importButtonLabel(counts: FinancialImportCounts): string {
  const rows = counts.rows - counts.refused;
  return `Import ${plural(rows, 'row')}`;
}

export function problemLine(problem: FinancialImportProblem): string {
  return problem.column
    ? `Row ${problem.row} · ${problem.column}: ${problem.message}`
    : `Row ${problem.row}: ${problem.message}`;
}

export function notLinkedLine(row: FinancialImportNotLinkedRow): string {
  return `Row ${row.row} · ${row.name}: ${row.reason}`;
}

export function listHeading(word: string, shown: number, total: number): string {
  return shown < total
    ? `${word} (${formatWhole(total)}, the first ${formatWhole(shown)} shown)`
    : `${word} (${formatWhole(total)})`;
}

export function currencyMissingText(currency: string): string {
  return `Currency ${currency} is no longer set up in Settings — this amount is left out of the totals.`;
}

export function recordLabel(record: { name: string; id: string }): string {
  return `${record.name} (${record.id})`;
}
