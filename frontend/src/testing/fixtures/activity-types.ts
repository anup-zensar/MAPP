import type { components } from '@/api/generated/schema';
import { LUBRICANTS, MADE, NOT_USED, RETAIL } from './settings';

type ActivityType = components['schemas']['ActivityType'];

/** Event: 5,000 USD by default, allowed in Lubricants 1.0 and the inactive Retail 0.9, in use. */
export const EVENT: ActivityType = {
  id: 'AT-1002',
  name: 'Event',
  defaultPlannedCost: { amount: '5000.00', currency: 'USD' },
  models: [
    { id: LUBRICANTS.id, name: LUBRICANTS.name, active: true },
    { id: RETAIL.id, name: RETAIL.name, active: false },
  ],
  active: true,
  usage: {
    inUse: true,
    summary: 'Used by 4 activities · Allowed in 2 models',
    parts: [
      { kind: 'activity', text: 'Used by 4 activities', count: 4, inRecycleBin: 0 },
      { kind: 'model', text: 'Allowed in 2 models', count: 2, inRecycleBin: 0 },
    ],
    records: [],
  },
  version: 3,
  ...MADE,
};

/** Podcast: no default cost, no models, unused. */
export const PODCAST: ActivityType = {
  ...EVENT,
  id: 'AT-1006',
  name: 'Podcast',
  defaultPlannedCost: null,
  models: [],
  usage: NOT_USED,
  version: 1,
};

/** Print advert: inactive and unused. */
export const PRINT: ActivityType = {
  ...PODCAST,
  id: 'AT-1007',
  name: 'Print advert',
  active: false,
};

export const TYPES = { items: [EVENT, PODCAST, PRINT], total: 3 };
