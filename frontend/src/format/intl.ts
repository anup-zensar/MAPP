/**
 * How dates, times and money look on screen (FR-021, design system §4, research R18), in the date
 * format a person chose (feature 003, FR-024). Pages read the person's format through
 * useFormatters(); these functions default to "DD MMM YYYY" ("12 Mar 2026").
 */
import type { components } from '@/api/generated/schema';
import { exact } from './decimal';

/** How dates are shown and typed (feature 003, contract DateFormat). */
export type DateFormat = components['schemas']['DateFormat'];

/** The default date format: "12 Mar 2026". */
export const DEFAULT_DATE_FORMAT: DateFormat = 'DD MMM YYYY';

/** A fixed list, because browsers' own "en-GB" short months now print "Sept". */
const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
const MONTH_NAMES = [
  'january',
  'february',
  'march',
  'april',
  'may',
  'june',
  'july',
  'august',
  'september',
  'october',
  'november',
  'december',
];

/** Every date format with 12 March 2026 written in it, as the profile page offers them. */
export const DATE_FORMATS: { value: DateFormat; example: string; label: string }[] = (
  ['DD MMM YYYY', 'DD/MM/YYYY', 'MM/DD/YYYY', 'YYYY-MM-DD'] as const
).map((value) => {
  const example = formatDate('2026-03-12', value);
  return { value, example, label: `${example} (${value})` };
});

/** Shown instead of a missing amount or date. Never 0, which would look like a real value. */
const NOTHING = '—';

/**
 * Money arrives as a decimal string ("12500.00"). Intl.NumberFormat formats such a string exactly,
 * so no floating-point rounding happens (constitution VI). TypeScript only lets it take strings
 * that look like numbers, hence the casts below.
 */
const dollars = new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' });
const amount = new Intl.NumberFormat('en-US', {
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
});
const compactDollars = new Intl.NumberFormat('en-US', {
  style: 'currency',
  currency: 'USD',
  notation: 'compact',
  maximumFractionDigits: 2,
});
const compactAmount = new Intl.NumberFormat('en-US', {
  notation: 'compact',
  maximumFractionDigits: 2,
});

/**
 * The calendar date "2026-03-12" as "12 Mar 2026" (or in the given format: "12/03/2026",
 * "03/12/2026", "2026-03-12"). The text is split by hand: new Date("2026-03-12") means midnight in
 * London, which is still the day before in the Americas.
 */
export function formatDate(date: string, format: DateFormat = DEFAULT_DATE_FORMAT): string {
  const [year, month, day] = date.split('-');
  switch (format) {
    case 'DD/MM/YYYY':
      return `${day}/${month}/${year}`;
    case 'MM/DD/YYYY':
      return `${month}/${day}/${year}`;
    case 'YYYY-MM-DD':
      return `${year}-${month}-${day}`;
    default:
      return `${day} ${MONTHS[Number(month) - 1]} ${year}`;
  }
}

/**
 * Reads a date typed in the person's format and returns it as "YYYY-MM-DD", or null when it is not
 * a real calendar date. Days and months may have one or two digits; in "DD MMM YYYY" the month may
 * be written short or in full, in any case ("12 mar 2026", "12 March 2026").
 */
export function parseDate(text: string, format: DateFormat = DEFAULT_DATE_FORMAT): string | null {
  const typed = text.trim();
  let parts: { day: string; month: number; year: string } | null;
  if (format === 'DD MMM YYYY') {
    const [, day = '', name = '', year = ''] =
      /^(\d{1,2})\s+([a-z]{3,})\.?\s+(\d{4})$/i.exec(typed) ?? [];
    const month = monthOfName(name);
    parts = day && month ? { day, month, year } : null;
  } else if (format === 'YYYY-MM-DD') {
    const [, year = '', month = '', day = ''] = /^(\d{4})-(\d{1,2})-(\d{1,2})$/.exec(typed) ?? [];
    parts = year ? { day, month: Number(month), year } : null;
  } else {
    const [, first = '', second = '', year = ''] =
      /^(\d{1,2})[/.-](\d{1,2})[/.-](\d{4})$/.exec(typed) ?? [];
    const [day, month] = format === 'DD/MM/YYYY' ? [first, second] : [second, first];
    parts = year ? { day, month: Number(month), year } : null;
  }
  if (!parts) {
    return null;
  }
  const day = Number(parts.day);
  const year = Number(parts.year);
  if (parts.month < 1 || parts.month > 12 || day < 1 || day > daysInMonth(year, parts.month)) {
    return null;
  }
  return `${parts.year}-${twoDigits(parts.month)}-${twoDigits(day)}`;
}

/**
 * "01 Feb – 30 Nov 2026" within one year, "01 Dec 2025 – 31 Jan 2026" across years, "—" when a
 * date is missing (feature 005, the server's FormatRange). Other formats write both dates in full.
 */
export function formatDateRange(
  start: string | null | undefined,
  end: string | null | undefined,
  format: DateFormat = DEFAULT_DATE_FORMAT,
): string {
  if (!start || !end) {
    return NOTHING;
  }
  if (format === 'DD MMM YYYY' && start.slice(0, 4) === end.slice(0, 4)) {
    const [, month, day] = start.split('-');
    return `${day} ${MONTHS[Number(month) - 1]} – ${formatDate(end, format)}`;
  }
  return `${formatDate(start, format)} – ${formatDate(end, format)}`;
}

/** The calendar date ("2026-03-12") on which a timestamp falls in the user's own time zone. */
export function localDate(timestamp: string | Date): string {
  const at = new Date(timestamp);
  return `${at.getFullYear()}-${twoDigits(at.getMonth() + 1)}-${twoDigits(at.getDate())}`;
}

/** "14:32": the time of a timestamp in the user's own time zone, on a 24-hour clock. */
export function formatTime(timestamp: string): string {
  const at = new Date(timestamp);
  return `${twoDigits(at.getHours())}:${twoDigits(at.getMinutes())}`;
}

/** "12 Mar 2026, 14:32": a timestamp in the user's own time zone. */
export function formatDateTime(
  timestamp: string,
  format: DateFormat = DEFAULT_DATE_FORMAT,
): string {
  return `${formatDate(localDate(timestamp), format)}, ${formatTime(timestamp)}`;
}

const WEEKDAYS = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];

/**
 * The calendar date "2026-10-12" as "Monday 12 Oct 2026", or with the date in the given format
 * ("Monday 2026-10-12"; the activity log's day headings, 038 FR-058). The weekday is worked out in
 * UTC, so the time zone cannot move it to another day.
 */
export function formatWeekdayDate(date: string, format: DateFormat = DEFAULT_DATE_FORMAT): string {
  const weekday = WEEKDAYS[new Date(`${date}T00:00:00Z`).getUTCDay()];
  return `${weekday} ${formatDate(date, format)}`;
}

/**
 * The month "2026-09" (or a date in it) as "Sep 2026", or in the given format: "09/2026",
 * "2026-09" (038 FR-058, D-356). Every month label with a year is written by this function.
 */
export function formatMonth(month: string, format: DateFormat = DEFAULT_DATE_FORMAT): string {
  const [year = '', number = ''] = month.split('-');
  switch (format) {
    case 'DD/MM/YYYY':
    case 'MM/DD/YYYY':
      return `${number}/${year}`;
    case 'YYYY-MM-DD':
      return `${year}-${number}`;
    default:
      return `${MONTHS[Number(number) - 1]} ${year}`;
  }
}

/**
 * How long ago something happened (feature 008, notifications): "just now", "1 minute ago",
 * "5 hours ago", "yesterday", "3 days ago" (by calendar day, in the user's time zone), and from
 * 7 days on the date itself.
 */
export function formatRelativeTime(
  timestamp: string,
  now: Date = new Date(),
  format: DateFormat = DEFAULT_DATE_FORMAT,
): string {
  const minutes = Math.floor((now.getTime() - new Date(timestamp).getTime()) / 60_000);
  if (minutes < 1) {
    return 'just now';
  }
  if (minutes < 60) {
    return minutes === 1 ? '1 minute ago' : `${minutes} minutes ago`;
  }
  if (minutes < 24 * 60) {
    const hours = Math.floor(minutes / 60);
    return hours === 1 ? '1 hour ago' : `${hours} hours ago`;
  }
  const day = localDate(timestamp);
  const days = calendarDaysBetween(day, localDate(now));
  if (days <= 1) {
    return 'yesterday';
  }
  if (days < 7) {
    return `${days} days ago`;
  }
  return formatDate(day, format);
}

/**
 * How long something has been waiting (feature 008, approvals): "less than an hour", "1 hour",
 * "5 hours", "1 day", "3 days".
 */
export function formatWaitingTime(since: string, now: Date = new Date()): string {
  const hours = Math.floor((now.getTime() - new Date(since).getTime()) / 3_600_000);
  if (hours < 1) {
    return 'less than an hour';
  }
  if (hours < 24) {
    return hours === 1 ? '1 hour' : `${hours} hours`;
  }
  const days = Math.floor(hours / 24);
  return days === 1 ? '1 day' : `${days} days`;
}

/** "$12,500.00" for US dollars, "12,500.00 EUR" for other currencies, "—" without an amount. */
export function formatMoney(value: string | null | undefined, currency: string): string {
  if (!value) {
    return NOTHING;
  }
  return currency === 'USD'
    ? dollars.format(exact(value))
    : `${amount.format(exact(value))} ${currency}`;
}

/**
 * "$1.25M" or "12.5K EUR" for summary cards, which show formatMoney's full value in a tooltip.
 * Amounts under 1,000 are short already and are shown in full, and so are amounts that would round
 * up to 1000 of the largest unit ("1000T", 038 FR-044).
 */
export function formatMoneyCompact(value: string | null | undefined, currency: string): string {
  if (!value || Math.abs(Number(value)) < 1000) {
    return formatMoney(value, currency);
  }
  const format = currency === 'USD' ? compactDollars : compactAmount;
  const parts = format.formatToParts(exact(value));
  const integer = parts.filter((part) => part.type === 'integer').map((part) => part.value);
  if (integer.join('').length > 3) {
    return formatMoney(value, currency);
  }
  const compact = parts.map((part) => part.value).join('');
  return currency === 'USD' ? compact : `${compact} ${currency}`;
}

function twoDigits(n: number): string {
  return String(n).padStart(2, '0');
}

/** 1–12 for a month name written short or in full ("mar", "March", "Sept"); 0 for anything else. */
function monthOfName(name: string): number {
  const lower = name.toLowerCase();
  return MONTH_NAMES.findIndex((full) => full.startsWith(lower)) + 1;
}

function daysInMonth(year: number, month: number): number {
  // Day 0 of the next month is the last day of this one; UTC keeps time zones out of it.
  return new Date(Date.UTC(year, month, 0)).getUTCDate();
}

/** Whole calendar days from one "YYYY-MM-DD" to another. */
function calendarDaysBetween(from: string, to: string): number {
  return Math.round((Date.parse(`${to}T00:00:00Z`) - Date.parse(`${from}T00:00:00Z`)) / 86_400_000);
}
