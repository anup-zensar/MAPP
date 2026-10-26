/**
 * The words of the activity log (feature 011, plan "Activity log page" and "User-visible
 * messages"): each entry's sentence, action chip, details and record link, and the day headings
 * and count lines of the list. Pure functions, so the log page, the History sections and their
 * tests share one wording.
 */
import type { components } from '@/api/generated/schema';
import type { StatusTone } from '@/ui/StatusChip/StatusChip';
import { formatWhole } from '@/format/decimal';
import { formatWeekdayDate } from '@/format/intl';
import { recordHref } from '@/ui/record-links';
import type { PlanningRecordType } from '@/ui/record-links';
import { plural } from '@/format/text';

export type ActivityLogEntry = components['schemas']['ActivityLogEntry'];
export type LogAction = ActivityLogEntry['action'];
export type RecordRef = components['schemas']['RecordRef'];

/** The action chip of each action: its word and colour (FR-005, design system §4). */
export const ACTION_CHIPS: Record<LogAction, { word: string; tone: StatusTone }> = {
  created: { word: 'Created', tone: 'positive' },
  changed: { word: 'Changed', tone: 'neutral' },
  moved: { word: 'Moved', tone: 'neutral' },
  copied: { word: 'Copied', tone: 'neutral' },
  statusChanged: { word: 'Status changed', tone: 'active' },
  restored: { word: 'Restored', tone: 'active' },
  deleted: { word: 'Deleted', tone: 'problem' },
  deletedPermanently: { word: 'Permanently deleted', tone: 'problem' },
  emptiedRecycleBin: { word: 'Permanently deleted', tone: 'problem' },
};

/** The planning records, which have their own page (recordHref), with their mid-sentence names. */
const PLANNING_LABELS: Record<PlanningRecordType, string> = {
  plan: 'plan',
  campaign: 'campaign',
  initiative: 'initiative',
  activity: 'activity',
  lineItem: 'line item',
};

/** "line item" → "Line item": a type label at the start of a sentence or a chip. */
export function capitalise(text: string): string {
  return text.charAt(0).toUpperCase() + text.slice(1);
}

/** The mid-sentence name of a planning record type ("line item"); the type itself otherwise. */
export function planningLabel(type: string): string {
  return isPlanningType(type) ? PLANNING_LABELS[type] : type;
}

export function isPlanningType(type: string): type is PlanningRecordType {
  return Object.hasOwn(PLANNING_LABELS, type);
}

/** "PR" for Priya Raman: the first letters of the first and the last name (FR-005 avatar). */
export function initialsOf(name: string): string {
  const [first, ...others] = name.trim().split(/\s+/).filter(Boolean);
  if (first === undefined) {
    return '?';
  }
  const last = others.at(-1) ?? '';
  return (first.charAt(0) + last.charAt(0)).toUpperCase();
}

/**
 * The start of an entry's sentence, before the record's name: "Priya Raman changed activity"
 * (FR-005, FR-006). An emptied recycle bin has no record: "Priya Raman emptied the recycle bin
 * (3 items)", the whole sentence.
 */
export function sentenceLead(entry: ActivityLogEntry): string {
  if (entry.action === 'emptiedRecycleBin') {
    const count = entry.items?.length ?? 0;
    return `${entry.actor.name} emptied the recycle bin (${plural(count, 'item')})`;
  }
  return `${entry.actor.name} ${entry.verb} ${entry.recordTypeLabel}`;
}

/** The whole sentence as text: "Priya Raman changed activity 'Film shoot'". */
export function entrySentence(entry: ActivityLogEntry): string {
  if (entry.action === 'emptiedRecycleBin') {
    return sentenceLead(entry);
  }
  return `${sentenceLead(entry)} '${entry.recordName}'`;
}

/** "in FY2026 Lubricants Europe › Helix launch": where the record was at the time; '' at the top. */
export function pathText(path: RecordRef[]): string {
  return path.length > 0 ? `in ${path.map((record) => record.name).join(' › ')}` : '';
}

/** How the record's name shows in an entry (FR-008). */
export type RecordLink =
  /** A link to the record, with " — now '<name>'" when it was renamed since. */
  | { kind: 'record'; href: string; now: string }
  /** A link to the recycle bin with the record highlighted. */
  | { kind: 'recycleBin'; href: string }
  /** Plain text, with " (permanently deleted)" or nothing after it. */
  | { kind: 'text'; suffix: string };

/**
 * Where the record's name in an entry leads (FR-008): an existing planning record opens its page;
 * a record in the recycle bin opens the bin with its row highlighted; a permanently deleted record
 * is plain text with "(permanently deleted)". Records without a page (settings, people) and
 * entries without a known state are plain text.
 */
export function recordLink(entry: ActivityLogEntry): RecordLink {
  const now = entry.recordNow;
  if (!now) {
    return { kind: 'text', suffix: '' };
  }
  switch (now.state) {
    case 'exists': {
      const href = pageOf(entry.recordType, entry.recordId, now.path);
      if (!href) {
        return { kind: 'text', suffix: renamed(entry) };
      }
      return { kind: 'record', href, now: renamed(entry) };
    }
    case 'inRecycleBin':
      return {
        kind: 'recycleBin',
        href: `/recycle-bin?${new URLSearchParams({ record: entry.recordId })}`,
      };
    case 'deletedPermanently':
      return { kind: 'text', suffix: ' (permanently deleted)' };
    case 'none':
      return { kind: 'text', suffix: '' };
  }
}

/** " — now 'Film shoot (final)'" when the record has another name today; '' otherwise. */
function renamed(entry: ActivityLogEntry): string {
  const name = entry.recordNow?.name ?? '';
  return name && name !== entry.recordName ? ` — now '${name}'` : '';
}

/** The pages of records outside the plan tree: global budgets (014) and GSAP transactions (015). */
const OTHER_PAGES = new Map<string, (id: string) => string>([
  ['globalBudget', (id) => `/budgets/${encodeURIComponent(id)}`],
  ['financialTransaction', (id) => `/financials/transactions/${encodeURIComponent(id)}`],
]);

/** The page of a record; undefined for records without one or when its plan is unknown. */
function pageOf(type: string, id: string, path: RecordRef[]): string | undefined {
  const otherPage = OTHER_PAGES.get(type);
  if (otherPage) {
    return otherPage(id);
  }
  if (!isPlanningType(type)) {
    return undefined;
  }
  const planId = path.find((record) => record.type === 'plan')?.id;
  if ((type === 'campaign' || type === 'initiative') && !planId) {
    return undefined;
  }
  return recordHref(type, id, planId);
}

/**
 * What **Show details** reveals (FR-007): each change "<label>: <old> → <new>", the comment, the
 * details by action ("Deleted with 2 activities" for a delete), and for an emptied recycle bin one
 * line per item: "Initiative 'Trade shows' (I-1009) with 1 activity and 1 line item".
 */
export function detailLines(entry: ActivityLogEntry): string[] {
  const lines = entry.changes.map(
    (change) => `${change.label}: ${change.oldText} → ${change.newText}`,
  );
  if (entry.comment) {
    lines.push(`Comment: ${entry.comment}`);
  }
  if (entry.details) {
    lines.push(entry.action === 'deleted' ? `Deleted ${entry.details}` : entry.details);
  }
  for (const item of entry.items ?? []) {
    const text = `${capitalise(item.recordTypeLabel)} '${item.recordName}' (${item.recordId})`;
    lines.push(item.details ? `${text} ${item.details}` : text);
  }
  return lines;
}

/** "Today", "Yesterday" or "Monday 12 Oct 2026" for a local calendar date (FR-004). */
export function dayLabel(date: string, today: string): string {
  if (date === today) {
    return 'Today';
  }
  if (date === dayBefore(today)) {
    return 'Yesterday';
  }
  return formatWeekdayDate(date);
}

/** The calendar day before "2026-03-01": "2026-02-28" (worked out in UTC, so no time zone moves it). */
function dayBefore(date: string): string {
  const day = new Date(`${date}T00:00:00Z`);
  day.setUTCDate(day.getUTCDate() - 1);
  return day.toISOString().slice(0, 10);
}

/** "1 entry", "12 entries", "1,204 entries". */
export function entriesText(count: number): string {
  return plural(count, 'entry', 'entries');
}

/**
 * The count at the end of the toolbar (FR-003), written as every list writes it (design system
 * §11, rule 14): "12 of 1,204 entries", and "1,204 of 1,204 entries" when no filter is in use.
 */
export function countLine(total: number, unfilteredTotal: number): string {
  return `${formatWhole(total)} of ${entriesText(unfilteredTotal)}`;
}

/** "Showing the last 50 of 1,204" under the list while more entries can be loaded (FR-009). */
export function loadedLine(loaded: number, total: number): string {
  return `Showing the last ${formatWhole(loaded)} of ${formatWhole(total)}`;
}
