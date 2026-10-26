import { describe, expect, it } from 'vitest';
import { formatDate } from '@/format/intl';
import { logSummary } from '@/testing/fixtures/activity-log';
import { filterChips, removeFilter } from './filter-chips';
import { NO_FILTERS } from './log-filters';
import type { LogFilterState } from './log-filters';

function filters(overrides: Partial<LogFilterState>): LogFilterState {
  return { ...NO_FILTERS, ...overrides };
}

describe('chips (FR-003)', () => {
  const summary = logSummary();
  const names = {
    people: summary.people,
    recordTypes: summary.recordTypes,
    plans: summary.plans,
    record: { type: 'activity', id: 'A-1042', name: 'Film shoot' },
    formatDate: (date: string) => formatDate(date),
  };

  it('names every filter in use', () => {
    const state = filters({
      person: 'U-1003',
      type: 'lineItem',
      actions: ['deleted', 'deletedPermanently'],
      plan: 'P-1001',
      record: 'A-1042',
      inside: true,
      range: 'week',
      q: 'film',
    });

    expect(filterChips(state, names).map((chip) => chip.label)).toEqual([
      'Person: Daniel Okoro',
      'Type: Line item',
      'Action: Deleted',
      'Action: Permanently deleted',
      'Plan: FY2026 Lubricants Europe',
      "Record: Activity 'Film shoot' (A-1042) and items inside",
      'Date: This week',
      "Search: 'film'",
    ]);
  });

  it('shows a custom range with both dates, and a record without items inside', () => {
    const state = filters({
      record: 'A-1042',
      range: 'custom',
      from: '2026-10-01',
      to: '2026-10-12',
    });

    expect(filterChips(state, names).map((chip) => chip.label)).toEqual([
      "Record: Activity 'Film shoot' (A-1042)",
      'Date: 01 Oct 2026 – 12 Oct 2026',
    ]);
  });

  it('takes one filter away per chip', () => {
    const state = filters({ actions: ['deleted', 'moved'], record: 'A-1042', inside: true });

    expect(removeFilter(state, 'action:deleted').actions).toEqual(['moved']);
    expect(removeFilter(state, 'record')).toMatchObject({ record: '', inside: false });
    expect(
      removeFilter(filters({ range: 'custom', from: '2026-10-01', to: '2026-10-02' }), 'date'),
    ).toEqual(NO_FILTERS);
  });
});
