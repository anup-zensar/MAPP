import type { components } from '@/api/generated/schema';

type Schemas = components['schemas'];

const NONE = { campaigns: 0, initiatives: 0, activities: 0, lineItems: 0 };

export function importCheck(
  overrides: Partial<Schemas['ImportCheck']> = {},
): Schemas['ImportCheck'] {
  return {
    fileName: 'plan.xlsx',
    rows: 3,
    ready: 2,
    withProblems: 1,
    levels: [
      { level: 'campaign', create: 1, update: 0, unchanged: 0, leftOut: 0 },
      { level: 'initiative', create: 0, update: 0, unchanged: 0, leftOut: 0 },
      { level: 'activity', create: 0, update: 1, unchanged: 0, leftOut: 1 },
      { level: 'lineItem', create: 0, update: 0, unchanged: 0, leftOut: 0 },
    ],
    problems: [
      {
        sheet: 'Activities',
        row: 4,
        column: 'End date',
        value: '31 Feb 2026',
        reason: 'Not a real date — write it as 2026-03-12',
      },
    ],
    ignoredColumns: ['Activities: Notes'],
    needMoreFields: 1,
    nothingToChange: false,
    ...overrides,
  };
}

export function importSummary(
  overrides: Partial<Schemas['ImportSummary']> = {},
): Schemas['ImportSummary'] {
  return {
    id: 'IMP-1001',
    planId: 'P-1001',
    fileName: 'plan.xlsx',
    status: 'imported',
    importedAt: '2026-10-18T09:00:00Z',
    importedBy: { id: 'U-1002', name: 'Priya Raman' },
    rowsRead: 3,
    rowsLeftOut: 1,
    created: { ...NONE, campaigns: 1 },
    updated: { ...NONE, activities: 1 },
    undoneAt: null,
    undoneBy: null,
    ...overrides,
  };
}

export function importDetail(
  overrides: Partial<Schemas['ImportDetail']> = {},
): Schemas['ImportDetail'] {
  return {
    import: importSummary(),
    problems: importCheck().problems,
    records: [
      { id: 'C-1009', level: 'campaign', name: 'Spring push', action: 'created', exists: true },
      { id: 'A-1001', level: 'activity', name: 'Launch film', action: 'updated', exists: true },
    ],
    undo: { ...NONE, campaigns: 1, alreadyDeleted: 0 },
    kept: [],
    ...overrides,
  };
}
