/**
 * Where things go on the Calendar (US5, FR-025 – FR-028), as pure functions of calendar dates
 * ("YYYY-MM-DD", read as days, never as local times): bars on the year's timeline, the Today line,
 * and the weeks of one month.
 */
import type { AllWorkRow } from './allWorkTypes';

const DAY_MS = 86_400_000;

/** A date as a whole day number, the same in every time zone. */
function dayNumber(date: string): number {
  return Date.parse(`${date}T00:00:00Z`) / DAY_MS;
}

function dateOf(day: number): string {
  return new Date(day * DAY_MS).toISOString().slice(0, 10);
}

/** The days from 1 January of `from` to 31 December of `to` (one year, or a range of up to 5). */
function rangeSpan(from: number, to: number = from): { first: number; last: number; days: number } {
  const first = dayNumber(`${from}-01-01`);
  const last = dayNumber(`${to}-12-31`);
  return { first, last, days: last - first + 1 };
}

/** The number of days of each month of a year (for month columns as wide as their days). */
export function monthDays(year: number): number[] {
  return Array.from({ length: 12 }, (_, month) =>
    new Date(Date.UTC(year, month + 1, 0)).getUTCDate(),
  );
}

/**
 * The days of each year of a range, for the year bands of a timeline over several years (016
 * FR-036).
 */
export function yearDays(from: number, to: number): number[] {
  return Array.from({ length: to - from + 1 }, (_, index) => {
    const year = from + index;
    return year % 4 === 0 && (year % 100 !== 0 || year % 400 === 0) ? 366 : 365;
  });
}

/** The days of each quarter of a year, for the quarter columns of a timeline over several years. */
export function quarterDays(year: number): number[] {
  const months = monthDays(year);
  return [0, 3, 6, 9].map((first) => months[first] + months[first + 1] + months[first + 2]);
}

/** A record's bar on the timeline, in per cent of the timeline's width. */
export interface Bar {
  left: number;
  width: number;
  /** It starts before the timeline ("Continues from <first year − 1>"). */
  continuesFrom: boolean;
  /** It ends after the timeline ("Continues into <last year + 1>"). */
  continuesInto: boolean;
  /** It has no end: the bar runs to the end of the timeline with an open end ("Ongoing"). */
  ongoing: boolean;
}

/**
 * The bar of a campaign, initiative or activity from its start to its end date, cut at the
 * timeline's edges (FR-025, FR-026): one year, or the years `year` to `yearTo` (016 FR-036). Plans
 * have no bar (D-124); records without dates have none.
 */
export function barOf(
  row: Pick<AllWorkRow, 'level' | 'startDate' | 'endDate' | 'ongoing'>,
  year: number,
  yearTo: number = year,
): Bar | null {
  if (row.level === 'plan' || !row.startDate || (!row.endDate && !row.ongoing)) {
    return null;
  }
  const { first, last, days } = rangeSpan(year, yearTo);
  const start = dayNumber(row.startDate);
  const end = row.ongoing ? last : dayNumber(row.endDate as string);
  if (end < first || start > last) {
    return null;
  }
  const from = Math.max(start, first);
  const to = Math.min(end, last);
  return {
    left: ((from - first) / days) * 100,
    width: ((to - from + 1) / days) * 100,
    continuesFrom: start < first,
    continuesInto: !row.ongoing && end > last,
    ongoing: Boolean(row.ongoing),
  };
}

/** Whether a record belongs in "No dates yet" (FR-026). */
export function hasNoDates(row: Pick<AllWorkRow, 'level' | 'startDate' | 'endDate' | 'ongoing'>) {
  return (
    (row.level === 'campaign' || row.level === 'initiative' || row.level === 'activity') &&
    (!row.startDate || (!row.endDate && !row.ongoing))
  );
}

/**
 * The Today line in per cent of the timeline's width (the middle of the day); null outside the
 * year or range.
 */
export function todayOffset(today: string, year: number, yearTo: number = year): number | null {
  const { first, last, days } = rangeSpan(year, yearTo);
  const day = dayNumber(today);
  if (day < first || day > last) {
    return null;
  }
  return ((day - first + 0.5) / days) * 100;
}

/**
 * The weeks of one month (1–12), Monday to Sunday (FR-028): each day "YYYY-MM-DD", or null for
 * the days of the weeks that belong to the month before or after.
 */
export function monthWeeks(year: number, month: number): (string | null)[][] {
  const first = dayNumber(`${year}-${String(month).padStart(2, '0')}-01`);
  const count = monthDays(year)[month - 1];
  // getUTCDay: Sunday 0 … Saturday 6; Monday first means Monday 0 … Sunday 6.
  const lead = (new Date(first * DAY_MS).getUTCDay() + 6) % 7;
  const cells: (string | null)[] = [
    ...Array.from({ length: lead }, () => null),
    ...Array.from({ length: count }, (_, index) => dateOf(first + index)),
  ];
  while (cells.length % 7 !== 0) {
    cells.push(null);
  }
  return Array.from({ length: cells.length / 7 }, (_, week) => cells.slice(week * 7, week * 7 + 7));
}

/** Whether an activity runs on a day: from its start to its end, or to the year's end when ongoing. */
export function runsOn(
  row: Pick<AllWorkRow, 'startDate' | 'endDate' | 'ongoing'>,
  date: string,
  year: number,
): boolean {
  if (!row.startDate || (!row.endDate && !row.ongoing)) {
    return false;
  }
  const end = row.ongoing ? `${year}-12-31` : (row.endDate as string);
  return row.startDate <= date && date <= end;
}
