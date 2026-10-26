import { describe, expect, it } from 'vitest';
import { ApiError } from '@/api/errors';
import { binItem, lineItemEntry } from '@/testing/fixtures/recycle-bin';
import {
  alreadyResolvedText,
  emptiedMessage,
  emptyBinText,
  insideEntriesWarning,
  purgeContentsText,
  purgeTitle,
  purgedMessage,
} from './purge-text';

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
