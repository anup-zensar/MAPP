/**
 * The words of the Financials page (015 plan, "User-visible messages"): every sentence built from
 * data is made here, so it reads the same in the table, the panel and the dialogs.
 */
import { formatWhole } from '@/format/decimal';
import { plural } from '@/format/text';
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

/** The tabs' names (FR-021). */
export const TAB_LABELS: Record<FinancialTab, string> = {
  all: 'All',
  purchaseOrder: 'Purchase orders',
  invoice: 'Invoices',
  payment: 'Payments',
  accrual: 'Accruals',
  notLinked: 'Not linked',
};

/** The type chip of a row and the panel. */
export const TYPE_WORDS: Record<FinancialTransactionType, string> = {
  purchaseOrder: 'Purchase order',
  invoice: 'Invoice',
  payment: 'Payment',
  accrual: 'Accrual',
};

/** "Not linked" and "Line item in the recycle bin" (the chips of the Linked to column). */
export const NOT_LINKED = 'Not linked';
export const LINE_ITEM_IN_BIN = 'Line item in the recycle bin';

export const NO_DATA = 'No GSAP data loaded yet.';
export const NO_ACCESS = "You don't have access to financial data — ask an administrator.";
export const NO_MATCH = 'No transactions match — clear a filter or search for something else';
export const NOTHING_TO_IMPORT = 'Nothing to import — fix the rows and choose the file again';
/**
 * Short enough for the 320 px search field (design system §5.2); the API also searches the
 * description and the linked line item's and activity's name (plan 015's longer wording did not fit).
 */
export const SEARCH_PLACEHOLDER = 'Search number, vendor, code or ID';
export const PAGE_SUBTITLE = 'Purchase orders, invoices, payments and accruals from GSAP';
export const TRANSACTION_GONE = "This GSAP transaction doesn't exist.";

/** The ⓘ of the four summary cards (plan, "User-visible messages"). */
export const TILE_TIPS = {
  committed: 'The value of purchase orders GSAP has issued for these line items.',
  invoiced: 'Supplier invoices GSAP has received against them, less credit notes.',
  paid: "Payments GSAP has released — Shell's actuals.",
  accrued: (month: string | null) =>
    month
      ? `Money owed for work received but not yet invoiced, as GSAP accrued it for ${monthText(month)}.`
      : 'Money owed for work received but not yet invoiced, as GSAP accrued it.',
};

const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];

/** "Sep 2026" for "2026-09". */
export function monthText(month: string): string {
  const [year, number] = month.split('-');
  const name = MONTHS[Number(number) - 1];
  return name ? `${name} ${year}` : month;
}

/** "Accrued (Sep 2026)", or "Accrued" when the data has no accruals (edge case). */
export function accruedLabel(month: string | null): string {
  return month ? `Accrued (${monthText(month)})` : 'Accrued';
}

/** "All (11)" — a tab's name with its count for the other filters. */
export function tabLabel(tab: FinancialTab, counts: FinancialTypeCounts | undefined): string {
  return counts ? `${TAB_LABELS[tab]} (${formatWhole(counts[tab])})` : TAB_LABELS[tab];
}

/** "4 of 11 transactions" at the end of the toolbar (design system §5.2). */
export function countLine(shown: number, all: number): string {
  return `${formatWhole(shown)} of ${formatWhole(all)} ${all === 1 ? 'transaction' : 'transactions'}`;
}

/**
 * "GSAP data as of 19 Oct 2026, 02:00 · Demo data (simulated GSAP)", or "… · Simulated GSAP
 * extract 'gsap.csv'"; "No GSAP data loaded yet." before any import (FR-008).
 */
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

/**
 * How a transaction was linked (FR-022): "HX key HX1006", "PO number 4500001001", "Linked by Alex
 * Morgan on 20 Oct 2026"; the not-linked reason when it is not linked; null while there is none.
 */
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

/** "Linked by GSAP data (HX key HX1006) — correct it in GSAP." for a link made by the rules. */
export function ruleLinkedNote(method: string): string {
  return `Linked by GSAP data (${method}) — correct it in GSAP.`;
}

/** "Unlink 'PO 4500001004 / 10' from 'Station signage (L-1007)'?" */
export function unlinkQuestion(transactionName: string, lineItem: string): string {
  return `Unlink '${transactionName}' from '${lineItem}'?`;
}

/** "4 rows: 2 new, 1 updated, 1 unchanged, 0 refused · 1 not linked" (FR-004). */
export function importCountsText(counts: FinancialImportCounts): string {
  return (
    `${plural(counts.rows, 'row')}: ${formatWhole(counts.new)} new, ` +
    `${formatWhole(counts.updated)} updated, ${formatWhole(counts.unchanged)} unchanged, ` +
    `${formatWhole(counts.refused)} refused · ${formatWhole(counts.notLinked)} not linked`
  );
}

/** "2 earlier transactions are now linked" (US4 scenario 8). */
export function relinkedText(count: number): string {
  return count === 1
    ? '1 earlier transaction is now linked'
    : `${formatWhole(count)} earlier transactions are now linked`;
}

/** "2 rows refused — fix them and import the file again." (US4 scenario 3). */
export function refusedText(count: number): string {
  return count === 1
    ? '1 row refused — fix it and import the file again.'
    : `${formatWhole(count)} rows refused — fix them and import the file again.`;
}

/** "Imported 'gsap.csv': 2 new, 1 updated · 1 not linked" (the toast after an import). */
export function importedToast(fileName: string, counts: FinancialImportCounts): string {
  return `Imported '${fileName}': ${formatWhole(counts.new)} new, ${formatWhole(counts.updated)} updated · ${formatWhole(counts.notLinked)} not linked`;
}

/** "Import 4 rows" — the import button names how many rows it will apply. */
export function importButtonLabel(counts: FinancialImportCounts): string {
  const rows = counts.rows - counts.refused;
  return `Import ${plural(rows, 'row')}`;
}

/** "Row 3 · amount: Amount must be a number like 1234.50 (a minus sign for credits)". */
export function problemLine(problem: FinancialImportProblem): string {
  return problem.column
    ? `Row ${problem.row} · ${problem.column}: ${problem.message}`
    : `Row ${problem.row}: ${problem.message}`;
}

/** "Row 7 · PO 4500001004 / 10: No line item for HX key HX1099 — check the tracking key in GSAP". */
export function notLinkedLine(row: FinancialImportNotLinkedRow): string {
  return `Row ${row.row} · ${row.name}: ${row.reason}`;
}

/** "Problems (2)", with "showing 200" when the API capped the list. */
export function listHeading(word: string, shown: number, total: number): string {
  return shown < total
    ? `${word} (${formatWhole(total)}, the first ${formatWhole(shown)} shown)`
    : `${word} (${formatWhole(total)})`;
}

/** "Currency CHF is no longer set up in Settings — this amount is left out of the totals." */
export function currencyMissingText(currency: string): string {
  return `Currency ${currency} is no longer set up in Settings — this amount is left out of the totals.`;
}

/** "Station signage (L-1007)". */
export function recordLabel(record: { name: string; id: string }): string {
  return `${record.name} (${record.id})`;
}
