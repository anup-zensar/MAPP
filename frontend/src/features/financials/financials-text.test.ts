import { describe, expect, it } from 'vitest';
import {
  FT_1003,
  FT_1004,
  FT_1009,
  manuallyLinked,
  transactionOf,
} from '@/testing/fixtures/financials';
import {
  accruedLabel,
  countLine,
  importButtonLabel,
  importCountsText,
  importedToast,
  listHeading,
  matchedByText,
  monthText,
  notLinkedLine,
  problemLine,
  refusedText,
  relinkedText,
  ruleLinkedNote,
  sourceLine,
  tabLabel,
  unlinkQuestion,
} from './financials-text';

const day = (timestamp: string) => `day(${timestamp.slice(0, 10)})`;
const counts = { rows: 4, new: 2, updated: 1, unchanged: 1, refused: 0, notLinked: 1, relinked: 0 };

describe('financialsText (015 plan, "User-visible messages")', () => {
  it('names the accrual month', () => {
    expect(monthText('2026-09')).toBe('Sep 2026');
    expect(accruedLabel('2026-09')).toBe('Accrued (Sep 2026)');
    expect(accruedLabel(null)).toBe('Accrued');
  });

  it('writes the tabs with their counts and the count line', () => {
    const tabCounts = {
      all: 11,
      purchaseOrder: 4,
      invoice: 3,
      payment: 2,
      accrual: 2,
      notLinked: 1,
    };
    expect(tabLabel('all', tabCounts)).toBe('All (11)');
    expect(tabLabel('purchaseOrder', tabCounts)).toBe('Purchase orders (4)');
    expect(tabLabel('notLinked', undefined)).toBe('Not linked');
    expect(countLine(4, 11)).toBe('4 of 11 transactions');
    expect(countLine(1, 1)).toBe('1 of 1 transaction');
  });

  it('writes the source line (FR-008)', () => {
    const dateTime = () => '19 Oct 2026, 02:00';
    expect(
      sourceLine(
        {
          dataAsOf: '2026-10-19T02:00:00Z',
          feedName: 'Demo data (simulated GSAP)',
          fileName: null,
          accrualMonth: '2026-09',
        },
        dateTime,
      ),
    ).toBe('GSAP data as of 19 Oct 2026, 02:00 · Demo data (simulated GSAP)');
    expect(
      sourceLine(
        {
          dataAsOf: '2026-10-20T09:00:00Z',
          feedName: 'Simulated GSAP extract',
          fileName: 'gsap.csv',
          accrualMonth: null,
        },
        dateTime,
      ),
    ).toBe("GSAP data as of 19 Oct 2026, 02:00 · Simulated GSAP extract 'gsap.csv'");
    expect(
      sourceLine({ dataAsOf: null, feedName: null, fileName: null, accrualMonth: null }, dateTime),
    ).toBe('No GSAP data loaded yet.');
  });

  it('says how a transaction was matched, or why not (FR-022)', () => {
    expect(matchedByText(transactionOf(FT_1003), day)).toBe('HX key HX1006');
    expect(matchedByText(transactionOf(FT_1009), day)).toBe('SWBS code GB01-M26-0006');
    expect(
      matchedByText(
        transactionOf(FT_1003, {
          link: { ...FT_1003.link, matchedBy: 'poNumber' },
          poNumber: '4500001001',
        }),
        day,
      ),
    ).toBe('PO number 4500001001');
    expect(
      matchedByText(
        transactionOf(FT_1003, {
          link: { ...FT_1003.link, matchedBy: 'lineItemId' },
          sentLineItemId: 'L-1006',
        }),
        day,
      ),
    ).toBe('Line item ID L-1006');
    expect(matchedByText(manuallyLinked(), day)).toBe('Linked by Alex Morgan on day(2026-10-20)');
    expect(matchedByText(transactionOf(FT_1004), day)).toBe(
      'No line item for HX key HX1099 — check the tracking key in GSAP.',
    );
    expect(ruleLinkedNote('HX key HX1006')).toBe(
      'Linked by GSAP data (HX key HX1006) — correct it in GSAP.',
    );
    expect(unlinkQuestion('PO 4500001004 / 10', 'Station signage (L-1007)')).toBe(
      "Unlink 'PO 4500001004 / 10' from 'Station signage (L-1007)'?",
    );
  });

  it('writes the import counts, results and lines (US4)', () => {
    expect(importCountsText(counts)).toBe(
      '4 rows: 2 new, 1 updated, 1 unchanged, 0 refused · 1 not linked',
    );
    expect(importCountsText({ ...counts, rows: 1, new: 1, updated: 0, unchanged: 0 })).toBe(
      '1 row: 1 new, 0 updated, 0 unchanged, 0 refused · 1 not linked',
    );
    expect(importButtonLabel(counts)).toBe('Import 4 rows');
    expect(importButtonLabel({ ...counts, refused: 3 })).toBe('Import 1 row');
    expect(relinkedText(2)).toBe('2 earlier transactions are now linked');
    expect(relinkedText(1)).toBe('1 earlier transaction is now linked');
    expect(refusedText(2)).toBe('2 rows refused — fix them and import the file again.');
    expect(importedToast('gsap.csv', counts)).toBe(
      "Imported 'gsap.csv': 2 new, 1 updated · 1 not linked",
    );
    expect(
      problemLine({
        row: 3,
        column: 'amount',
        message: 'Amount must be a number like 1234.50 (a minus sign for credits).',
      }),
    ).toBe('Row 3 · amount: Amount must be a number like 1234.50 (a minus sign for credits).');
    expect(problemLine({ row: 4, column: '', message: 'The row is empty.' })).toBe(
      'Row 4: The row is empty.',
    );
    expect(
      notLinkedLine({
        row: 7,
        name: 'PO 4500001004 / 10',
        reason: 'No line item for HX key HX1099 — check the tracking key in GSAP.',
      }),
    ).toBe(
      'Row 7 · PO 4500001004 / 10: No line item for HX key HX1099 — check the tracking key in GSAP.',
    );
    expect(listHeading('Problems', 2, 2)).toBe('Problems (2)');
    expect(listHeading('Problems', 200, 350)).toBe('Problems (350, the first 200 shown)');
  });
});
