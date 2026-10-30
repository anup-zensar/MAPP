import { describe, expect, it } from 'vitest';
import {
  formatDate,
  formatDateRange,
  formatDateTime,
  formatMoney,
  formatMoneyCompact,
  formatRelativeTime,
  formatTime,
  formatWaitingTime,
  formatMonth,
  formatWeekdayDate,
  localDate,
  parseDate,
} from './intl';

// Vitest runs in UTC (vite.config.ts), so "local time" below is UTC unless a test says otherwise.

/** Runs `check` as if the computer were set to another time zone. */
function inTimeZone(timeZone: string, check: () => void): void {
  const original = process.env.TZ;
  process.env.TZ = timeZone;
  try {
    check();
  } finally {
    if (original === undefined) {
      delete process.env.TZ;
    } else {
      process.env.TZ = original;
    }
  }
}

describe('formatDate', () => {
  it('shows a calendar date as DD MMM YYYY (FR-021)', () => {
    expect(formatDate('2026-03-12')).toBe('12 Mar 2026');
    expect(formatDate('2026-03-05')).toBe('05 Mar 2026');
  });

  it('writes September as "Sep", never "Sept" (FR-021)', () => {
    expect(formatDate('2026-09-12')).toBe('12 Sep 2026');
  });

  it('never moves a calendar date to the day before, also west of London (FR-021)', () => {
    inTimeZone('America/New_York', () => {
      expect(formatDate('2026-03-12')).toBe('12 Mar 2026');
    });
  });
});

describe('dates and times of a timestamp', () => {
  it('shows a timestamp as DD MMM YYYY, HH:mm in the user’s time zone (FR-021)', () => {
    expect(formatDateTime('2026-03-12T14:32:00Z')).toBe('12 Mar 2026, 14:32');
  });

  it('shows the time on a 24-hour clock with two-digit hours and minutes (FR-021)', () => {
    expect(formatTime('2026-03-12T09:05:00Z')).toBe('09:05');
    expect(formatTime('2026-03-12T23:59:00Z')).toBe('23:59');
  });

  it('finds the calendar date of a timestamp in the user’s time zone', () => {
    expect(localDate('2026-03-12T23:30:00Z')).toBe('2026-03-12');

    // 23:30 UTC is already 05:00 the next morning in India.
    inTimeZone('Asia/Kolkata', () => {
      expect(localDate('2026-03-12T23:30:00Z')).toBe('2026-03-13');
      expect(formatDateTime('2026-03-12T23:30:00Z')).toBe('13 Mar 2026, 05:00');
    });
  });
});

describe("the person's date format in headings and month labels (038 FR-058, D-362)", () => {
  it.each([
    ['DD MMM YYYY', 'Tuesday 27 Oct 2026', 'Sep 2026'],
    ['DD/MM/YYYY', 'Tuesday 27/10/2026', '09/2026'],
    ['MM/DD/YYYY', 'Tuesday 10/27/2026', '09/2026'],
    ['YYYY-MM-DD', 'Tuesday 2026-10-27', '2026-09'],
  ] as const)('038 FR-058: %s writes "%s" and "%s"', (format, heading, month) => {
    expect(formatWeekdayDate('2026-10-27', format)).toBe(heading);
    expect(formatMonth('2026-09', format)).toBe(month);
  });

  it('038 FR-058: writes September as "Sep", never "Sept"', () => {
    expect(formatMonth('2026-09')).toBe('Sep 2026');
    expect(formatMonth('2027-01')).toBe('Jan 2027');
  });
});

describe('formatWeekdayDate (feature 011)', () => {
  it('names the weekday before the date, for the activity log’s day headings (FR-004)', () => {
    expect(formatWeekdayDate('2026-10-12')).toBe('Monday 12 Oct 2026');
    expect(formatWeekdayDate('2026-10-18')).toBe('Sunday 18 Oct 2026');
  });

  it('keeps the weekday of the calendar date in any time zone (FR-004)', () => {
    inTimeZone('America/New_York', () => {
      expect(formatWeekdayDate('2026-10-12')).toBe('Monday 12 Oct 2026');
    });
  });
});

describe('formatMoney', () => {
  it('shows US dollars with the $ sign and two decimals (FR-021)', () => {
    expect(formatMoney('12500', 'USD')).toBe('$12,500.00');
  });

  it('shows other currencies with their code after the amount (FR-021)', () => {
    expect(formatMoney('12500', 'EUR')).toBe('12,500.00 EUR');
  });

  it('shows a negative amount with a minus sign (FR-021)', () => {
    expect(formatMoney('-12500', 'USD')).toBe('-$12,500.00');
    expect(formatMoney('-12500', 'EUR')).toBe('-12,500.00 EUR');
  });

  it('shows "—" when there is no amount, never 0 (FR-021)', () => {
    expect(formatMoney('', 'USD')).toBe('—');
    expect(formatMoney(null, 'USD')).toBe('—');
    expect(formatMoney(undefined, 'EUR')).toBe('—');
  });

  it('shows zero as a real amount', () => {
    expect(formatMoney('0', 'USD')).toBe('$0.00');
  });

  it('formats the decimal string exactly, without floating-point rounding (constitution VI)', () => {
    // As a JavaScript number this amount would become 12345678901234568.
    expect(formatMoney('12345678901234567.89', 'USD')).toBe('$12,345,678,901,234,567.89');
  });
});

describe('formatMoneyCompact', () => {
  it('shortens large amounts for summary cards (FR-021)', () => {
    expect(formatMoneyCompact('1250000', 'USD')).toBe('$1.25M');
    expect(formatMoneyCompact('12500', 'EUR')).toBe('12.5K EUR');
    expect(formatMoneyCompact('-1250000', 'USD')).toBe('-$1.25M');
  });

  it('shows amounts under 1,000 in full', () => {
    expect(formatMoneyCompact('950', 'USD')).toBe('$950.00');
    expect(formatMoneyCompact('12.5', 'EUR')).toBe('12.50 EUR');
  });

  it('shows "—" when there is no amount (FR-021)', () => {
    expect(formatMoneyCompact('', 'USD')).toBe('—');
  });

  it('038 FR-044: never rounds up to "1000T"; the full amount shows instead', () => {
    expect(formatMoneyCompact('999999999999999.99', 'USD')).toBe('$999,999,999,999,999.99');
    expect(formatMoneyCompact('-999995000000000', 'EUR')).toBe('-999,995,000,000,000.00 EUR');
    expect(formatMoneyCompact('999994999999999', 'USD')).toBe('$999.99T');
  });
});

describe('date formats (feature 003, FR-024)', () => {
  it('writes a date in each of the four formats', () => {
    expect(formatDate('2026-03-12', 'DD MMM YYYY')).toBe('12 Mar 2026');
    expect(formatDate('2026-03-12', 'DD/MM/YYYY')).toBe('12/03/2026');
    expect(formatDate('2026-03-12', 'MM/DD/YYYY')).toBe('03/12/2026');
    expect(formatDate('2026-03-12', 'YYYY-MM-DD')).toBe('2026-03-12');
    expect(formatDateTime('2026-03-12T14:32:00Z', 'DD/MM/YYYY')).toBe('12/03/2026, 14:32');
  });

  it.each([
    ['12 Mar 2026', 'DD MMM YYYY', '2026-03-12'],
    ['1 march 2026', 'DD MMM YYYY', '2026-03-01'],
    ['05 SEP 2026', 'DD MMM YYYY', '2026-09-05'],
    ['5 Sept 2026', 'DD MMM YYYY', '2026-09-05'],
    ['12/03/2026', 'DD/MM/YYYY', '2026-03-12'],
    ['1/3/2026', 'DD/MM/YYYY', '2026-03-01'],
    ['03/12/2026', 'MM/DD/YYYY', '2026-03-12'],
    ['2026-3-1', 'YYYY-MM-DD', '2026-03-01'],
    ['29/02/2028', 'DD/MM/YYYY', '2028-02-29'],
  ] as const)('reads "%s" in %s as %s', (text, format, expected) => {
    expect(parseDate(text, format)).toBe(expected);
  });

  it.each([
    ['31/02/2026', 'DD/MM/YYYY'],
    ['29/02/2026', 'DD/MM/YYYY'],
    ['13/13/2026', 'MM/DD/YYYY'],
    ['12 Ma 2026', 'DD MMM YYYY'],
    ['12 Foo 2026', 'DD MMM YYYY'],
    ['2026-03-12', 'DD/MM/YYYY'],
    ['', 'YYYY-MM-DD'],
  ] as const)('refuses "%s" in %s', (text, format) => {
    expect(parseDate(text, format)).toBeNull();
  });
});

describe('formatDateRange (feature 005)', () => {
  it('writes the year once when both dates are in the same year', () => {
    expect(formatDateRange('2026-02-01', '2026-11-30')).toBe('01 Feb – 30 Nov 2026');
  });

  it('writes both years across a new year', () => {
    expect(formatDateRange('2025-12-01', '2026-01-31')).toBe('01 Dec 2025 – 31 Jan 2026');
  });

  it('writes "—" when a date is missing', () => {
    expect(formatDateRange('2026-02-01', null)).toBe('—');
    expect(formatDateRange(undefined, '2026-02-01')).toBe('—');
  });

  it('writes both dates in full in the other formats', () => {
    expect(formatDateRange('2026-02-01', '2026-11-30', 'DD/MM/YYYY')).toBe(
      '01/02/2026 – 30/11/2026',
    );
  });
});

describe('formatRelativeTime and formatWaitingTime (feature 008)', () => {
  const now = new Date('2026-03-12T12:00:00Z');

  it.each([
    ['2026-03-12T11:59:30Z', 'just now'],
    ['2026-03-12T11:59:00Z', '1 minute ago'],
    ['2026-03-12T11:15:00Z', '45 minutes ago'],
    ['2026-03-12T11:00:00Z', '1 hour ago'],
    ['2026-03-12T01:00:00Z', '11 hours ago'],
    ['2026-03-11T08:00:00Z', 'yesterday'],
    ['2026-03-09T12:00:00Z', '3 days ago'],
    ['2026-03-06T12:00:00Z', '6 days ago'],
    ['2026-03-05T12:00:00Z', '05 Mar 2026'],
  ])('says %s was %s', (timestamp, expected) => {
    expect(formatRelativeTime(timestamp, now)).toBe(expected);
  });

  it.each([
    ['2026-03-12T11:30:00Z', 'less than an hour'],
    ['2026-03-12T11:00:00Z', '1 hour'],
    ['2026-03-12T07:00:00Z', '5 hours'],
    ['2026-03-11T12:00:00Z', '1 day'],
    ['2026-03-09T11:00:00Z', '3 days'],
  ])('says something waiting since %s has waited %s', (since, expected) => {
    expect(formatWaitingTime(since, now)).toBe(expected);
  });
});
