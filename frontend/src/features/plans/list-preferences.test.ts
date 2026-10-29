import fc from 'fast-check';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { z } from 'zod';
import {
  DEFAULT_FILTERS,
  readFilters,
  readRemembered,
  remember,
  toSearchParams,
} from './list-preferences';
import type { PlanListFilters } from './list-preferences';
import { PLAN_TYPES } from './plan-currency';

const VALID = z
  .strictObject({
    status: z.enum(['all', 'draft', 'published']),
    yearFrom: z.int().nullable(),
    yearTo: z.int().nullable(),
    types: z.array(z.enum(PLAN_TYPES)),
    scope: z.enum(['all', 'myTeams']),
    bcpir: z.enum(['include', 'only', 'exclude']),
    q: z.string(),
    layout: z.enum(['cards', 'table']),
  })
  .refine(({ yearFrom, yearTo }) =>
    yearFrom === null || yearTo === null ? yearFrom === yearTo : yearFrom <= yearTo,
  );

const KEYS = [
  'status',
  'yearFrom',
  'yearTo',
  'year',
  'type',
  'scope',
  'bcpir',
  'q',
  'layout',
  'other',
];
const WORDS = [
  'draft',
  'published',
  'myTeams',
  'table',
  'strategic',
  'crossMarket',
  '2026',
  '999',
  'only',
];

/** Any address: the list's parameters with any text or one of their words. */
const anyAddress = fc
  .array(fc.tuple(fc.constantFrom(...KEYS), fc.oneof(fc.string(), fc.constantFrom(...WORDS))))
  .map((pairs) => new URLSearchParams(pairs));

const year = fc.integer({ min: 1000, max: 9999 });

/** Any filter set the list can show (a search that is not only spaces). */
const anyFilters: fc.Arbitrary<PlanListFilters> = fc
  .record({
    status: fc.constantFrom('all', 'draft', 'published'),
    years: fc.option(fc.tuple(year, year)),
    types: fc.subarray(PLAN_TYPES),
    scope: fc.constantFrom('all', 'myTeams'),
    bcpir: fc.constantFrom('include', 'only', 'exclude'),
    q: fc.string().filter((q) => q === '' || q.trim() !== ''),
    layout: fc.constantFrom('cards', 'table'),
  })
  .map(({ years, ...rest }) => ({
    ...rest,
    yearFrom: years ? Math.min(...years) : null,
    yearTo: years ? Math.max(...years) : null,
  }));

afterEach(() => {
  window.localStorage.clear();
  vi.restoreAllMocks();
});

describe('listPreferences (FR-003)', () => {
  it('never fails on any address or stored text, and reads a valid filter set', () => {
    fc.assert(
      fc.property(
        anyAddress,
        fc.oneof(
          fc.string(),
          fc.jsonValue().map((value) => JSON.stringify(value)),
        ),
        (search, text) => {
          window.localStorage.setItem('mapp.plans.list.U-1002', text);
          expect(VALID.safeParse(readFilters(search, 'U-1002')).success).toBe(true);
          expect(VALID.safeParse(readRemembered('U-1002')).success).toBe(true);
        },
      ),
    );
  });

  it('reads back every filter set it writes, to the address and to the storage', () => {
    fc.assert(
      fc.property(anyFilters, (filters) => {
        window.localStorage.clear();
        expect(readFilters(toSearchParams(filters), 'U-1002')).toEqual(filters);
        remember('U-1002', filters);
        expect(readRemembered('U-1002')).toEqual({ ...filters, q: '' });
      }),
    );
  });

  it('uses the defaults when nothing is remembered', () => {
    expect(readFilters(new URLSearchParams(), 'U-1002')).toEqual(DEFAULT_FILTERS);
  });

  it('remembers filters and layout per person, never the search', () => {
    remember('U-1002', {
      status: 'draft',
      yearFrom: 2025,
      yearTo: 2026,
      types: ['strategic'],
      scope: 'myTeams',
      bcpir: 'only',
      q: 'china',
      layout: 'table',
    });

    expect(readFilters(new URLSearchParams(), 'U-1002')).toEqual({
      status: 'draft',
      yearFrom: 2025,
      yearTo: 2026,
      types: ['strategic'],
      scope: 'myTeams',
      bcpir: 'only',
      q: '',
      layout: 'table',
    });
    expect(readFilters(new URLSearchParams(), 'U-1004')).toEqual(DEFAULT_FILTERS);
  });

  it('lets the address win over what is remembered', () => {
    remember('U-1002', { ...DEFAULT_FILTERS, status: 'draft', layout: 'table' });

    expect(readFilters(new URLSearchParams('year=2025&q=p-1003'), 'U-1002')).toEqual({
      ...DEFAULT_FILTERS,
      yearFrom: 2025,
      yearTo: 2025,
      q: 'p-1003',
    });
  });

  it('reads a single year, remembered before 016, as a one-year range (D-269)', () => {
    window.localStorage.setItem(
      'mapp.plans.list.U-1002',
      JSON.stringify({ status: 'all', year: 2026, scope: 'all', layout: 'cards' }),
    );

    expect(readRemembered('U-1002')).toEqual({ ...DEFAULT_FILTERS, yearFrom: 2026, yearTo: 2026 });
  });

  it('keeps a range in order and drops unknown plan types', () => {
    expect(
      readFilters(
        new URLSearchParams('yearFrom=2027&yearTo=2025&type=strategic&type=global'),
        'U-1002',
      ),
    ).toEqual({ ...DEFAULT_FILTERS, yearFrom: 2025, yearTo: 2027, types: ['strategic'] });
    expect(readFilters(new URLSearchParams('yearTo=2026'), 'U-1002')).toEqual({
      ...DEFAULT_FILTERS,
      yearFrom: 2026,
      yearTo: 2026,
    });
  });

  it('ignores invalid values in the address and in storage', () => {
    expect(
      readFilters(new URLSearchParams('status=open&year=next&scope=mine&layout=grid'), 'U-1002'),
    ).toEqual(DEFAULT_FILTERS);
    window.localStorage.setItem('mapp.plans.list.U-1002', '{not json');
    expect(readRemembered('U-1002')).toEqual(DEFAULT_FILTERS);
  });

  it('keeps working when the storage is blocked', () => {
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new Error('blocked');
    });
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new Error('blocked');
    });

    expect(() => remember('U-1002', DEFAULT_FILTERS)).not.toThrow();
    expect(readRemembered('U-1002')).toEqual(DEFAULT_FILTERS);
  });

  it('puts only the values that differ from the defaults in the address', () => {
    expect(toSearchParams(DEFAULT_FILTERS).toString()).toBe('');
    expect(
      toSearchParams({
        status: 'published',
        yearFrom: 2025,
        yearTo: 2026,
        types: ['operational', 'crossMarket'],
        scope: 'myTeams',
        bcpir: 'exclude',
        q: 'x',
        layout: 'table',
      }).toString(),
    ).toBe(
      'status=published&yearFrom=2025&yearTo=2026&type=operational&type=crossMarket&scope=myTeams&bcpir=exclude&q=x&layout=table',
    );
  });
});
