import { describe, expect, it } from 'vitest';
import {
  allWorkOptions,
  defaultPreferences,
  seedRows,
  standard2026,
} from '@/testing/fixtures/all-work';
import type { AllWorkRow, ViewState } from './all-work-types';
import {
  chipsOf,
  clearAll,
  continuedPath,
  dropColumn,
  fromAddress,
  hasAddressFilters,
  hasFilters,
  leftOutOf,
  moveColumn,
  nextSort,
  pageOf,
  removeChip,
  sameState,
  standardState,
  toQueryParams,
  toggleRow,
  visibleRows,
} from './view-state';

function keysOf(state: ViewState, rows: AllWorkRow[] = seedRows()) {
  return visibleRows(rows, state).map((visible) => visible.row.key);
}

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

describe('fromAddress (D-122)', () => {
  it('applies the API filter names of the address over the standard view', () => {
    const params = new URLSearchParams(
      'year=2027&owner=U-1003&activityStatus=declined&activityStatus=nonsense&strategic=only&search=helix',
    );

    expect(hasAddressFilters(params)).toBe(true);
    const state = fromAddress(params, standard2026());
    expect(state.year).toBe(2027);
    expect(state.filters.owners).toEqual([{ id: 'U-1003', label: 'U-1003' }]);
    expect(state.filters.activityStatuses).toEqual(['declined']);
    expect(state.filters.strategic).toBe('only');
    expect(state.search).toBe('helix');
    expect(state.columns).toEqual(standard2026().columns);
  });

  it('ignores a year outside 2000–2100 and addresses without filters', () => {
    expect(fromAddress(new URLSearchParams('year=1999'), standard2026()).year).toBe(2026);
    expect(hasAddressFilters(new URLSearchParams('tab=x'))).toBe(false);
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

describe('visibleRows (FR-011, FR-012, FR-006)', () => {
  it('opens plans, campaigns and initiatives and closes activities by default', () => {
    expect(keysOf(standard2026())).toEqual([
      'P-1001',
      'C-1001',
      'I-1001',
      'A-1001',
      'A-1002',
      'C-1002',
      'I-1002',
      'A-1003',
      'P-1002',
      'C-1003',
    ]);
    const [plan] = visibleRows(seedRows(), standard2026());
    expect(plan).toMatchObject({ expanded: true, childCount: 2 });
  });

  it('opens or closes one row against the mode', () => {
    const opened = toggleRow(standard2026(), 'A-1001');
    expect(keysOf(opened)).toContain('L-1001');
    expect(keysOf(opened)).toContain('A-1001:unassigned');

    const closedAgain = toggleRow(opened, 'A-1001');
    expect(keysOf(closedAgain)).not.toContain('L-1001');

    const planClosed = toggleRow(standard2026(), 'P-1001');
    expect(keysOf(planClosed)).toEqual(['P-1001', 'P-1002', 'C-1003']);
  });

  it('opens everything with Expand all and nothing with Collapse all', () => {
    const all = { ...standard2026(), expansion: { mode: 'allExpanded' as const, toggled: [] } };
    expect(keysOf(all)).toHaveLength(seedRows().length);

    const none = { ...standard2026(), expansion: { mode: 'allCollapsed' as const, toggled: [] } };
    expect(keysOf(none)).toEqual(['P-1001', 'P-1002']);
  });

  it('hides the unassigned rows while the switch is off, and counts them only when shown (FR-012)', () => {
    const all = { ...standard2026(), expansion: { mode: 'allExpanded' as const, toggled: [] } };
    expect(keysOf({ ...all, showUnassigned: false })).not.toContain('A-1001:unassigned');
    const activity = visibleRows(seedRows(), { ...all, showUnassigned: false }).find(
      (visible) => visible.row.key === 'A-1001',
    );
    expect(activity?.childCount).toBe(1);
  });

  it('opens the path to a search hit and keeps everything else closed (FR-006)', () => {
    const rows = seedRows().map((row) =>
      row.key === 'L-1002' ? { ...row, searchHit: { fields: ['name' as const] } } : row,
    );
    const state = { ...standard2026(), search: 'paid' };

    expect(keysOf(state, rows)).toEqual([
      'P-1001',
      'C-1001',
      'I-1001',
      'A-1001',
      'A-1002',
      'L-1002',
      'A-1002:unassigned',
      'C-1002',
      'P-1002',
    ]);
  });
});

describe('pageOf and continuedPath (D-115)', () => {
  const all = { ...standard2026(), expansion: { mode: 'allExpanded' as const, toggled: [] } };
  const rows = visibleRows(seedRows(), all);

  it('cuts the rows into pages and keeps the page within the last one', () => {
    expect(pageOf(rows, 1, 5)).toMatchObject({ page: 1, pageCount: 3, from: 1, to: 5, total: 14 });
    expect(pageOf(rows, 9, 5)).toMatchObject({ page: 3, from: 11, to: 14 });
    expect(pageOf([], 1, 50)).toMatchObject({ page: 1, pageCount: 1, from: 0, to: 0, total: 0 });
  });

  it('names the rows above a page that starts inside a branch', () => {
    const second = pageOf(rows, 2, 5).rows;
    expect(second[0].row.key).toBe('A-1001:unassigned');
    expect(continuedPath(second, seedRows())).toBe(
      'FY2026 Lubricants Europe › Helix launch › Helix digital › Helix launch event',
    );
    expect(continuedPath(pageOf(rows, 1, 5).rows, seedRows())).toBe('');
  });
});

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
    const [declined, daniel, , , strategic, showLevels] = chipsOf(state, allWorkOptions());
    expect(removeChip(state, declined.key).filters.activityStatuses).toEqual([]);
    expect(removeChip(state, daniel.key).filters.owners).toEqual([
      { id: 'U-1009', label: 'Tom Weber' },
    ]);
    expect(removeChip(state, strategic.key).filters.strategic).toBe('include');
    expect(removeChip(state, showLevels.key).filters.showLevels).toBe('everything');
  });

  it('clears every filter and the search but keeps the year', () => {
    const cleared = clearAll({ ...state, year: 2027 });
    expect(cleared.year).toBe(2027);
    expect(hasFilters(cleared)).toBe(false);
    expect(hasFilters(state)).toBe(true);
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
});

describe('sorting and column order (FR-015, FR-021)', () => {
  it('sorts ascending, then descending, then in the plan-tree order again', () => {
    const first = nextSort(null, 'plannedCost');
    expect(first).toEqual({ column: 'plannedCost', direction: 'asc' });
    const second = nextSort(first, 'plannedCost');
    expect(second).toEqual({ column: 'plannedCost', direction: 'desc' });
    expect(nextSort(second, 'plannedCost')).toBeNull();
    expect(nextSort(second, 'owner')).toEqual({ column: 'owner', direction: 'asc' });
  });

  it('moves a column one place up or down, never past the ends', () => {
    expect(moveColumn(['id', 'status', 'owner'], 'owner', -1)).toEqual(['id', 'owner', 'status']);
    expect(moveColumn(['id', 'status', 'owner'], 'id', 1)).toEqual(['status', 'id', 'owner']);
    expect(moveColumn(['id', 'status'], 'id', -1)).toEqual(['id', 'status']);
    expect(moveColumn(['id', 'status'], 'status', 1)).toEqual(['id', 'status']);
  });

  it('puts a dragged column in the place of the one it is dropped on', () => {
    const columns = ['id', 'status', 'owner', 'plannedCost'];
    expect(dropColumn(columns, 'plannedCost', 'status')).toEqual([
      'id',
      'plannedCost',
      'status',
      'owner',
    ]);
    expect(dropColumn(columns, 'id', 'owner')).toEqual(['status', 'owner', 'id', 'plannedCost']);
    expect(dropColumn(columns, 'id', 'id')).toEqual(columns);
  });
});
