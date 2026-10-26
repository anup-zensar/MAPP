import { describe, expect, it } from 'vitest';
import { PLAN_REF, binItem, lineItemEntry, restorePreview } from '@/testing/fixtures/recycle-bin';
import {
  blockedText,
  changeSentence,
  destinationSentence,
  parentEntryLine,
  parentsIntro,
  restoreConfirmLabel,
  restoreTitle,
  restoredHref,
  restoredMessage,
} from './restore-text';
import type { RestorePreviewChange } from './restore-text';

const PARENT = { type: 'initiative', typeLabel: 'initiative', id: 'I-1009', name: 'Trade shows' };

describe('restore dialog (FR-018 – FR-024, FR-026)', () => {
  it('names the record and the confirm button', () => {
    expect(restoreTitle(binItem())).toBe("Restore initiative 'Trade shows'?");
    expect(restoreConfirmLabel(binItem(), 0)).toBe('Restore initiative');
    expect(restoreConfirmLabel(binItem(), 1)).toBe('Restore both');
    expect(restoreConfirmLabel(binItem(), 2)).toBe('Restore all 3');
  });

  it('says where it returns, with its contents', () => {
    expect(destinationSentence(restorePreview())).toBe(
      'It returns to FY2026 Lubricants Europe › Helix launch with 1 activity and 1 line item.',
    );
    expect(
      destinationSentence(
        restorePreview({
          destination: [],
          entry: binItem({ recordType: 'plan', recordTypeLabel: 'plan', contents: [] }),
        }),
      ),
    ).toBe('It returns to the Plans list.');
    expect(destinationSentence(restorePreview({ destination: [], entry: lineItemEntry() }))).toBe(
      'It returns to its list.',
    );
  });

  const change = (overrides: Partial<RestorePreviewChange>): RestorePreviewChange => ({
    recordType: 'activity',
    recordTypeLabel: 'activity',
    recordId: 'A-1042',
    recordName: 'Fleet expo stand',
    restored: true,
    field: 'status',
    label: 'Status',
    oldText: 'Approved',
    newText: 'Created',
    ...overrides,
  });

  it.each([
    [
      change({}),
      "Activity 'Fleet expo stand' (A-1042) comes back as Created and must be submitted for approval again.",
    ],
    [
      change({ recordType: 'plan', recordTypeLabel: 'plan', recordId: 'I-1009', newText: 'Draft' }),
      'It comes back as Draft and must be published again.',
    ],
    [
      change({ restored: false, oldText: 'Approved', newText: 'Created' }),
      "Activity 'Fleet expo stand' (A-1042) goes back to Created because its line items change, and must be submitted for approval again.",
    ],
    [
      change({ recordType: 'lineItem', recordTypeLabel: 'line item', recordId: 'L-1012' }),
      "Line item 'Fleet expo stand' (L-1012) changes from Approved to Created.",
    ],
    [
      change({
        recordId: 'I-1009',
        field: 'name',
        label: 'Name',
        oldText: 'Trade shows',
        newText: 'Trade shows (restored)',
      }),
      "It comes back as 'Trade shows (restored)' because the name 'Trade shows' is taken.",
    ],
    [
      change({ field: 'owner', label: 'Owner', oldText: 'Ana', newText: 'Priya Raman' }),
      "Activity 'Fleet expo stand' (A-1042): Owner changes from Ana to Priya Raman.",
    ],
  ])('says what the restore changes (%#)', (restoreChange, sentence) => {
    expect(changeSentence(restoreChange, binItem())).toBe(sentence);
  });

  it('names the parents that come back first (FR-023)', () => {
    expect(parentsIntro(PARENT)).toBe(
      "Its initiative 'Trade shows' is also in the recycle bin. Restoring brings these back first:",
    );
    expect(parentEntryLine(binItem())).toBe(
      "Initiative 'Trade shows' (I-1009) with 1 activity and 1 line item",
    );
    expect(parentEntryLine(lineItemEntry())).toBe("Line item 'Print ads' (L-1020)");
  });

  it('explains a record whose parent was deleted permanently (FR-024)', () => {
    expect(blockedText(lineItemEntry({ missingParent: PARENT }))).toBe(
      "This line item can't be restored because its initiative 'Trade shows' was permanently deleted. You can delete it permanently instead.",
    );
  });

  it('names the restored record and leads to its page (FR-026)', () => {
    expect(restoredMessage('initiative', 'Trade shows')).toBe("Initiative 'Trade shows' restored");
    expect(restoredHref('initiative', 'I-1009', [PLAN_REF])).toBe(
      '/plans/P-1001/initiatives/I-1009',
    );
    expect(restoredHref('plan', 'P-1001', [])).toBe('/plans/P-1001');
    expect(restoredHref('campaign', 'C-1', [])).toBeUndefined();
    expect(restoredHref('currency', 'EUR', [])).toBeUndefined();
    expect(restoredHref('globalBudget', 'GB-1001', [])).toBe('/budgets/GB-1001');
  });
});
