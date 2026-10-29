import type { BulkKind, BulkLevel, BulkResult, LevelCounts, SelectedRecord } from './bulk-types';

export const MAX_RECORDS = 1000;
export const LIMIT_MESSAGE =
  'Bulk actions handle up to 1,000 records at a time — narrow the filters';
export const NOT_OFFERED_FIELDS =
  'Names, IDs, money, calculated fields, Ongoing, text and % split fields are edited one record at a time.';

const LEVEL_WORDS: Record<BulkLevel, [string, string]> = {
  campaign: ['campaign', 'campaigns'],
  initiative: ['initiative', 'initiatives'],
  activity: ['activity', 'activities'],
};

const ACTION_WORDS: Record<BulkKind, string> = {
  changeOwner: 'Change owner',
  changeField: 'Change field',
  submit: 'Submit for approval',
  copy: 'Copy',
  move: 'Move',
  delete: 'Delete',
};

export function actionWord(action: BulkKind): string {
  return ACTION_WORDS[action];
}

export function levelText(level: BulkLevel, count: number): string {
  const [one, many] = LEVEL_WORDS[level];
  return `${count} ${count === 1 ? one : many}`;
}

export function countByLevel(records: Iterable<SelectedRecord>): LevelCounts {
  const counts: LevelCounts = { campaigns: 0, initiatives: 0, activities: 0 };
  for (const record of records) {
    if (record.level === 'campaign') counts.campaigns += 1;
    if (record.level === 'initiative') counts.initiatives += 1;
    if (record.level === 'activity') counts.activities += 1;
  }
  return counts;
}

/** "2 campaigns, 10 activities": only the levels present. */
export function countsText(counts: LevelCounts): string {
  const parts: string[] = [];
  if (counts.campaigns > 0) parts.push(levelText('campaign', counts.campaigns));
  if (counts.initiatives > 0) parts.push(levelText('initiative', counts.initiatives));
  if (counts.activities > 0) parts.push(levelText('activity', counts.activities));
  return parts.join(', ');
}

export function levelsPresent(records: Iterable<SelectedRecord>): BulkLevel[] {
  const counts = countByLevel(records);
  const levels: BulkLevel[] = [];
  if (counts.campaigns > 0) levels.push('campaign');
  if (counts.initiatives > 0) levels.push('initiative');
  if (counts.activities > 0) levels.push('activity');
  return levels;
}

export function selectionSummary(records: SelectedRecord[]): string {
  return `${records.length} selected · ${countsText(countByLevel(records))}`;
}

/** "33 changed · 2 skipped · 1 failed": skipped and failed only when there are some. */
export function resultSummary(result: BulkResult): string {
  const parts = [`${result.counts.done} ${result.verb}`];
  if (result.counts.skipped > 0) parts.push(`${result.counts.skipped} skipped`);
  if (result.counts.failed > 0) parts.push(`${result.counts.failed} failed`);
  return parts.join(' · ');
}

export function recordsText(count: number): string {
  return `${count} ${count === 1 ? 'record' : 'records'}`;
}
