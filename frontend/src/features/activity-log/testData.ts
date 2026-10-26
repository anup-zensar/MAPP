/**
 * Activity-log entries and summaries for the tests of feature 011 (only tests import this file).
 * Times are in UTC, as the tests run in UTC: "today" is 19 Oct 2026 when a test fixes the clock
 * at TODAY_NOON.
 */
import type { components } from '@/api/generated/schema';

type ActivityLogEntry = components['schemas']['ActivityLogEntry'];
type ActivityLogPage = components['schemas']['ActivityLogPage'];
type ActivityLogSummary = components['schemas']['ActivityLogSummary'];

/** Monday 19 Oct 2026, 12:00 UTC. */
export const TODAY_NOON = new Date('2026-10-19T12:00:00Z');

export const PRIYA_REF = { id: 'U-1002', name: 'Priya Raman' };
export const DANIEL_REF = { id: 'U-1003', name: 'Daniel Okoro' };

const PLAN_REF = { type: 'plan', id: 'P-1001', name: 'FY2026 Lubricants Europe' };
const INITIATIVE_REF = { type: 'initiative', id: 'I-1003', name: 'Helix launch' };

/** An entry with every field; pass what a test needs to differ. */
export function logEntry(overrides: Partial<ActivityLogEntry> = {}): ActivityLogEntry {
  return {
    id: 'LOG-1042',
    seq: 1042,
    at: '2026-10-19T09:15:00Z',
    actor: PRIYA_REF,
    action: 'changed',
    verb: 'changed',
    recordType: 'activity',
    recordTypeLabel: 'activity',
    recordId: 'A-1042',
    recordName: 'Film shoot',
    path: [PLAN_REF, INITIATIVE_REF],
    changes: [],
    comment: '',
    details: '',
    items: [],
    recordNow: {
      state: 'exists',
      name: 'Film shoot',
      path: [PLAN_REF, INITIATIVE_REF],
      binEntryId: '',
    },
    ...overrides,
  };
}

/** A page of the log as GET /api/v1/activity-log answers it. */
export function logPage(
  items: ActivityLogEntry[],
  overrides: Partial<ActivityLogPage> = {},
): ActivityLogPage {
  return {
    items,
    total: items.length,
    unfilteredTotal: items.length,
    record: null,
    ...overrides,
  };
}

/** GET /api/v1/activity-log/summary with two people, two record types and one plan. */
export function logSummary(overrides: Partial<ActivityLogSummary> = {}): ActivityLogSummary {
  return {
    changesToday: 5,
    changesThisWeek: 23,
    peopleActiveThisWeek: 2,
    deletionsThisWeek: 1,
    total: 140,
    people: [DANIEL_REF, PRIYA_REF],
    recordTypes: [
      { type: 'activity', label: 'activity', count: 90 },
      { type: 'lineItem', label: 'line item', count: 50 },
    ],
    plans: [PLAN_REF],
    ...overrides,
  };
}

/**
 * The History section's request with an empty log (feature 011, FR-011): for the tests of the plan,
 * campaign, initiative, activity and line item pages, which all show the section.
 */
export const HISTORY_ANSWERS = {
  'GET /api/v1/activity-log': { body: logPage([]) },
};
