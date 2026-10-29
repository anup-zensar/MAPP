import fc from 'fast-check';
import { describe, expect, it } from 'vitest';
import { dashboardHref, DEFAULT_FILTERS, toQuery, writeFilters } from './dashboard-links';
import { readFilters } from './schemas';

describe('the dashboard address (036 FR-007)', () => {
  it('reads any address into filters the API accepts', () => {
    fc.assert(
      fc.property(
        fc.dictionary(fc.constantFrom('year', 'planType', 'currency', 'team', 'plan'), fc.string()),
        (raw) => {
          const filters = readFilters(new URLSearchParams(raw));
          expect(filters.year === null || (filters.year >= 2000 && filters.year <= 2100)).toBe(
            true,
          );
          expect(['', 'operational', 'strategic', 'crossMarket']).toContain(filters.planType);
          expect(filters.currency === '' || /^[A-Z]{3}$/.test(filters.currency)).toBe(true);
          expect(filters.team.length).toBeLessThanOrEqual(40);
        },
      ),
    );
  });

  it('writes and reads the same filters back', () => {
    const filters = {
      year: 2027,
      planType: 'strategic' as const,
      currency: 'EUR',
      team: 'T-1002',
      plan: 'P-1003',
    };
    expect(readFilters(writeFilters(filters))).toEqual(filters);
    expect(readFilters(new URLSearchParams())).toEqual(DEFAULT_FILTERS);
  });

  it('asks the Executive dashboard without a market and the Regional one with it (FR-022)', () => {
    const filters = { ...DEFAULT_FILTERS, team: 'T-1001', plan: 'P-1001' };
    expect(toQuery('executive', filters)).toEqual({});
    expect(toQuery('regional', filters)).toEqual({ teamId: 'T-1001', planId: 'P-1001' });
    expect(dashboardHref('regional', { ...DEFAULT_FILTERS, year: 2026 })).toBe(
      '/dashboards/regional?year=2026',
    );
  });
});
