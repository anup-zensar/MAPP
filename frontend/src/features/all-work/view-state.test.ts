import fc from 'fast-check';
import { describe, expect, it } from 'vitest';
import {
  DEFAULT_COLUMNS,
  allWorkOptions,
  defaultPreferences,
  standard2026,
} from '@/testing/fixtures/all-work';
import type { ViewState } from './all-work-types';
import { leftOutOf, sameState, standardState, toQueryParams } from './view-state';

describe('standardState (FR-009, FR-019)', () => {
  it('is the Grid of the year without filters, with the remembered columns and sort', () => {
    const state = standardState(
      { columns: ['id', 'plannedCost'], sort: { column: 'plannedCost', direction: 'desc' } },
      2026,
    );

    expect(state).toEqual({
      view: 'grid',
      year: 2026,
      filters: {
        plans: [],
        planStatuses: [],
        containerStatuses: [],
        activityStatuses: [],
        owners: [],
        teams: [],
        businessUnits: [],
        activityTypes: [],
        strategic: 'include',
        showLevels: 'everything',
      },
      search: '',
      columns: ['id', 'plannedCost'],
      sort: { column: 'plannedCost', direction: 'desc' },
      expansion: { mode: 'default', toggled: [] },
      showUnassigned: true,
      period: 'month',
      calendarLayout: 'timeline',
    });
  });
});

describe('toQueryParams (FR-004, FR-018)', () => {
  it('sends only the year, the view and the columns in order for the standard Grid', () => {
    expect(toQueryParams(standard2026())).toEqual({
      year: 2026,
      view: 'grid',
      column: defaultPreferences().columns,
    });
  });

  it('sends every filter in use, repeated values as lists, the trimmed search and the sort', () => {
    const state: ViewState = {
      ...standard2026(),
      columns: ['plannedCost', 'id'],
      search: '  helix ',
      sort: { column: 'plannedCost', direction: 'desc' },
      filters: {
        plans: [{ id: 'P-1001', label: 'FY2026 Lubricants Europe' }],
        planStatuses: ['published'],
        containerStatuses: ['inProgress', 'running'],
        activityStatuses: ['declined'],
        owners: [
          { id: 'U-1003', label: 'Daniel Okoro' },
          { id: 'U-1002', label: 'Priya Raman' },
        ],
        teams: [{ id: 'T-1001', label: 'Lubricants Europe' }],
        businessUnits: [{ id: 'SO-1001', label: 'Lubricants' }],
        activityTypes: [{ id: 'AT-1001', label: 'Event' }],
        strategic: 'only',
        showLevels: 'completeBranches',
      },
    };

    expect(toQueryParams(state)).toEqual({
      year: 2026,
      view: 'grid',
      plan: ['P-1001'],
      planStatus: ['published'],
      containerStatus: ['inProgress', 'running'],
      activityStatus: ['declined'],
      owner: ['U-1003', 'U-1002'],
      team: ['T-1001'],
      businessUnit: ['SO-1001'],
      activityType: ['AT-1001'],
      strategic: 'only',
      showLevels: 'completeBranches',
      search: 'helix',
      column: ['plannedCost', 'id'],
      sort: 'plannedCost:desc',
    });
  });

  it('sends the period with Budget & cost and no columns with the other views (FR-022)', () => {
    expect(toQueryParams({ ...standard2026(), view: 'budget', period: 'quarter' })).toEqual({
      year: 2026,
      view: 'budget',
      period: 'quarter',
    });
    expect(toQueryParams({ ...standard2026(), view: 'calendar' })).toEqual({
      year: 2026,
      view: 'calendar',
    });
  });
});

describe('sameState (FR-030)', () => {
  it('ignores the order of chosen values, opened rows and stored names, not the column order', () => {
    const a: ViewState = {
      ...standard2026(),
      filters: {
        ...standard2026().filters,
        owners: [
          { id: 'U-1', label: 'One' },
          { id: 'U-2', label: 'Two' },
        ],
        activityStatuses: ['declined', 'approved'],
      },
      expansion: { mode: 'default', toggled: ['A-1', 'C-1'] },
    };
    const b: ViewState = {
      ...a,
      filters: {
        ...a.filters,
        owners: [
          { id: 'U-2', label: 'Two (renamed)' },
          { id: 'U-1', label: 'One' },
        ],
        activityStatuses: ['approved', 'declined'],
      },
      expansion: { mode: 'default', toggled: ['C-1', 'A-1'] },
    };

    expect(sameState(a, b)).toBe(true);
    expect(sameState(a, { ...a, columns: [...a.columns].reverse() })).toBe(false);
    expect(sameState(a, { ...a, view: 'budget' })).toBe(false);
  });
});

describe('leftOutOf (FR-031)', () => {
  it('drops and names the values and columns that no longer exist', () => {
    const state: ViewState = {
      ...standard2026(),
      columns: ['id', 'attr.SA-0999', 'plannedCost'],
      sort: { column: 'attr.SA-0999', direction: 'asc' },
      filters: {
        ...standard2026().filters,
        plans: [{ id: 'P-0999', label: 'FY2026 Old plan' }],
        owners: [
          { id: 'U-1003', label: 'Daniel Okoro' },
          { id: 'U-0999', label: 'Jane Doe' },
        ],
      },
    };

    const { state: cleaned, names } = leftOutOf(state, allWorkOptions());

    expect(names).toEqual([
      'Plan: FY2026 Old plan',
      'Owner: Jane Doe',
      '1 column',
      'the sort order',
    ]);
    expect(cleaned.filters.plans).toEqual([]);
    expect(cleaned.filters.owners).toEqual([{ id: 'U-1003', label: 'Daniel Okoro' }]);
    expect(cleaned.columns).toEqual(['id', 'plannedCost']);
    expect(cleaned.sort).toBeNull();
  });

  it('leaves a view that still fits alone', () => {
    expect(leftOutOf(standard2026(), allWorkOptions()).names).toEqual([]);
  });

  it('fits any saved view once: a second fit leaves nothing out (FR-007)', () => {
    const owners = [...allWorkOptions().owners.map((owner) => owner.id), 'U-0999'];
    const columns = [...DEFAULT_COLUMNS, 'attr.SA-0999'];
    fc.assert(
      fc.property(fc.subarray(owners), fc.shuffledSubarray(columns), (ownerIds, chosen) => {
        const saved: ViewState = {
          ...standard2026(),
          columns: chosen,
          filters: { ...standard2026().filters, owners: ownerIds.map((id) => ({ id, label: id })) },
        };
        const { state } = leftOutOf(saved, allWorkOptions());

        expect(leftOutOf(state, allWorkOptions())).toEqual({ state, names: [] });
      }),
    );
  });
});
