/** How dates, times and money look on screen (design system §4), in the person's date format. */
import type { components } from '@/api/generated/schema';
import { exact } from './decimal';

export type DateFormat = components['schemas']['DateFormat'];

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

export const DATE_FORMATS: { value: DateFormat; example: string; label: string }[] = (
  ['DD MMM YYYY', 'DD/MM/YYYY', 'MM/DD/YYYY', 'YYYY-MM-DD'] as const
).map((value) => {
  const example = formatDate('2026-03-12', value);
  return { value, example, label: `${example} (${value})` };
});

/** Shown instead of a missing amount or date. Never 0, which would look like a real value. */
const NOTHING = '—';

/** Intl.NumberFormat formats decimal strings exactly (constitution VI); TS needs the casts. */
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

/** Split by hand: new Date("2026-03-12") is London midnight, the day before in the Americas. */
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

/** A date typed in the person's format as "YYYY-MM-DD"; null when it is not a real date. */
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

/** "01 Feb – 30 Nov 2026" in one year, both dates in full across years; "—" for a missing date. */
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

export function localDate(timestamp: string | Date): string {
  const at = new Date(timestamp);
  return `${at.getFullYear()}-${twoDigits(at.getMonth() + 1)}-${twoDigits(at.getDate())}`;
}

export function formatTime(timestamp: string): string {
  const at = new Date(timestamp);
  return `${twoDigits(at.getHours())}:${twoDigits(at.getMinutes())}`;
}

export function formatDateTime(
  timestamp: string,
  format: DateFormat = DEFAULT_DATE_FORMAT,
): string {
  return `${formatDate(localDate(timestamp), format)}, ${formatTime(timestamp)}`;
}

const WEEKDAYS = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];

/** "Monday 12 Oct 2026"; the weekday is worked out in UTC so the time zone cannot move it. */
export function formatWeekdayDate(date: string, format: DateFormat = DEFAULT_DATE_FORMAT): string {
  const weekday = WEEKDAYS[new Date(`${date}T00:00:00Z`).getUTCDay()];
  return `${weekday} ${formatDate(date, format)}`;
}

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

/** "just now", "5 hours ago", "yesterday", "3 days ago" by calendar day, from 7 days the date. */
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

export function formatMoney(value: string | null | undefined, currency: string): string {
  if (!value) {
    return NOTHING;
  }
  return currency === 'USD'
    ? dollars.format(exact(value))
    : `${amount.format(exact(value))} ${currency}`;
}

/** "$1.25M"; amounts under 1,000, or that would round up to "1000T", are shown in full. */
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

function monthOfName(name: string): number {
  const lower = name.toLowerCase();
  return MONTH_NAMES.findIndex((full) => full.startsWith(lower)) + 1;
}

function daysInMonth(year: number, month: number): number {
  // Day 0 of the next month is the last day of this one; UTC keeps time zones out of it.
  return new Date(Date.UTC(year, month, 0)).getUTCDate();
}

function calendarDaysBetween(from: string, to: string): number {
  return Math.round((Date.parse(`${to}T00:00:00Z`) - Date.parse(`${from}T00:00:00Z`)) / 86_400_000);
}
