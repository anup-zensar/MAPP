/**
 * Recycle-bin entries and restore previews for the tests of feature 011 (only tests import this
 * file): the seeded initiative "Trade shows" with one activity and one line item inside.
 */
import type { components } from '@/api/schema';

type BinItem = components['schemas']['RecycleBinListItem'];
type RestorePreview = components['schemas']['RestorePreview'];

export const PRIYA_REF = { id: 'U-1002', name: 'Priya Raman' };

export const PLAN_REF = { type: 'plan', id: 'P-1001', name: 'FY2026 Lubricants Europe' };
export const CAMPAIGN_REF = { type: 'campaign', id: 'C-1002', name: 'Helix launch' };

/** The initiative "Trade shows", deleted on 17 Oct 2026, with its activity and line item. */
export function binItem(overrides: Partial<BinItem> = {}): BinItem {
  return {
    id: 'BIN-1001',
    recordType: 'initiative',
    recordTypeLabel: 'initiative',
    recordTypePlural: 'initiatives',
    recordId: 'I-1009',
    recordName: 'Trade shows',
    path: [PLAN_REF, CAMPAIGN_REF],
    parent: CAMPAIGN_REF,
    deletedAt: '2026-10-17T10:00:00Z',
    deletedBy: PRIYA_REF,
    contents: [
      { recordType: 'activity', label: 'activity', count: 1 },
      { recordType: 'lineItem', label: 'line item', count: 1 },
    ],
    restoreState: 'restorable',
    missingParent: null,
    parentEntryIds: [],
    insideEntryIds: [],
    records: [
      {
        recordType: 'initiative',
        recordTypeLabel: 'initiative',
        recordId: 'I-1009',
        recordName: 'Trade shows',
        parentId: 'C-1002',
      },
      {
        recordType: 'activity',
        recordTypeLabel: 'activity',
        recordId: 'A-1042',
        recordName: 'Fleet expo stand',
        parentId: 'I-1009',
      },
      {
        recordType: 'lineItem',
        recordTypeLabel: 'line item',
        recordId: 'L-1012',
        recordName: 'Stand build',
        parentId: 'A-1042',
      },
    ],
    ...overrides,
  };
}

/** A line item deleted on its own, with nothing inside. */
export function lineItemEntry(overrides: Partial<BinItem> = {}): BinItem {
  return binItem({
    id: 'BIN-1002',
    recordType: 'lineItem',
    recordTypeLabel: 'line item',
    recordTypePlural: 'line items',
    recordId: 'L-1020',
    recordName: 'Print ads',
    contents: [],
    records: [
      {
        recordType: 'lineItem',
        recordTypeLabel: 'line item',
        recordId: 'L-1020',
        recordName: 'Print ads',
        parentId: 'A-1001',
      },
    ],
    ...overrides,
  });
}

/** The restore preview of "Trade shows": back into Helix launch, nothing changes. */
export function restorePreview(overrides: Partial<RestorePreview> = {}): RestorePreview {
  return {
    entry: binItem(),
    destination: [PLAN_REF, CAMPAIGN_REF],
    parentEntries: [],
    changes: [],
    notes: [],
    ...overrides,
  };
}
