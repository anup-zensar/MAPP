import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  DEFAULT_FILTERS,
  readFilters,
  readRemembered,
  remember,
  toSearchParams,
} from './listPreferences';

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
      year: 2026,
      scope: 'myTeams',
      q: 'china',
      layout: 'table',
    });

    expect(readFilters(new URLSearchParams(), 'U-1002')).toEqual({
      status: 'draft',
      year: 2026,
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
      year: 2025,
      q: 'p-1003',
    });
  });

  it('ignores invalid values in the address and in storage', () => {
    expect(
      readFilters(new URLSearchParams('status=open&year=next&scope=mine&layout=grid'), 'U-1002'),
    ).toEqual(DEFAULT_FILTERS);
    window.localStorage.setItem('northstar.plans.list.U-1002', '{not json');
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
        year: 2026,
        scope: 'myTeams',
        q: 'x',
        layout: 'table',
      }).toString(),
    ).toBe('status=published&year=2026&scope=myTeams&q=x&layout=table');
  });
});
