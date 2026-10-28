import fc from 'fast-check';
import { describe, expect, it } from 'vitest';
import {
  DEFAULT_LIST_SEARCH,
  LIST_SORTS,
  readListSearch,
  toListQuery,
  writeListSearch,
} from './schemas';
import type { ListSearch } from './schemas';

const STATUSES = ['created', 'submitted', 'inReview', 'declined', 'approved', 'closed'] as const;

const validSearch: fc.Arbitrary<ListSearch> = fc.record({
  q: fc.string({ maxLength: 120 }).filter((text) => text.trim() === text),
  status: fc.uniqueArray(fc.constantFrom(...STATUSES), { maxLength: 6 }),
  type: fc.uniqueArray(fc.stringMatching(/^SO-[0-9]{4}$/), { maxLength: 3 }),
  typeNotCompleted: fc.boolean(),
  vendor: fc.uniqueArray(fc.stringMatching(/^[A-Za-z][A-Za-z ]{0,20}[A-Za-z]$/), {
    maxLength: 3,
  }),
  vendorNotCompleted: fc.boolean(),
  owner: fc.uniqueArray(fc.stringMatching(/^U-[0-9]{4}$/), { maxLength: 3 }),
  from: fc.constantFrom('', '2026-01-31', '2026-10-27'),
  to: fc.constantFrom('', '2026-12-31'),
  waiting: fc.boolean(),
  sort: fc.constantFrom(...LIST_SORTS),
  page: fc.integer({ min: 1, max: 500 }),
});

describe('032 FR-045: list URL state', () => {
  it('reads any query string into a usable search', () => {
    fc.assert(
      fc.property(fc.string(), (text) => {
        const search = readListSearch(new URLSearchParams(text));
        expect(search.page).toBeGreaterThanOrEqual(1);
        expect(LIST_SORTS).toContain(search.sort);
        expect(search.status.every((status) => STATUSES.includes(status))).toBe(true);
        expect(search.q.length).toBeLessThanOrEqual(120);
        expect(
          [search.from, search.to].every((day) => day === '' || /^\d{4}-\d{2}-\d{2}$/.test(day)),
        ).toBe(true);
      }),
    );
  });

  it('falls back to the default of each value it cannot use', () => {
    const search = readListSearch(
      new URLSearchParams(
        'status=lost&status=approved&sort=colour&page=0&from=2026-02-31&to=soon&waiting=maybe',
      ),
    );

    expect(search).toEqual({ ...DEFAULT_LIST_SEARCH, status: ['approved'] });
  });

  it('writes and reads back every valid search, defaults left out of the address', () => {
    fc.assert(
      fc.property(validSearch, (search) => {
        expect(readListSearch(writeListSearch(search))).toEqual(search);
      }),
    );
    expect(writeListSearch(DEFAULT_LIST_SEARCH).toString()).toBe('');
  });

  it('asks the API for the search, 50 purchase requests a page', () => {
    expect(
      toListQuery({
        ...DEFAULT_LIST_SEARCH,
        q: ' HX1006 ',
        status: ['approved'],
        vendorNotCompleted: true,
        from: '2026-10-01',
        waiting: true,
        page: 2,
      }),
    ).toEqual({
      q: 'HX1006',
      status: ['approved'],
      vendorNotCompleted: true,
      createdFrom: '2026-10-01',
      waitingForGsap: true,
      sort: 'id',
      page: 2,
      pageSize: 50,
    });
  });
});
