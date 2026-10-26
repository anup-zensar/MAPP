/**
 * Settings records for feature 002's page tests (only tests import this file): the demo models,
 * activity types and currencies as the API answers them, and the API's error answers.
 */
import type { components } from '@/api/generated/schema';

type Model = components['schemas']['Model'];
type ActivityType = components['schemas']['ActivityType'];
type Currency = components['schemas']['Currency'];
type Usage = components['schemas']['Usage'];
type ErrorBody = components['schemas']['Error'];
type RecycleBinEntry = components['schemas']['RecycleBinEntry'];

export const PRIYA_REF = { id: 'U-1002', name: 'Priya Raman' };
export const DANIEL_REF = { id: 'U-1003', name: 'Daniel Okoro' };

/** Created by Priya on 12 Mar 2026, last changed by Daniel on 14 Mar 2026. */
export const MADE = {
  createdAt: '2026-03-12T09:00:00Z',
  createdBy: PRIYA_REF,
  updatedAt: '2026-03-14T10:00:00Z',
  updatedBy: DANIEL_REF,
};

export const NOT_USED: Usage = { inUse: false, summary: 'Not used yet', parts: [], records: [] };

export const USED_BY_TWO_PLANS: Usage = {
  inUse: true,
  summary: 'Used by 2 plans',
  parts: [{ kind: 'plan', text: 'Used by 2 plans', count: 2, inRecycleBin: 0 }],
  records: [
    { type: 'plan', id: 'P-1001', name: 'FY2026 Lubricants Europe' },
    { type: 'plan', id: 'P-1002', name: 'FY2026 Mobility' },
  ],
};

function activityType(id: string, name: string, active = true): ActivityType {
  return {
    id,
    name,
    defaultPlannedCost: null,
    models: [],
    active,
    usage: NOT_USED,
    version: 1,
    ...MADE,
  };
}

export const EVENT = activityType('AT-1002', 'Event');
export const WEBINAR = activityType('AT-1005', 'Webinar');
export const PRINT = activityType('AT-1007', 'Print advert', false);
export const ACTIVITY_TYPES = { items: [EVENT, PRINT, WEBINAR], total: 3 };

export function ref(type: ActivityType) {
  return { id: type.id, name: type.name, active: type.active };
}

/** Lubricants 1.0: 2 % and 5 %, 10,000 USD, Event and Webinar, used by 2 plans. */
export const LUBRICANTS: Model = {
  id: 'MOD-1001',
  name: 'Lubricants 1.0',
  inqToMqlRate: '2.00',
  mqlToDealRate: '5.00',
  averageDealSize: { amount: '10000.00', currency: 'USD' },
  activityTypes: [ref(EVENT), ref(WEBINAR)],
  active: true,
  usage: USED_BY_TWO_PLANS,
  version: 3,
  ...MADE,
};

/** Mobility 1.0: unused. */
export const MOBILITY: Model = {
  id: 'MOD-1002',
  name: 'Mobility 1.0',
  inqToMqlRate: '3.00',
  mqlToDealRate: '4.00',
  averageDealSize: { amount: '5000.00', currency: 'USD' },
  activityTypes: [ref(EVENT)],
  active: true,
  usage: NOT_USED,
  version: 1,
  ...MADE,
};

/** Retail 0.9: inactive, unused. */
export const RETAIL: Model = {
  ...MOBILITY,
  id: 'MOD-1003',
  name: 'Retail 0.9',
  active: false,
  activityTypes: [ref(PRINT)],
};

export const MODELS = { items: [LUBRICANTS, MOBILITY, RETAIL], total: 3 };

export const USD: Currency = {
  code: 'USD',
  name: 'US dollar',
  symbol: '$',
  rate: '1',
  inverseRate: '1',
  rateChangedAt: '2026-01-05T09:00:00Z',
  rateChangedBy: PRIYA_REF,
  active: true,
  usage: {
    inUse: true,
    summary: 'Used by 3 line items · Display currency of 4 people',
    parts: [],
    records: [],
  },
  locks: [
    { field: 'rate', reason: "USD is MAPP's base currency; its rate is always 1." },
    { field: 'active', reason: "USD is MAPP's base currency; its rate is always 1." },
    { field: 'delete', reason: "USD is MAPP's base currency; its rate is always 1." },
    { field: 'code', reason: "The code can't change once the currency is saved." },
  ],
  version: 1,
  ...MADE,
};

/** EUR at 0.92, used by 14 line items and 2 people. */
export const EUR: Currency = {
  code: 'EUR',
  name: 'Euro',
  symbol: '€',
  rate: '0.92',
  inverseRate: '1.086957',
  rateChangedAt: '2026-03-14T10:00:00Z',
  rateChangedBy: DANIEL_REF,
  active: true,
  usage: {
    inUse: true,
    summary: 'Used by 14 line items · Display currency of 2 people',
    parts: [
      { kind: 'lineItem', text: 'Used by 14 line items', count: 14, inRecycleBin: 0 },
      { kind: 'person', text: 'Display currency of 2 people', count: 2, inRecycleBin: 0 },
    ],
    records: [],
  },
  locks: [{ field: 'code', reason: "The code can't change once the currency is saved." }],
  version: 4,
  ...MADE,
};

/** JPY at 150, inactive and unused. */
export const JPY: Currency = {
  ...EUR,
  code: 'JPY',
  name: 'Japanese yen',
  symbol: '¥',
  rate: '150',
  inverseRate: '0.006667',
  active: false,
  usage: NOT_USED,
  version: 1,
};

export const CURRENCIES = { items: [USD, EUR, JPY], total: 3 };

/** The API's answer to a save with problems. */
export function validationFailed(fields: { field: string; message: string }[]): ErrorBody {
  return { error: { code: 'validation_failed', message: 'Check the highlighted fields.', fields } };
}

/** The API's answer when someone else saved first. */
export function versionConflict(): ErrorBody {
  return {
    error: {
      code: 'version_conflict',
      message: 'Someone else changed this record.',
      fields: [],
      conflict: { changedBy: 'Daniel Okoro', changedAt: '2026-03-14T10:42:00Z' },
    },
  };
}

/** The API's answer to a delete of a record in use. */
export function inUse(message: string): ErrorBody {
  return { error: { code: 'in_use', message, fields: [] } };
}

/** The API's answer to a delete: the record's recycle-bin entry, for Undo. */
export function deleted(recordTypeLabel: string, recordId: string, recordName: string) {
  const recycleBinEntry: RecycleBinEntry = {
    id: 'RB-1001',
    recordType: recordTypeLabel,
    recordTypeLabel,
    recordId,
    recordName,
    path: [],
    parent: null,
    contents: [],
    deletedAt: '2026-03-15T10:00:00Z',
    deletedBy: PRIYA_REF,
  };
  return { recycleBinEntry };
}
