import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  DEFAULT_FILTERS,
  readFilters,
  readRemembered,
  remember,
  toSearchParams,
} from './list-preferences';

afterEach(() => {
  window.localStorage.clear();
  vi.restoreAllMocks();
});

describe('listPreferences (FR-003)', () => {
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
      q: 'china',
      layout: 'table',
    });

    expect(readFilters(new URLSearchParams(), 'U-1002')).toEqual({
      status: 'draft',
      yearFrom: 2025,
      yearTo: 2026,
      types: ['strategic'],
      scope: 'myTeams',
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
        q: 'x',
        layout: 'table',
      }).toString(),
    ).toBe(
      'status=published&yearFrom=2025&yearTo=2026&type=operational&type=crossMarket&scope=myTeams&q=x&layout=table',
    );
  });
});
