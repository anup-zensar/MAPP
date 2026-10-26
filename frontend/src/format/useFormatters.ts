import { useSession } from '@/session/useSession';
import {
  DEFAULT_DATE_FORMAT,
  formatDate,
  formatDateRange,
  formatDateTime,
  formatMoney,
  formatMoneyCompact,
  formatRelativeTime,
  parseDate,
} from './intl';
import type { DateFormat } from './intl';

/** An amount as the API answers it (contract Money). */
interface MoneyValue {
  amount: string;
  currency: string;
}

/** The formatting functions of useFormatters(), bound to the signed-in person's preferences. */
export interface Formatters {
  /** The person's date format ("DD MMM YYYY" when they have not chosen one). */
  dateFormat: DateFormat;
  /** The person's display currency ("USD" when they have not chosen one, D-191). */
  displayCurrency: string;
  /** "12 Mar 2026" (or the person's format) for a calendar date "YYYY-MM-DD". */
  date: (date: string) => string;
  /** "12 Mar 2026, 14:32" for a timestamp, in the person's time zone. */
  dateTime: (timestamp: string) => string;
  /** "01 Feb – 30 Nov 2026"; "—" when a date is missing. */
  dateRange: (start: string | null | undefined, end: string | null | undefined) => string;
  /** "just now", "3 days ago", … then the date. */
  relativeTime: (timestamp: string) => string;
  /** Reads a date typed in the person's format: "YYYY-MM-DD", or null when it is not one. */
  parseDate: (text: string) => string | null;
  /**
   * "$12,500.00" / "12,500.00 EUR" for an amount exactly as the API answered it — the API has
   * already converted it to the display currency where it should (feature 003 P6); "—" for none.
   */
  money: (value: MoneyValue | null | undefined) => string;
  /** "$1.25M" for summary cards (with money() as the full value). */
  moneyCompact: (value: MoneyValue | null | undefined) => string;
}

/**
 * Dates and money for the signed-in person (feature 003, FR-023, FR-024): every page of 003 and
 * later features shows and reads dates only through this hook. Without a signed-in person it uses
 * the defaults.
 */
export function useFormatters(): Formatters {
  const { user } = useSession();
  const dateFormat = user?.preferences.dateFormat ?? DEFAULT_DATE_FORMAT;
  const displayCurrency = user?.preferences.displayCurrency ?? 'USD';

  return {
    dateFormat,
    displayCurrency,
    date: (date) => formatDate(date, dateFormat),
    dateTime: (timestamp) => formatDateTime(timestamp, dateFormat),
    dateRange: (start, end) => formatDateRange(start, end, dateFormat),
    relativeTime: (timestamp) => formatRelativeTime(timestamp, new Date(), dateFormat),
    parseDate: (text) => parseDate(text, dateFormat),
    money: (value) =>
      value ? formatMoney(value.amount, value.currency) : formatMoney(null, 'USD'),
    moneyCompact: (value) =>
      value ? formatMoneyCompact(value.amount, value.currency) : formatMoney(null, 'USD'),
  };
}
