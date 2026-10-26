import type { ApiError } from '@/api/errors';
import { resolvedMessage } from '@/api/errors';
import { formatDate, formatTime, localDate } from '@/format/intl';
import type { DateFormat } from '@/format/intl';
import { joinWithAnd, plural } from '@/format/text';
import { contentsText, titled } from './bin-text';
import type { BinItem } from './bin-text';

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
