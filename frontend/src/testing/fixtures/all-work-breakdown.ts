import type { HttpHandler } from 'msw';
import type { components } from '@/api/generated/schema';
import { reply } from '../replies';

type Schemas = components['schemas'];
type Breakdown = Schemas['AllWorkBreakdown'];
type Group = Schemas['AllWorkBreakdownGroup'];
type Grouping = Schemas['AllWorkGrouping'];
type Attribute = Schemas['AllWorkFilterableAttribute'];

export const BRANDS: Attribute = {
  id: 'SA-1020',
  label: 'Brands',
  level: 'activity',
  type: 'multipleChoice',
  values: [
    { id: 'SO-1201', label: 'Helix', inactive: false },
    { id: 'SO-1202', label: 'Rimula', inactive: false },
  ],
};

export const SPEND_TYPE: Attribute = {
  id: 'SA-1030',
  label: 'Spend type',
  level: 'lineItem',
  type: 'singleChoice',
  values: [
    { id: 'SO-1301', label: 'Media', inactive: false },
    { id: 'SO-1302', label: 'Events', inactive: false },
  ],
};

export const COUNTRY: Attribute = {
  id: 'SA-1013',
  label: 'Country / OU',
  level: 'campaign',
  type: 'multipleChoice',
  values: [
    { id: 'SO-1101', label: 'Germany', inactive: false },
    { id: 'SO-1102', label: 'France', inactive: false },
  ],
};

export const BY_SPEND_TYPE: Grouping = {
  key: 'attr:SA-1030',
  label: 'Spend type',
  level: 'lineItem',
  hasBudget: false,
};

export const GROUPINGS: Grouping[] = [
  { key: 'plan', label: 'Plan', level: null, hasBudget: true },
  { key: 'activityType', label: 'Activity type', level: null, hasBudget: false },
  { key: 'attr:SA-1013', label: 'Country / OU', level: 'campaign', hasBudget: true },
  { key: 'attr:SA-1020', label: 'Brands', level: 'activity', hasBudget: false },
  BY_SPEND_TYPE,
];

/** A group of `planned` dollars, its share and counts; children only for a then-by grouping. */
export function group(key: string, label: string, planned: string, rest: Partial<Group> = {}) {
  return {
    key,
    label,
    activities: 1,
    lineItems: 1,
    plannedCost: { amount: planned, currency: 'USD' },
    share: '50.00',
    budget: null,
    budgetUsedPercent: null,
    overBudget: false,
    children: [],
    ...rest,
  } satisfies Group;
}

export function breakdown(rest: Partial<Breakdown> = {}): Breakdown {
  return {
    by: BY_SPEND_TYPE,
    thenBy: null,
    currency: 'USD',
    groups: [
      group('SO-1301', 'Media', '6000.00', { share: '60.00' }),
      group('notOnLineItem', 'Not on a line item', '4000.00', { share: '40.00', lineItems: 0 }),
    ],
    total: group('total', 'Total', '10000.00', { share: '100.00', activities: 2, lineItems: 1 }),
    countsInEach: false,
    notes: [],
    groupings: GROUPINGS,
    ...rest,
  };
}

/** The answers a page with a Planned cost by card needs when a test does not care about it. */
export const NO_BREAKDOWN: HttpHandler[] = [
  reply('GET /api/v1/all-work/options', {
    body: {
      year: 2026,
      currentYear: 2026,
      years: [2026],
      plans: [],
      owners: [],
      teams: [],
      businessUnits: [],
      activityTypes: [],
      columnGroups: [],
      defaultColumns: [],
      attributes: [],
    },
  }),
  reply('GET /api/v1/all-work/breakdown', { body: breakdown({ groups: [] }) }),
];
