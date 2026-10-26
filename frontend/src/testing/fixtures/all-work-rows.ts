import type {
  AllWorkCell,
  AllWorkColumn,
  AllWorkRow,
  AllWorkStatus,
  Money,
} from '@/features/all-work';

export function usd(amount: string): Money {
  return { amount, currency: 'USD' };
}

type Level = AllWorkRow['level'];
const ALL: Level[] = ['plan', 'campaign', 'initiative', 'activity', 'lineItem'];
const CONTAINERS: Level[] = ['plan', 'campaign', 'initiative'];

export function column(
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
export interface RowSpec {
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
      const status = spec.status ? (STATUS[spec.status] ?? null) : null;
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
  const status = spec.status ? (STATUS[spec.status] ?? null) : null;
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
