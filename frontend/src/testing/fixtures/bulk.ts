import type { components } from '@/api/generated/schema';

type BulkActionResult = components['schemas']['BulkActionResult'];
type BulkOutcome = components['schemas']['BulkOutcome'];

export function outcome(overrides: Partial<BulkOutcome> = {}): BulkOutcome {
  return {
    recordId: 'A-1001',
    level: 'activity',
    name: 'Launch film',
    planId: 'P-1001',
    outcome: 'done',
    reasons: [],
    ...overrides,
  };
}

export function bulkResult(overrides: Partial<BulkActionResult> = {}): BulkActionResult {
  return {
    id: null,
    preview: true,
    action: 'changeOwner',
    verb: 'changed',
    note: 'Bulk change of 1 activity',
    selected: { campaigns: 0, initiatives: 0, activities: 1 },
    counts: { done: 1, skipped: 0, failed: 0 },
    outcomes: [outcome()],
    reapproval: [],
    effects: [],
    deleteSummary: null,
    recycleBinEntries: [],
    undoAvailableUntil: null,
    ...overrides,
  };
}
