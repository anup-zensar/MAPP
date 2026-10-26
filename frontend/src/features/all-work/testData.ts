/**
 * Contract-shaped answers of the All work API for the feature's tests, after the seed story
 * (docs/product/demo-users-and-data.md): FY2026 Lubricants Europe with Helix launch (one activity
 * with a positive and one with a negative unassigned balance) and Rimula fleet (an activity
 * without dates), and FY2026 Mobility China without activities.
 */
import type {
  AllWorkCell,
  AllWorkColumn,
  AllWorkOptions,
  AllWorkPreferences,
  AllWorkResult,
  AllWorkRow,
  AllWorkStatus,
  Money,
  SavedView,
  ViewState,
} from './allWorkTypes';
import { standardState } from './viewState';

export function usd(amount: string): Money {
  return { amount, currency: 'USD' };
}

type Level = AllWorkRow['level'];
const ALL: Level[] = ['plan', 'campaign', 'initiative', 'activity', 'lineItem'];
const CONTAINERS: Level[] = ['plan', 'campaign', 'initiative'];

function column(
  key: string,
  label: string,
  kind: AllWorkColumn['kind'],
  levels: Level[],
  group: AllWorkColumn['group'] = 'standard',
): AllWorkColumn {
  return { key, label, group, kind, levels, sortable: true, inactive: false };
}

/** The Standard group of data-model §6.1. */
export const STANDARD_COLUMNS: AllWorkColumn[] = [
  column('id', 'ID', 'text', ALL),
  column('level', 'Level', 'text', ALL),
  column('owner', 'Owner', 'text', ALL),
  column('startDate', 'Start date', 'date', [...CONTAINERS, 'activity']),
  column('endDate', 'End date', 'date', [...CONTAINERS, 'activity']),
  column('status', 'Status', 'status', ALL),
  column('timing', 'Timing', 'status', ['campaign', 'initiative']),
  column('completeness', 'Completeness', 'status', ALL),
  column('planBudget', 'Plan budget', 'money', CONTAINERS),
  column('remainingBudget', 'Remaining budget', 'money', ['plan', 'campaign']),
  column('budgetUsed', 'Budget used', 'percent', CONTAINERS),
  column('plannedCost', 'Planned cost', 'money', [...ALL, 'unassigned']),
  column('lineItems', 'Line items', 'money', [...CONTAINERS, 'activity']),
  column('unassigned', 'Unassigned planned cost', 'money', [
    ...CONTAINERS,
    'activity',
    'unassigned',
  ]),
  column('activities', 'Activities', 'number', CONTAINERS),
  column('inq', 'INQ', 'number', [...CONTAINERS, 'activity']),
  column('mql', 'MQL', 'number', [...CONTAINERS, 'activity']),
  column('revenue', 'Revenue', 'money', [...CONTAINERS, 'activity']),
  column('createdAt', 'Created', 'dateTime', ALL),
  column('updatedAt', 'Last changed', 'dateTime', ALL),
];

export const CLUSTER_COLUMN = column('attr.SA-1001', 'Cluster', 'text', ['campaign'], 'campaign');

/** The default columns (FR-019). */
export const DEFAULT_COLUMNS = [
  'id',
  'status',
  'owner',
  'startDate',
  'endDate',
  'planBudget',
  'plannedCost',
  'lineItems',
  'unassigned',
];

export function columnsOf(keys: string[]): AllWorkColumn[] {
  const all = [...STANDARD_COLUMNS, CLUSTER_COLUMN];
  return keys.map((key) => {
    const found = all.find((candidate) => candidate.key === key);
    if (!found) throw new Error(`No test column ${key}`);
    return found;
  });
}

const STATUS: Record<string, AllWorkStatus> = {
  published: { value: 'published', label: 'Published', tone: 'positive' },
  draft: { value: 'draft', label: 'Draft', tone: 'neutral' },
  inProgress: {
    value: 'inProgress',
    label: 'In progress',
    tone: 'review',
    detail: '1 of 3 activities approved',
  },
  noActivities: { value: 'noActivities', label: 'No activities', tone: 'neutral' },
  approved: { value: 'approved', label: 'Approved', tone: 'positive' },
  declined: { value: 'declined', label: 'Declined', tone: 'problem' },
  created: { value: 'created', label: 'Created', tone: 'neutral' },
  filledIn: { value: 'allFilledIn', label: 'All filled in', tone: 'positive' },
};

/** What a test row is made of; the cells of the requested columns are built from it. */
interface RowSpec {
  key: string;
  level: Level;
  name: string;
  depth: number;
  parentKey: string | null;
  planId: string;
  status?: keyof typeof STATUS;
  owner?: string;
  startDate?: string;
  endDate?: string;
  ongoing?: boolean;
  budget?: string;
  plannedCost?: string;
  lineItems?: string;
  unassigned?: string;
  activityId?: string;
  strategic?: boolean;
  hasChildren?: boolean;
}

const money = (amount: string | undefined): AllWorkCell =>
  amount === undefined
    ? { kind: 'empty' }
    : { kind: 'money', money: usd(amount), ...(amount.startsWith('-') && { tone: 'problem' }) };

/** The cell of one column for a row (blank on levels the column does not belong to). */
function cellOf(spec: RowSpec, key: string): AllWorkCell | null {
  const col = [...STANDARD_COLUMNS, CLUSTER_COLUMN].find((candidate) => candidate.key === key);
  if (!col || !col.levels.includes(spec.level)) {
    return null;
  }
  switch (key) {
    case 'id':
      return { kind: 'text', text: spec.key };
    case 'status': {
      const status = spec.status ? STATUS[spec.status] : null;
      return status ? { kind: 'status', text: status.label, tone: status.tone } : { kind: 'empty' };
    }
    case 'owner':
      return spec.owner ? { kind: 'text', text: spec.owner } : { kind: 'empty' };
    case 'startDate':
      return spec.startDate ? { kind: 'date', date: spec.startDate } : { kind: 'empty' };
    case 'endDate':
      if (spec.ongoing) return { kind: 'text', text: 'Ongoing' };
      return spec.endDate ? { kind: 'date', date: spec.endDate } : { kind: 'empty' };
    case 'planBudget':
      return money(spec.budget);
    case 'plannedCost':
      return money(spec.plannedCost);
    case 'lineItems':
      return money(spec.lineItems);
    case 'unassigned':
      return money(spec.unassigned);
    default:
      return { kind: 'empty' };
  }
}

/** A contract-shaped row with the cells of `columns`. */
export function row(spec: RowSpec, columns: string[] = DEFAULT_COLUMNS): AllWorkRow {
  const status = spec.status ? STATUS[spec.status] : null;
  return {
    key: spec.key,
    parentKey: spec.parentKey,
    level: spec.level,
    recordId: spec.level === 'unassigned' ? (spec.activityId ?? spec.key) : spec.key,
    planId: spec.planId,
    ...(spec.activityId && { activityId: spec.activityId }),
    depth: spec.depth,
    name: spec.name,
    role: 'match',
    status: spec.level === 'unassigned' ? null : status,
    ...(spec.strategic !== undefined && { strategic: spec.strategic }),
    ...(spec.startDate && { startDate: spec.startDate }),
    ...(spec.endDate && { endDate: spec.endDate }),
    ...(spec.ongoing && { ongoing: true }),
    ...(spec.owner && { owner: { id: ownerId(spec.owner), name: spec.owner, inactive: false } }),
    ...(spec.plannedCost && { plannedCost: usd(spec.plannedCost) }),
    ...(spec.budget && { budget: usd(spec.budget) }),
    ...(spec.level === 'activity' &&
      spec.unassigned !== undefined && { balance: usd(spec.unassigned) }),
    hasChildren: spec.hasChildren ?? false,
    cells: columns.map((key) => cellOf(spec, key)),
  };
}

function ownerId(name: string): string {
  return (
    { 'Priya Raman': 'U-1002', 'Daniel Okoro': 'U-1003', 'Mei Lin': 'U-1004' }[name] ?? 'U-9999'
  );
}

/** The seed story's rows, in tree order. */
export function seedRows(columns: string[] = DEFAULT_COLUMNS): AllWorkRow[] {
  const specs: RowSpec[] = [
    {
      key: 'P-1001',
      level: 'plan',
      name: 'FY2026 Lubricants Europe',
      depth: 0,
      parentKey: null,
      planId: 'P-1001',
      status: 'published',
      owner: 'Priya Raman',
      startDate: '2026-01-01',
      endDate: '2026-12-31',
      budget: '1200000.00',
      plannedCost: '17000.00',
      lineItems: '15000.00',
      unassigned: '3000.00',
      strategic: false,
      hasChildren: true,
    },
    {
      key: 'C-1001',
      level: 'campaign',
      name: 'Helix launch',
      depth: 1,
      parentKey: 'P-1001',
      planId: 'P-1001',
      status: 'inProgress',
      owner: 'Priya Raman',
      startDate: '2026-03-01',
      endDate: '2026-06-30',
      budget: '400000.00',
      plannedCost: '17000.00',
      lineItems: '15000.00',
      unassigned: '3000.00',
      hasChildren: true,
    },
    {
      key: 'I-1001',
      level: 'initiative',
      name: 'Helix digital',
      depth: 2,
      parentKey: 'C-1001',
      planId: 'P-1001',
      status: 'inProgress',
      owner: 'Priya Raman',
      startDate: '2026-03-01',
      endDate: '2026-06-30',
      plannedCost: '17000.00',
      lineItems: '15000.00',
      unassigned: '3000.00',
      hasChildren: true,
    },
    {
      key: 'A-1001',
      level: 'activity',
      name: 'Helix launch event',
      depth: 3,
      parentKey: 'I-1001',
      planId: 'P-1001',
      status: 'approved',
      owner: 'Priya Raman',
      startDate: '2026-03-01',
      endDate: '2026-05-31',
      plannedCost: '12000.00',
      lineItems: '9000.00',
      unassigned: '3000.00',
      hasChildren: true,
    },
    {
      key: 'L-1001',
      level: 'lineItem',
      name: 'Venue hire',
      depth: 4,
      parentKey: 'A-1001',
      planId: 'P-1001',
      activityId: 'A-1001',
      status: 'filledIn',
      owner: 'Priya Raman',
      plannedCost: '9000.00',
    },
    {
      key: 'A-1001:unassigned',
      level: 'unassigned',
      name: 'Unassigned planned cost (Sys_Gen_Balance)',
      depth: 4,
      parentKey: 'A-1001',
      planId: 'P-1001',
      activityId: 'A-1001',
      plannedCost: '3000.00',
      unassigned: '3000.00',
    },
    {
      key: 'A-1002',
      level: 'activity',
      name: 'Helix social ads',
      depth: 3,
      parentKey: 'I-1001',
      planId: 'P-1001',
      status: 'declined',
      owner: 'Daniel Okoro',
      startDate: '2026-04-01',
      endDate: '2026-06-30',
      plannedCost: '5000.00',
      lineItems: '6000.00',
      unassigned: '-1000.00',
      hasChildren: true,
    },
    {
      key: 'L-1002',
      level: 'lineItem',
      name: 'Paid social',
      depth: 4,
      parentKey: 'A-1002',
      planId: 'P-1001',
      activityId: 'A-1002',
      status: 'filledIn',
      owner: 'Daniel Okoro',
      plannedCost: '6000.00',
    },
    {
      key: 'A-1002:unassigned',
      level: 'unassigned',
      name: 'Unassigned planned cost (Sys_Gen_Balance)',
      depth: 4,
      parentKey: 'A-1002',
      planId: 'P-1001',
      activityId: 'A-1002',
      plannedCost: '-1000.00',
      unassigned: '-1000.00',
    },
    {
      key: 'C-1002',
      level: 'campaign',
      name: 'Rimula fleet',
      depth: 1,
      parentKey: 'P-1001',
      planId: 'P-1001',
      status: 'inProgress',
      owner: 'Priya Raman',
      hasChildren: true,
    },
    {
      key: 'I-1002',
      level: 'initiative',
      name: 'Fleet days',
      depth: 2,
      parentKey: 'C-1002',
      planId: 'P-1001',
      status: 'inProgress',
      owner: 'Priya Raman',
      hasChildren: true,
    },
    {
      key: 'A-1003',
      level: 'activity',
      name: 'Fleet roadshow',
      depth: 3,
      parentKey: 'I-1002',
      planId: 'P-1001',
      status: 'created',
      owner: 'Priya Raman',
    },
    {
      key: 'P-1002',
      level: 'plan',
      name: 'FY2026 Mobility China',
      depth: 0,
      parentKey: null,
      planId: 'P-1002',
      status: 'draft',
      owner: 'Mei Lin',
      startDate: '2026-01-01',
      endDate: '2026-12-31',
      budget: '800000.00',
      strategic: true,
      hasChildren: true,
    },
    {
      key: 'C-1003',
      level: 'campaign',
      name: 'Retail push',
      depth: 1,
      parentKey: 'P-1002',
      planId: 'P-1002',
      status: 'noActivities',
      owner: 'Mei Lin',
      startDate: '2026-02-01',
      endDate: '2026-11-30',
    },
  ];
  return specs.map((spec) => row(spec, columns));
}

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
export const PAGE_ANSWERS = {
  'GET /api/v1/all-work/preferences': { body: defaultPreferences() },
  'GET /api/v1/all-work/options': { body: allWorkOptions() },
  'GET /api/v1/saved-views': { body: { items: [], total: 0 } },
};

const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];

/** The amounts of a row by month (1–12), in USD; the other months are empty. */
type ByMonth = Record<number, string>;

/** Twelve months, or four quarters summed from them, as the API answers them (null = "—"). */
function periodsOf(byMonth: ByMonth, period: 'month' | 'quarter'): (Money | null)[] {
  const cents = (month: number) => Math.round(Number(byMonth[month] ?? 0) * 100);
  const amount = (value: number, any: boolean) => (any ? usd((value / 100).toFixed(2)) : null);
  if (period === 'month') {
    return MONTHS.map((_, index) => (byMonth[index + 1] ? usd(byMonth[index + 1]) : null));
  }
  return [0, 1, 2, 3].map((quarter) => {
    const months = [1, 2, 3].map((offset) => quarter * 3 + offset);
    return amount(
      months.reduce((sum, month) => sum + cents(month), 0),
      months.some((month) => byMonth[month] !== undefined),
    );
  });
}

function totalOf(byMonth: ByMonth): Money | null {
  const values = Object.values(byMonth);
  if (values.length === 0) return null;
  return usd(
    (values.reduce((sum, value) => sum + Math.round(Number(value) * 100), 0) / 100).toFixed(2),
  );
}

/**
 * GET /api/v1/all-work for Budget & cost on the seed story (FR-022 – FR-024): Helix launch event runs
 * March to May with Venue hire spread evenly and 3,000.00 unassigned; Helix social ads runs April
 * to June with line items 1,000.00 above its planned cost, which is not spread.
 */
export function budgetResult(period: 'month' | 'quarter' = 'month'): AllWorkResult {
  const helixEvent: ByMonth = { 3: '4000.00', 4: '4000.00', 5: '4000.00' };
  const venue: ByMonth = { 3: '3000.00', 4: '3000.00', 5: '3000.00' };
  const unassigned: ByMonth = { 3: '1000.00', 4: '1000.00', 5: '1000.00' };
  const social: ByMonth = { 4: '2000.00', 5: '2000.00', 6: '2000.00' };
  const helix: ByMonth = { 3: '4000.00', 4: '6000.00', 5: '6000.00', 6: '2000.00' };
  const byKey: Record<string, ByMonth> = {
    'P-1001': helix,
    'C-1001': helix,
    'I-1001': helix,
    'A-1001': helixEvent,
    'L-1001': venue,
    'A-1001:unassigned': unassigned,
    'A-1002': social,
    'L-1002': social,
  };
  const rows = seedRows([]).map((candidate) => {
    const byMonth = byKey[candidate.key] ?? {};
    return {
      ...candidate,
      periods: periodsOf(byMonth, period),
      total: totalOf(byMonth),
    };
  });
  return gridResult({
    view: 'budget',
    columns: [],
    periods:
      period === 'month'
        ? MONTHS.map((label, index) => ({
            key: `2026-${String(index + 1).padStart(2, '0')}`,
            label,
          }))
        : ['Q1', 'Q2', 'Q3', 'Q4'].map((label) => ({ key: label, label })),
    rows,
    totals: {
      activities: 3,
      cells: [],
      plannedCost: usd('17000.00'),
      planBudget: usd('2000000.00'),
      periods: periodsOf(helix, period),
      total: usd('18000.00'),
    },
  });
}

/**
 * GET /api/v1/all-work for the Calendar on the seed story (FR-025): the Grid's rows without line
 * items and unassigned rows, dated 18 Oct 2026.
 */
export function calendarResult(overrides: Partial<AllWorkResult> = {}): AllWorkResult {
  return gridResult({
    view: 'calendar',
    columns: [],
    rows: seedRows([]).filter(
      (candidate) => candidate.level !== 'lineItem' && candidate.level !== 'unassigned',
    ),
    totals: { ...gridResult().totals, cells: [] },
    ...overrides,
  });
}
