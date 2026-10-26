import { describe, expect, it } from 'vitest';
import { binItem } from '@/testing/fixtures/recycle-bin';
import {
  contentsText,
  matchSearch,
  restoreStateLine,
  rowContentsText,
  showingLine,
  typePills,
  wasInText,
} from './bin-text';

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
