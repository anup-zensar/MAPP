import { describe, expect, it } from 'vitest';
import {
  formatDate,
  formatDateTime,
  formatMoney,
  formatMoneyCompact,
  formatTime,
  localDate,
} from './format';

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
});
