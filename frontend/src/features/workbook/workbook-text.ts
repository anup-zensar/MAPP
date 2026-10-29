import type { components } from '@/api/generated/schema';
import type { ImportCheck, ImportLevel } from './api';

type LevelTotals = components['schemas']['ImportLevelTotals'];

export const LEVEL_PLURALS: Record<ImportLevel, string> = {
  campaign: 'Campaigns',
  initiative: 'Initiatives',
  activity: 'Activities',
  lineItem: 'Line items',
};

const LEVEL_WORDS: Record<ImportLevel, [string, string]> = {
  campaign: ['campaign', 'campaigns'],
  initiative: ['initiative', 'initiatives'],
  activity: ['activity', 'activities'],
  lineItem: ['line item', 'line items'],
};

export const FILE_TOO_LARGE =
  'The file is larger than 5 MB — remove unused rows or columns, or split it.';
export const CHOOSE_FILE = 'Choose an .xlsx or .csv file.';

export function rowsText(count: number): string {
  return `${count} ${count === 1 ? 'row' : 'rows'}`;
}

export function checkSummary(check: ImportCheck): string {
  return `${rowsText(check.rows)} · ${check.ready} ready · ${check.withProblems} with problems`;
}

/** What the import will do: "3 to create · 12 to update"; "Nothing to change" when neither. */
export function changeSummary(check: ImportCheck): string {
  const create = check.levels.reduce((sum, l) => sum + l.create, 0);
  const update = check.levels.reduce((sum, l) => sum + l.update, 0);
  if (create + update === 0) {
    return 'Nothing to change';
  }
  return [create > 0 && `${create} to create`, update > 0 && `${update} to update`]
    .filter(Boolean)
    .join(' · ');
}

/** "1 campaign, 2 activities": only the levels present; "nothing" when none. */
export function totalsText(totals: LevelTotals): string {
  const parts = (
    [
      ['campaign', totals.campaigns],
      ['initiative', totals.initiatives],
      ['activity', totals.activities],
      ['lineItem', totals.lineItems],
    ] as const
  )
    .filter(([, count]) => count > 0)
    .map(([level, count]) => `${count} ${LEVEL_WORDS[level][count === 1 ? 0 : 1]}`);
  return parts.length > 0 ? parts.join(', ') : 'nothing';
}
