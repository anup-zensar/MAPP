/**
 * The words of the recycle bin page (feature 011, plan "Recycle bin page"): the count line, each
 * row's texts, the search over an entry's contents and the type pills.
 */
import type { components } from '@/api/generated/schema';
import { formatWhole } from '@/format/decimal';
import { joinWithAnd, plural } from '@/format/text';

export type BinItem = components['schemas']['RecycleBinListItem'];
export type BinRecord = components['schemas']['RecycleBinRecord'];
export type ContentCount = components['schemas']['ContentCount'];
export type RecordRef = components['schemas']['RecordRef'];

/** "line item" → "Line item". */
export function capitalise(text: string): string {
  return text.charAt(0).toUpperCase() + text.slice(1);
}

/** "Initiative 'Trade shows'": a record named with its type, at the start of a sentence. */
export function titled(typeLabel: string, name: string): string {
  return `${capitalise(typeLabel)} '${name}'`;
}

/** The header's description (FR-014); the count is at the end of the toolbar. */
export const BIN_DESCRIPTION =
  'Deleted records stay here until you restore them or delete them permanently.';

/** What the empty recycle bin says under its title (FR-014). */
export const EMPTY_BIN_TEXT =
  'When you delete a plan, campaign, initiative, activity or line item, it comes here so you can bring it back.';

/** "2 activities and 5 line items"; '' when nothing was inside. */
export function contentsText(contents: ContentCount[]): string {
  return joinWithAnd(contents.map((content) => `${content.count} ${content.label}`));
}

/** A row's contents: "2 activities · 5 line items", or "Nothing inside" (FR-015). */
export function rowContentsText(contents: ContentCount[]): string {
  return contents.length > 0
    ? contents.map((content) => `${content.count} ${content.label}`).join(' · ')
    : 'Nothing inside';
}

/** "Was in FY2026 Lubricants Europe › Helix launch"; '' for a top-level record (FR-015). */
export function wasInText(path: RecordRef[]): string {
  return path.length > 0 ? `Was in ${path.map((record) => record.name).join(' › ')}` : '';
}

/** Why a row cannot simply come back (FR-017); '' when it can. */
export function restoreStateLine(item: BinItem): string {
  const parent = item.missingParent;
  if (!parent) {
    return '';
  }
  switch (item.restoreState) {
    case 'parentInRecycleBin':
      return `Its ${parent.typeLabel} '${parent.name}' is also in the recycle bin`;
    case 'parentDeletedPermanently':
      return `Can't be restored — its ${parent.typeLabel} '${parent.name}' was permanently deleted`;
    default:
      return '';
  }
}

/** Whether an entry matches the search, and the line that says which record inside matched. */
export interface SearchMatch {
  matches: boolean;
  /** "Contains A-1042 'Film shoot' and 2 more" when only records inside matched; '' otherwise. */
  containsText: string;
}

/**
 * The search on name or ID (FR-016): it looks at the entry's record and every record inside it,
 * ignoring letter case.
 */
export function matchSearch(item: BinItem, search: string): SearchMatch {
  const query = search.trim().toLowerCase();
  if (!query) {
    return { matches: true, containsText: '' };
  }
  const hit = (name: string, id: string) =>
    name.toLowerCase().includes(query) || id.toLowerCase().includes(query);
  if (hit(item.recordName, item.recordId)) {
    return { matches: true, containsText: '' };
  }
  const [first, ...others] = insideRecords(item).filter((record) =>
    hit(record.recordName, record.recordId),
  );
  if (!first) {
    return { matches: false, containsText: '' };
  }
  const more = others.length > 0 ? ` and ${others.length} more` : '';
  return { matches: true, containsText: `Contains ${first.recordId} '${first.recordName}'${more}` };
}

/** The records inside an entry, without its own record (which `records` lists first). */
export function insideRecords(item: BinItem): BinRecord[] {
  return (item.records ?? []).filter((record) => record.recordId !== item.recordId);
}

/** Whether a record is the entry's own record or one of the records inside it. */
export function holdsRecord(item: BinItem, recordId: string): boolean {
  return item.recordId === recordId || insideRecords(item).some((r) => r.recordId === recordId);
}

/** The type pills: All and the five planning levels, then any other type in the bin (FR-016). */
export function typePills(items: BinItem[]): { value: string; label: string }[] {
  const pills = [
    { value: 'all', label: 'All' },
    { value: 'plan', label: 'Plans' },
    { value: 'campaign', label: 'Campaigns' },
    { value: 'initiative', label: 'Initiatives' },
    { value: 'activity', label: 'Activities' },
    { value: 'lineItem', label: 'Line items' },
  ];
  for (const item of items) {
    if (!pills.some((pill) => pill.value === item.recordType)) {
      const pluralLabel = item.recordTypePlural ?? `${item.recordTypeLabel}s`;
      pills.push({ value: item.recordType, label: capitalise(pluralLabel) });
    }
  }
  return pills;
}

/**
 * "3 of 12 items" at the end of the toolbar (FR-016), written as every list writes its count
 * (design system §11, rule 14).
 */
export function showingLine(shown: number, total: number): string {
  return `${formatWhole(shown)} of ${plural(total, 'item')}`;
}
