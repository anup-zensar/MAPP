/**
 * The words of the recycle bin (feature 011, plan "Recycle bin page", "Restore sentences" and
 * "User-visible messages"): the count line, each row's texts, the search over an entry's contents,
 * the type pills, and the restore, permanent-delete and empty dialogs and their messages.
 */
import type { ApiError } from '@/api/errors';
import { resolvedMessage } from '@/api/errors';
import type { components } from '@/api/schema';
import { formatDate, formatTime, localDate } from '@/lib/format';
import type { DateFormat } from '@/lib/format';
import { recordHref } from '@/lib/recordLinks';
import type { PlanningRecordType } from '@/lib/recordLinks';
import { formatWhole } from '@/lib/decimal';
import { joinWithAnd, plural } from '@/lib/text';

export type BinItem = components['schemas']['RecycleBinListItem'];
export type BinRecord = components['schemas']['RecycleBinRecord'];
export type ContentCount = components['schemas']['ContentCount'];
export type RestorePreview = components['schemas']['RestorePreview'];
export type RestorePreviewChange = components['schemas']['RestorePreviewChange'];
type RecordRef = components['schemas']['RecordRef'];
type LabelledRef = components['schemas']['LabelledRef'];

/** "line item" → "Line item". */
export function capitalise(text: string): string {
  return text.charAt(0).toUpperCase() + text.slice(1);
}

/** "Initiative 'Trade shows'": a record named with its type, at the start of a sentence. */
function titled(typeLabel: string, name: string): string {
  return `${capitalise(typeLabel)} '${name}'`;
}

// ── The page ──────────────────────────────────────────────────────────────────────────────

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
  const inside = insideRecords(item).filter((record) => hit(record.recordName, record.recordId));
  if (inside.length === 0) {
    return { matches: false, containsText: '' };
  }
  const [first] = inside;
  const more = inside.length > 1 ? ` and ${inside.length - 1} more` : '';
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

// ── Restore ───────────────────────────────────────────────────────────────────────────────

/** "Restore initiative 'Trade shows'?" */
export function restoreTitle(item: BinItem): string {
  return `Restore ${item.recordTypeLabel} '${item.recordName}'?`;
}

/** "Restore initiative", "Restore both" or "Restore all 3" (FR-023). */
export function restoreConfirmLabel(item: BinItem, parentEntries: number): string {
  if (parentEntries === 0) {
    return `Restore ${item.recordTypeLabel}`;
  }
  return parentEntries === 1 ? 'Restore both' : `Restore all ${parentEntries + 1}`;
}

/**
 * Where it returns (FR-018): "It returns to FY2026 Lubricants Europe › Helix launch with 2
 * activities and 5 line items."; at the top: "It returns to the Plans list" (plans) or "It
 * returns to its list".
 */
export function destinationSentence(preview: RestorePreview): string {
  const { entry, destination } = preview;
  const place =
    destination.length > 0
      ? destination.map((record) => record.name).join(' › ')
      : entry.recordType === 'plan'
        ? 'the Plans list'
        : 'its list';
  const contents = contentsText(entry.contents);
  return `It returns to ${place}${contents ? ` with ${contents}` : ''}.`;
}

/** One sentence per change the restore makes (plan "Restore sentences", FR-019, FR-020, FR-022). */
export function changeSentence(change: RestorePreviewChange, entry: BinItem): string {
  const subject =
    change.recordId === entry.recordId
      ? 'It'
      : `${titled(change.recordTypeLabel, change.recordName)} (${change.recordId})`;
  if (change.field === 'status') {
    if (change.restored && change.recordType === 'activity') {
      return `${subject} comes back as ${change.newText} and must be submitted for approval again.`;
    }
    if (change.restored && change.recordType === 'plan') {
      return `${subject} comes back as ${change.newText} and must be published again.`;
    }
    if (!change.restored && change.recordType === 'activity') {
      return `${subject} goes back to ${change.newText} because its line items change, and must be submitted for approval again.`;
    }
    return `${subject} changes from ${change.oldText} to ${change.newText}.`;
  }
  if (change.field === 'name') {
    return `${subject} comes back as '${change.newText}' because the name '${change.oldText}' is taken.`;
  }
  return `${subject}: ${change.label} changes from ${change.oldText} to ${change.newText}.`;
}

/** "Its initiative 'Trade shows' is also in the recycle bin. Restoring brings these back first:" */
export function parentsIntro(parent: LabelledRef): string {
  return `Its ${parent.typeLabel} '${parent.name}' is also in the recycle bin. Restoring brings these back first:`;
}

/** "Initiative 'Trade shows' (I-1009) with 1 activity and 1 line item" (FR-023). */
export function parentEntryLine(entry: BinItem): string {
  const contents = contentsText(entry.contents);
  return `${titled(entry.recordTypeLabel, entry.recordName)} (${entry.recordId})${contents ? ` with ${contents}` : ''}`;
}

/** "This activity can't be restored" (FR-024). */
export function blockedTitle(item: BinItem): string {
  return `This ${item.recordTypeLabel} can't be restored`;
}

/** Why, and what can be done instead (FR-024). */
export function blockedText(item: BinItem): string {
  const parent = item.missingParent;
  const because = parent
    ? ` because its ${parent.typeLabel} '${parent.name}' was permanently deleted`
    : '';
  return `This ${item.recordTypeLabel} can't be restored${because}. You can delete it permanently instead.`;
}

/** "Initiative 'Trade shows' restored" — the message with View (FR-026). */
export function restoredMessage(typeLabel: string, name: string): string {
  return `${titled(typeLabel, name)} restored`;
}

/**
 * The page of a restored record, for the message's View: a planning record's page, inside the
 * plan it returned to, or a global budget's page (014 FR-035); undefined for records without a page.
 */
export function restoredHref(
  recordType: string,
  recordId: string,
  destination: RecordRef[],
): string | undefined {
  if (recordType === 'globalBudget') {
    return `/budgets/${encodeURIComponent(recordId)}`;
  }
  const planningTypes: string[] = ['plan', 'campaign', 'initiative', 'activity', 'lineItem'];
  if (!planningTypes.includes(recordType)) {
    return undefined;
  }
  const planId = destination.find((record) => record.type === 'plan')?.id;
  if ((recordType === 'campaign' || recordType === 'initiative') && !planId) {
    return undefined;
  }
  return recordHref(recordType as PlanningRecordType, recordId, planId);
}

// ── Delete permanently and empty ──────────────────────────────────────────────────────────

/** "Delete 'Trade shows' permanently?" (FR-028) */
export function purgeTitle(item: BinItem): string {
  return `Delete '${item.recordName}' permanently?`;
}

/** "This also deletes 1 activity and 1 line item."; '' when nothing was inside (FR-028). */
export function purgeContentsText(item: BinItem): string {
  const contents = contentsText(item.contents);
  return contents ? `This also deletes ${contents}.` : '';
}

/** The other entries that can no longer come back once this one is gone (FR-028); '' for none. */
export function insideEntriesWarning(count: number): string {
  if (count === 0) {
    return '';
  }
  return count === 1
    ? '1 other item in the recycle bin belongs inside it and can no longer be restored.'
    : `${count} other items in the recycle bin belong inside it and can no longer be restored.`;
}

/** "Initiative 'Trade shows' deleted permanently" (FR-030). */
export function purgedMessage(typeLabel: string, name: string): string {
  return `${titled(typeLabel, name)} deleted permanently`;
}

/**
 * What emptying deletes (FR-029): "This permanently deletes 3 items and everything inside them:
 * 2 activities and 5 line items."
 */
export function emptyBinText(items: BinItem[]): string {
  const totals = contentsTotals(items);
  const what = `This permanently deletes ${plural(items.length, 'item')}`;
  return totals ? `${what} and everything inside them: ${totals}.` : `${what}.`;
}

/** The plural names of the planning records, for counts the API only gave as singular. */
const PLANNING_PLURALS: Record<string, string | undefined> = {
  plan: 'plans',
  campaign: 'campaigns',
  initiative: 'initiatives',
  activity: 'activities',
  lineItem: 'line items',
};

/** The contents of all entries added up by record type: "3 activities and 6 line items". */
function contentsTotals(items: BinItem[]): string {
  const totals = new Map<string, { count: number; singular?: string; plural?: string }>();
  for (const item of items) {
    for (const content of item.contents) {
      const total = totals.get(content.recordType) ?? { count: 0 };
      total.count += content.count;
      if (content.count === 1) {
        total.singular = content.label;
      } else {
        total.plural = content.label;
      }
      totals.set(content.recordType, total);
    }
  }
  return joinWithAnd(
    [...totals].map(([recordType, total]) => {
      if (total.count === 1) {
        return `1 ${total.singular ?? total.plural}`;
      }
      const label =
        total.plural ??
        items.find((item) => item.recordType === recordType)?.recordTypePlural ??
        PLANNING_PLURALS[recordType] ??
        `${total.singular}s`;
      return `${total.count} ${label}`;
    }),
  );
}

/** The message after emptying (FR-029, FR-030). */
export function emptiedMessage(count: number): string {
  if (count === 0) {
    return 'Nothing was deleted — someone else already emptied the recycle bin.';
  }
  return `The recycle bin is empty — ${plural(count, 'item')} deleted permanently`;
}

/**
 * "'Trade shows' is no longer in the recycle bin — Priya Raman restored it at 14:05" (FR-032),
 * also for an entry someone deleted permanently ("deleted it permanently").
 */
export function alreadyResolvedText(
  error: ApiError,
  name: string,
  dateFormat?: DateFormat,
): string {
  const resolution = error.resolution;
  if (resolution?.how !== 'deletedPermanently') {
    return resolvedMessage(error, name, dateFormat);
  }
  const day = localDate(resolution.at);
  const time = `at ${formatTime(resolution.at)}`;
  const when = day === localDate(new Date()) ? time : `on ${formatDate(day, dateFormat)} ${time}`;
  return `'${name}' is no longer in the recycle bin — ${resolution.by.name} deleted it permanently ${when}`;
}
