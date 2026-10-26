import type {
  AllWorkOptions,
  AllWorkPreferences,
  AllWorkResult,
  SavedView,
  ViewState,
} from '@/features/all-work';
import { standardState } from '@/features/all-work';
import { reply } from '../replies';
import {
  CLUSTER_COLUMN,
  DEFAULT_COLUMNS,
  STANDARD_COLUMNS,
  column,
  columnsOf,
  usd,
} from './all-work-rows';
import { seedRows } from './all-work-seed';

/** GET /api/v1/all-work for the Grid on the seed story; `overrides` replace whole parts. */
export function gridResult(overrides: Partial<AllWorkResult> = {}): AllWorkResult {
  const columns = DEFAULT_COLUMNS;
  return {
    year: 2026,
    today: '2026-10-18',
    view: 'grid',
    displayCurrency: 'USD',
    generatedAt: '2026-10-18T14:05:00Z',
    columns: columnsOf(columns),
    periods: [],
    rows: seedRows(columns),
    summary: {
      campaigns: 3,
      initiatives: 2,
      activities: 3,
      plannedCost: usd('17000.00'),
      planBudget: usd('2000000.00'),
      unassignedPlannedCost: usd('3000.00'),
      negativeBalanceCount: 1,
    },
    count: { shown: 3, total: 3, noun: 'activities' },
    totals: {
      activities: 3,
      cells: [
        null,
        null,
        null,
        null,
        null,
        { kind: 'money', money: usd('2000000.00') },
        { kind: 'money', money: usd('17000.00') },
        { kind: 'money', money: usd('15000.00') },
        { kind: 'money', money: usd('3000.00') },
      ],
      plannedCost: usd('17000.00'),
      planBudget: usd('2000000.00'),
    },
    leftOut: [],
    ...overrides,
  };
}

/** GET /api/v1/all-work answering nothing for the filters. */
export function emptyResult(overrides: Partial<AllWorkResult> = {}): AllWorkResult {
  return gridResult({
    rows: [],
    summary: {
      campaigns: 0,
      initiatives: 0,
      activities: 0,
      plannedCost: null,
      planBudget: null,
      unassignedPlannedCost: null,
      negativeBalanceCount: 0,
    },
    count: { shown: 0, total: 3, noun: 'activities' },
    totals: {
      activities: 0,
      cells: DEFAULT_COLUMNS.map(() => null),
      plannedCost: null,
      planBudget: null,
    },
    ...overrides,
  });
}

/** GET /api/v1/all-work/options for 2026. */
export function allWorkOptions(overrides: Partial<AllWorkOptions> = {}): AllWorkOptions {
  return {
    year: 2026,
    currentYear: 2026,
    years: [2027, 2026, 2025],
    plans: [
      { id: 'P-1001', label: 'FY2026 Lubricants Europe', status: 'published', strategic: false },
      { id: 'P-1002', label: 'FY2026 Mobility China', status: 'draft', strategic: true },
    ],
    owners: [
      { id: 'U-1003', label: 'Daniel Okoro', inactive: false },
      { id: 'U-1004', label: 'Mei Lin', inactive: false },
      { id: 'U-1002', label: 'Priya Raman', inactive: false },
      { id: 'U-1009', label: 'Tom Weber', inactive: true },
    ],
    teams: [
      { id: 'T-1001', label: 'Lubricants Europe', inactive: false, mine: true },
      { id: 'T-1003', label: 'Global Marketing Ops', inactive: false, mine: false },
      { id: 'T-1002', label: 'Mobility China', inactive: false, mine: false },
    ],
    businessUnits: [
      { id: 'SO-1001', label: 'Lubricants', inactive: false },
      { id: 'SO-1002', label: 'Mobility', inactive: false },
    ],
    activityTypes: [
      { id: 'AT-1001', label: 'Event', inactive: false },
      { id: 'AT-1002', label: 'Digital media', inactive: false },
      { id: 'AT-1003', label: 'Sponsorship', inactive: true },
    ],
    columnGroups: [
      { group: 'standard', label: 'Standard', columns: STANDARD_COLUMNS },
      {
        group: 'plan',
        label: 'Plan',
        columns: [column('plan.description', 'Description', 'text', ['plan'], 'plan')],
      },
      { group: 'campaign', label: 'Campaign', columns: [CLUSTER_COLUMN] },
      {
        group: 'initiative',
        label: 'Initiative',
        columns: [
          column('initiative.description', 'Description', 'text', ['initiative'], 'initiative'),
        ],
      },
      {
        group: 'activity',
        label: 'Activity',
        columns: [column('activity.type', 'Activity type', 'text', ['activity'], 'activity')],
      },
      {
        group: 'lineItem',
        label: 'Line item',
        columns: [
          column(
            'lineItem.localPlannedCost',
            'Planned cost in local currency',
            'money',
            ['lineItem'],
            'lineItem',
          ),
        ],
      },
    ],
    defaultColumns: DEFAULT_COLUMNS,
    ...overrides,
  };
}

/** GET /api/v1/all-work/preferences of someone who never changed the columns. */
export function defaultPreferences(): AllWorkPreferences {
  return { columns: [...DEFAULT_COLUMNS], sort: null };
}

/** The standard state of 2026 (FR-009). */
export function standard2026(): ViewState {
  return standardState(defaultPreferences(), 2026);
}

/** A saved view of Priya's. */
export function savedView(overrides: Partial<SavedView> = {}): SavedView {
  return {
    id: 'SV-1001',
    name: 'Declined activities',
    isDefault: false,
    state: {
      ...standard2026(),
      filters: { ...standard2026().filters, activityStatuses: ['declined'] },
    },
    createdAt: '2026-10-01T09:00:00Z',
    updatedAt: '2026-10-01T09:00:00Z',
    ...overrides,
  };
}

/** The first-visit answers of the page (no saved views, the default preferences). */
export const PAGE_ANSWERS = [
  reply('GET /api/v1/all-work/preferences', { body: defaultPreferences() }),
  reply('GET /api/v1/all-work/options', { body: allWorkOptions() }),
  reply('GET /api/v1/saved-views', { body: { items: [], total: 0 } }),
];
