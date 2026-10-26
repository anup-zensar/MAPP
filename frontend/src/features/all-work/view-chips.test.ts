import { describe, expect, it } from 'vitest';
import { nth } from '@/testing/dom';
import { allWorkOptions, standard2026 } from '@/testing/fixtures/all-work';
import type { ViewState } from './all-work-types';
import { chipsOf, clearAll, hasFilters, removeChip } from './view-chips';

describe('chips and Clear all (FR-008)', () => {
  const state: ViewState = {
    ...standard2026(),
    search: 'helix',
    filters: {
      ...standard2026().filters,
      activityStatuses: ['declined'],
      owners: [
        { id: 'U-1003', label: 'Daniel' },
        { id: 'U-1009', label: 'Tom Weber' },
      ],
      teams: [{ id: 'T-0001', label: 'Old team' }],
      strategic: 'exclude',
      showLevels: 'initiativesWithoutActivities',
    },
  };

  it('shows one chip per value with the filter name, never the year', () => {
    expect(chipsOf(state, allWorkOptions()).map((chip) => chip.label)).toEqual([
      'Activity status: Declined',
      'Owner: Daniel Okoro',
      'Owner: Tom Weber (inactive)',
      'Team: Old team',
      'Strategic / AOP plans: Exclude',
      'Show levels: Initiatives without activities',
    ]);
    expect(chipsOf(standard2026(), allWorkOptions())).toEqual([]);
  });

  it('removes one value per chip', () => {
    const chips = chipsOf(state, allWorkOptions());
    const without = (at: number) => removeChip(state, nth(chips, at).key).filters;
    expect(without(0).activityStatuses).toEqual([]);
    expect(without(1).owners).toEqual([{ id: 'U-1009', label: 'Tom Weber' }]);
    expect(without(4).strategic).toBe('include');
    expect(without(5).showLevels).toBe('everything');
  });

  it('clears every filter and the search but keeps the year', () => {
    const cleared = clearAll({ ...state, year: 2027 });
    expect(cleared.year).toBe(2027);
    expect(hasFilters(cleared)).toBe(false);
    expect(hasFilters(state)).toBe(true);
  });
});
