/** The words of the activity log (011), shared by the log page, the History sections and tests. */
import type { components } from '@/api/generated/schema';
import type { StatusTone } from '@/ui/StatusChip/StatusChip';
import { formatWhole } from '@/format/decimal';
import { formatWeekdayDate } from '@/format/intl';
import type { DateFormat } from '@/format/intl';
import { isPlanningType, recordHref } from '@/ui/record-links';
import type { PlanningRecordType } from '@/ui/record-links';
import { plural } from '@/format/text';

export type ActivityLogEntry = components['schemas']['ActivityLogEntry'];
export type LogAction = ActivityLogEntry['action'];
export type RecordRef = components['schemas']['RecordRef'];

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

const PLANNING_LABELS: Record<PlanningRecordType, string> = {
  plan: 'plan',
  campaign: 'campaign',
  initiative: 'initiative',
  activity: 'activity',
  lineItem: 'line item',
};

export function capitalise(text: string): string {
  return text.charAt(0).toUpperCase() + text.slice(1);
}

export function planningLabel(type: string): string {
  return isPlanningType(type) ? PLANNING_LABELS[type] : type;
}

export function initialsOf(name: string): string {
  const [first, ...others] = name.trim().split(/\s+/).filter(Boolean);
  if (first === undefined) {
    return '?';
  }
  const last = others.at(-1) ?? '';
  return (first.charAt(0) + last.charAt(0)).toUpperCase();
}

/** "Priya Raman changed activity", before the name; an emptied bin gets the whole sentence. */
export function sentenceLead(entry: ActivityLogEntry): string {
  if (entry.action === 'emptiedRecycleBin') {
    const count = entry.items?.length ?? 0;
    return `${entry.actor.name} emptied the recycle bin (${plural(count, 'item')})`;
  }
  return `${entry.actor.name} ${entry.verb} ${entry.recordTypeLabel}`;
}

export function entrySentence(entry: ActivityLogEntry): string {
  if (entry.action === 'emptiedRecycleBin') {
    return sentenceLead(entry);
  }
  return `${sentenceLead(entry)} '${entry.recordName}'`;
}

export function pathText(path: RecordRef[]): string {
  return path.length > 0 ? `in ${path.map((record) => record.name).join(' › ')}` : '';
}

export type RecordLink =
  | { kind: 'record'; href: string; now: string }
  | { kind: 'recycleBin'; href: string }
  | { kind: 'text'; suffix: string };

/** Where the name leads: the record's page, the bin with its row highlighted, or text. */
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

function renamed(entry: ActivityLogEntry): string {
  const name = entry.recordNow?.name ?? '';
  return name && name !== entry.recordName ? ` — now '${name}'` : '';
}

const OTHER_PAGES = new Map<string, (id: string) => string>([
  ['globalBudget', (id) => `/budgets/${encodeURIComponent(id)}`],
  ['financialTransaction', (id) => `/financials/transactions/${encodeURIComponent(id)}`],
]);

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

/** **Show details**: "<label>: <old> → <new>", the comment, action details, bin items. */
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

/** "Today", "Yesterday" or "Monday 12 Oct 2026" in the person's date format. */
export function dayLabel(date: string, today: string, format?: DateFormat): string {
  if (date === today) {
    return 'Today';
  }
  if (date === dayBefore(today)) {
    return 'Yesterday';
  }
  return formatWeekdayDate(date, format);
}

/** The calendar day before "2026-03-01": "2026-02-28" (worked out in UTC, so no time zone moves it). */
function dayBefore(date: string): string {
  const day = new Date(`${date}T00:00:00Z`);
  day.setUTCDate(day.getUTCDate() - 1);
  return day.toISOString().slice(0, 10);
}

export function entriesText(count: number): string {
  return plural(count, 'entry', 'entries');
}

/** "12 of 1,204 entries", also when no filter is in use (design system §11, rule 14). */
export function countLine(total: number, unfilteredTotal: number): string {
  return `${formatWhole(total)} of ${entriesText(unfilteredTotal)}`;
}

export function loadedLine(loaded: number, total: number): string {
  return `Showing the last ${formatWhole(loaded)} of ${formatWhole(total)}`;
}
