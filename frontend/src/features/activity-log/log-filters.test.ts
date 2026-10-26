import { describe, expect, it } from 'vitest';
import {
  CUSTOM_RANGE_ERROR,
  NO_FILTERS,
  apiQuery,
  customRangeError,
  filtersToParams,
  isFiltered,
  rangeBounds,
  todayStart,
  weekStart,
} from './log-filters';
import type { LogFilterState } from './log-filters';
import { filtersFromParams } from './schemas';

/** Runs `check` with the computer's clock in another time zone (the tests run in UTC). */
function inTimeZone(timeZone: string, check: () => void): void {
  const original = process.env.TZ;
  process.env.TZ = timeZone;
  try {
    check();
  } finally {
    if (original === undefined) {
      delete process.env.TZ;
    } else {
      process.env.TZ = original;
    }
  }
}

/** Thursday 22 Oct 2026, 15:30 UTC. */
const THURSDAY = new Date('2026-10-22T15:30:00Z');

function filters(overrides: Partial<LogFilterState>): LogFilterState {
  return { ...NO_FILTERS, ...overrides };
}

describe('address ⇄ filters ⇄ API query (FR-003)', () => {
  it('reads every filter from the address and writes the same address back', () => {
    const address =
      'person=U-1003&type=activity&action=deleted&action=deletedPermanently&plan=P-1001' +
      '&record=A-1042&inside=1&range=custom&from=2026-10-01&to=2026-10-12&q=film';
    const state = filtersFromParams(new URLSearchParams(address));

    expect(state).toEqual({
      person: 'U-1003',
      type: 'activity',
      actions: ['deleted', 'deletedPermanently'],
      plan: 'P-1001',
      record: 'A-1042',
      inside: true,
      range: 'custom',
      from: '2026-10-01',
      to: '2026-10-12',
      q: 'film',
    });
    expect(filtersToParams(state).toString()).toBe(address);
  });

  it('leaves out what it does not know', () => {
    const state = filtersFromParams(
      new URLSearchParams('action=sneezed&range=decade&inside=1&from=2026-10-01'),
    );

    expect(state).toEqual(NO_FILTERS);
    expect(filtersToParams(NO_FILTERS).toString()).toBe('');
  });

  it('asks the API for each filter in use, with the search trimmed', () => {
    const state = filters({
      person: 'U-1003',
      type: 'activity',
      actions: ['deleted'],
      plan: 'P-1001',
      record: 'A-1042',
      inside: true,
      q: ' film ',
    });

    expect(apiQuery(state, THURSDAY)).toEqual({
      actorId: 'U-1003',
      recordType: 'activity',
      action: ['deleted'],
      planId: 'P-1001',
      recordId: 'A-1042',
      includeInside: true,
      q: 'film',
    });
    expect(apiQuery(NO_FILTERS, THURSDAY)).toEqual({});
  });

  it('sends nothing while the custom range ends before it starts, and says why', () => {
    const wrong = filters({ range: 'custom', from: '2026-10-12', to: '2026-10-01' });

    expect(customRangeError(wrong)).toBe(CUSTOM_RANGE_ERROR);
    expect(apiQuery(wrong, THURSDAY)).toBeNull();
    expect(
      customRangeError(filters({ range: 'custom', from: '2026-10-12', to: '2026-10-12' })),
    ).toBeUndefined();
  });

  it('counts as filtered once a filter is in use', () => {
    expect(isFiltered(NO_FILTERS)).toBe(false);
    expect(isFiltered(filters({ range: 'custom', from: '2026-10-01' }))).toBe(false);
    expect(isFiltered(filters({ range: 'week' }))).toBe(true);
    expect(isFiltered(filters({ q: 'film' }))).toBe(true);
    expect(isFiltered(filters({ q: '  ' }))).toBe(false);
  });
});

describe('date ranges in the person’s time zone (FR-002, FR-003)', () => {
  it('starts today, this week (Monday), the last 30 days and this year at local midnight', () => {
    expect(rangeBounds(filters({ range: 'today' }), THURSDAY)).toEqual({
      from: '2026-10-22T00:00:00.000Z',
    });
    expect(rangeBounds(filters({ range: 'week' }), THURSDAY)).toEqual({
      from: '2026-10-19T00:00:00.000Z',
    });
    expect(rangeBounds(filters({ range: '30d' }), THURSDAY)).toEqual({
      from: '2026-09-23T00:00:00.000Z',
    });
    expect(rangeBounds(filters({ range: 'year' }), THURSDAY)).toEqual({
      from: '2026-01-01T00:00:00.000Z',
    });
  });

  it('ends a custom range at the midnight after its last day', () => {
    const custom = filters({ range: 'custom', from: '2026-10-01', to: '2026-10-12' });

    expect(apiQuery(custom, THURSDAY)).toEqual({
      from: '2026-10-01T00:00:00.000Z',
      to: '2026-10-13T00:00:00.000Z',
    });
  });

  it('takes Monday as the week’s first day, also on a Sunday and on a Monday', () => {
    expect(weekStart(new Date('2026-10-25T20:00:00Z'))).toBe('2026-10-19T00:00:00.000Z');
    expect(weekStart(new Date('2026-10-19T00:30:00Z'))).toBe('2026-10-19T00:00:00.000Z');
    expect(todayStart(THURSDAY)).toBe('2026-10-22T00:00:00.000Z');
  });

  it('uses the local midnight in another time zone', () => {
    inTimeZone('Asia/Kolkata', () => {
      // 15:30 UTC is 21:00 in India, still Thursday 22 Oct there.
      expect(todayStart(THURSDAY)).toBe('2026-10-21T18:30:00.000Z');
      expect(weekStart(THURSDAY)).toBe('2026-10-18T18:30:00.000Z');
      expect(
        apiQuery(filters({ range: 'custom', from: '2026-10-01', to: '2026-10-12' }), THURSDAY),
      ).toEqual({ from: '2026-09-30T18:30:00.000Z', to: '2026-10-12T18:30:00.000Z' });
    });
    inTimeZone('America/New_York', () => {
      // 01:00 UTC on Monday is still Sunday evening in New York: the week began the Monday before.
      const sundayEvening = new Date('2026-10-26T01:00:00Z');
      expect(todayStart(sundayEvening)).toBe('2026-10-25T04:00:00.000Z');
      expect(weekStart(sundayEvening)).toBe('2026-10-19T04:00:00.000Z');
    });
  });
});
