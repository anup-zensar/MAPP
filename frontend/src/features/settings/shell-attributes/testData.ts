/**
 * Shell attributes and options for the tests of this folder, shaped as the API answers them
 * (backend/api/openapi.yaml). Only the tests use them.
 */
import type { components } from '@/api/schema';

type ShellAttribute = components['schemas']['ShellAttribute'];
type ShellAttributeOption = components['schemas']['ShellAttributeOption'];
type Usage = components['schemas']['Usage'];

export const PRIYA_REF = { id: 'U-1002', name: 'Priya Raman' };
export const DANIEL_REF = { id: 'U-1003', name: 'Daniel Okoro' };

export const NOT_USED: Usage = { inUse: false, summary: 'Not used yet', parts: [], records: [] };

/** "Used by <n> activities", counted in `count`. */
export function usedBy(summary: string, count = 1): Usage {
  return {
    inUse: true,
    summary,
    parts: [{ kind: 'activity', text: summary, count, inRecycleBin: 0 }],
    records: [],
  };
}

/** A Shell attribute created by Priya on 12 Mar 2026 and last changed by Daniel on 14 Mar 2026. */
export function shellAttribute(
  overrides: Partial<ShellAttribute> & { id: string; label: string },
): ShellAttribute {
  return {
    helpText: '',
    level: 'activity',
    type: 'singleChoice',
    marking: 'optional',
    section: 'Shell attributes',
    position: 1,
    parentAttribute: null,
    condition: [],
    businessUnit: false,
    active: true,
    optionCount: 0,
    activeOptionCount: 0,
    usage: NOT_USED,
    locks: [],
    version: 1,
    createdAt: '2026-03-12T09:00:00Z',
    createdBy: PRIYA_REF,
    updatedAt: '2026-03-14T10:00:00Z',
    updatedBy: DANIEL_REF,
    ...overrides,
  };
}

/** An option of `attributeId`, active and unused unless told otherwise. */
export function shellOption(
  overrides: Partial<ShellAttributeOption> & { id: string; label: string; attributeId: string },
): ShellAttributeOption {
  return {
    position: 1,
    parentOptions: [],
    active: true,
    usage: NOT_USED,
    version: 1,
    createdAt: '2026-03-12T09:00:00Z',
    createdBy: PRIYA_REF,
    updatedAt: '2026-03-12T09:00:00Z',
    updatedBy: PRIYA_REF,
    ...overrides,
  };
}

/** The demo's activity attributes: two sections, an inactive one, one in use. */
export const MEDIA_CHANNEL = shellAttribute({
  id: 'SA-1010',
  label: 'Media channel',
  marking: 'neededToSubmit',
  position: 1,
  optionCount: 4,
  activeOptionCount: 4,
  usage: usedBy('Used by 12 activities (1 in the recycle bin)', 12),
});

export const PIR = shellAttribute({
  id: 'SA-1011',
  label: 'PIR',
  position: 2,
  optionCount: 2,
  activeOptionCount: 2,
});

export const BRANDS = shellAttribute({
  id: 'SA-1012',
  label: 'Brands',
  type: 'multipleChoiceSplit',
  marking: 'neededToSubmit',
  section: 'Lubricants attributes',
  position: 3,
  optionCount: 4,
  activeOptionCount: 4,
  condition: [{ id: 'SO-BU-LUB', name: 'Lubricants', active: true }],
});

export const OLD_NOTES = shellAttribute({
  id: 'SA-1013',
  label: 'Old notes',
  type: 'longText',
  section: 'Lubricants attributes',
  position: 4,
  active: false,
});

export const ACTIVITY_ATTRIBUTES = [MEDIA_CHANNEL, PIR, BRANDS, OLD_NOTES];

/** The plan-level Business unit attribute, locked because conditions depend on it (FR-023). */
export const BUSINESS_UNIT_LOCK =
  "Business unit can't be deleted, set inactive, moved or given another type, because other attributes are shown or hidden by it.";

export const BUSINESS_UNIT = shellAttribute({
  id: 'SA-1001',
  label: 'Business unit',
  level: 'plan',
  type: 'multipleChoice',
  marking: 'neededToPublish',
  businessUnit: true,
  optionCount: 3,
  activeOptionCount: 3,
  locks: ['type', 'level', 'section', 'position', 'active', 'delete'].map((field) => ({
    field,
    reason: BUSINESS_UNIT_LOCK,
  })),
});

/** `{ items, total }` as the list answers of the API. */
export function page<T>(items: T[], total = items.length) {
  return { items, total };
}
