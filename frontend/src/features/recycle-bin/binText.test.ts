import { describe, expect, it } from 'vitest';
import { ApiError } from '@/api/errors';
import {
  alreadyResolvedText,
  blockedText,
  changeSentence,
  contentsText,
  destinationSentence,
  emptiedMessage,
  emptyBinText,
  insideEntriesWarning,
  matchSearch,
  parentEntryLine,
  parentsIntro,
  purgeContentsText,
  purgeTitle,
  purgedMessage,
  restoreConfirmLabel,
  restoreStateLine,
  restoreTitle,
  restoredHref,
  restoredMessage,
  rowContentsText,
  showingLine,
  typePills,
  wasInText,
} from './binText';
import type { RestorePreviewChange } from './binText';
import { PLAN_REF, binItem, lineItemEntry, restorePreview } from './testData';

const PARENT = { type: 'initiative', typeLabel: 'initiative', id: 'I-1009', name: 'Trade shows' };

describe('rows (FR-014, FR-015, FR-017)', () => {
  it('counts the items shown of all, singular for one (FR-016)', () => {
    expect(showingLine(3, 12)).toBe('3 of 12 items');
    expect(showingLine(1, 1)).toBe('1 of 1 item');
    expect(showingLine(1250, 1250)).toBe('1,250 of 1,250 items');
  });

  it('says what was inside, or Nothing inside', () => {
    expect(rowContentsText(binItem().contents)).toBe('1 activity · 1 line item');
    expect(rowContentsText([])).toBe('Nothing inside');
    expect(
      contentsText([
        { recordType: 'activity', label: 'activities', count: 2 },
        { recordType: 'lineItem', label: 'line items', count: 5 },
      ]),
    ).toBe('2 activities and 5 line items');
  });

  it('says where it was, nothing for a top-level record', () => {
    expect(wasInText(binItem().path)).toBe('Was in FY2026 Lubricants Europe › Helix launch');
    expect(wasInText([])).toBe('');
  });

  it('says why a row cannot simply come back', () => {
    expect(restoreStateLine(binItem())).toBe('');
    expect(
      restoreStateLine(binItem({ restoreState: 'parentInRecycleBin', missingParent: PARENT })),
    ).toBe("Its initiative 'Trade shows' is also in the recycle bin");
    expect(
      restoreStateLine(
        binItem({ restoreState: 'parentDeletedPermanently', missingParent: PARENT }),
      ),
    ).toBe("Can't be restored — its initiative 'Trade shows' was permanently deleted");
  });
});

describe('search and type pills (FR-016)', () => {
  it('finds an entry by its own name or ID, ignoring letter case', () => {
    expect(matchSearch(binItem(), 'trade')).toEqual({ matches: true, containsText: '' });
    expect(matchSearch(binItem(), 'i-1009')).toEqual({ matches: true, containsText: '' });
    expect(matchSearch(binItem(), '')).toEqual({ matches: true, containsText: '' });
    expect(matchSearch(binItem(), 'zzz')).toEqual({ matches: false, containsText: '' });
  });

  it('finds an entry by a record inside it and says which', () => {
    expect(matchSearch(binItem(), 'a-1042')).toEqual({
      matches: true,
      containsText: "Contains A-1042 'Fleet expo stand'",
    });
    expect(matchSearch(binItem(), 'ST')).toEqual({
      matches: true,
      containsText: "Contains A-1042 'Fleet expo stand' and 1 more",
    });
  });

  it('offers the five levels and any other type in the bin', () => {
    const currency = binItem({
      recordType: 'currency',
      recordTypeLabel: 'currency',
      recordTypePlural: 'currencies',
    });

    expect(typePills([binItem(), currency]).map((pill) => pill.label)).toEqual([
      'All',
      'Plans',
      'Campaigns',
      'Initiatives',
      'Activities',
      'Line items',
      'Currencies',
    ]);
  });
});

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

describe('delete permanently and empty (FR-028 – FR-030, FR-032)', () => {
  it('says what goes with the record and what else can no longer come back', () => {
    expect(purgeTitle(binItem())).toBe("Delete 'Trade shows' permanently?");
    expect(purgeContentsText(binItem())).toBe('This also deletes 1 activity and 1 line item.');
    expect(purgeContentsText(lineItemEntry())).toBe('');
    expect(insideEntriesWarning(0)).toBe('');
    expect(insideEntriesWarning(1)).toBe(
      '1 other item in the recycle bin belongs inside it and can no longer be restored.',
    );
    expect(insideEntriesWarning(2)).toBe(
      '2 other items in the recycle bin belong inside it and can no longer be restored.',
    );
    expect(purgedMessage('initiative', 'Trade shows')).toBe(
      "Initiative 'Trade shows' deleted permanently",
    );
  });

  it('adds up what emptying deletes', () => {
    expect(emptyBinText([binItem(), binItem({ id: 'BIN-1003' }), lineItemEntry()])).toBe(
      'This permanently deletes 3 items and everything inside them: 2 activities and 2 line items.',
    );
    expect(emptyBinText([binItem()])).toBe(
      'This permanently deletes 1 item and everything inside them: 1 activity and 1 line item.',
    );
    expect(emptyBinText([lineItemEntry()])).toBe('This permanently deletes 1 item.');
  });

  it('says how many items emptying deleted', () => {
    expect(emptiedMessage(3)).toBe('The recycle bin is empty — 3 items deleted permanently');
    expect(emptiedMessage(1)).toBe('The recycle bin is empty — 1 item deleted permanently');
    expect(emptiedMessage(0)).toBe(
      'Nothing was deleted — someone else already emptied the recycle bin.',
    );
  });

  it('says who already restored or deleted an entry, and when (FR-032)', () => {
    const resolved = (how: 'restored' | 'deletedPermanently') =>
      new ApiError('alreadyResolved', {
        message: 'Already resolved.',
        resolution: { how, by: { id: 'U-1003', name: 'Daniel Okoro' }, at: '2026-03-12T14:05:00Z' },
      });

    expect(alreadyResolvedText(resolved('deletedPermanently'), 'Trade shows')).toBe(
      "'Trade shows' is no longer in the recycle bin — Daniel Okoro deleted it permanently on 12 Mar 2026 at 14:05",
    );
    expect(alreadyResolvedText(resolved('restored'), 'Trade shows')).toBe(
      "'Trade shows' is no longer in the recycle bin — Daniel Okoro restored it on 12 Mar 2026 at 14:05",
    );
  });
});
