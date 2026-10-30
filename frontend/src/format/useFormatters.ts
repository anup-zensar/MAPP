import { useSession } from '@/session/useSession';
import {
  DEFAULT_DATE_FORMAT,
  formatDate,
  formatDateRange,
  formatDateTime,
  formatMoney,
  formatMoneyCompact,
  formatMonth,
  formatRelativeTime,
  formatWeekdayDate,
  parseDate,
} from './intl';
import type { DateFormat } from './intl';

interface MoneyValue {
  amount: string;
  currency: string;
}

export interface Formatters {
  dateFormat: DateFormat;
  /** The person's display currency ("USD" when they have not chosen one, D-191). */
  displayCurrency: string;
  date: (date: string) => string;
  weekdayDate: (date: string) => string;
  month: (month: string) => string;
  dateTime: (timestamp: string) => string;
  dateRange: (start: string | null | undefined, end: string | null | undefined) => string;
  relativeTime: (timestamp: string) => string;
  parseDate: (text: string) => string | null;
  /** Exactly as the API answered, already in the display currency where it should be (003 P6). */
  money: (value: MoneyValue | null | undefined) => string;
  moneyCompact: (value: MoneyValue | null | undefined) => string;
}

/** Dates and money for the signed-in person: every page shows and reads dates only through it. */
export function useFormatters(): Formatters {
  const { user } = useSession();
  const dateFormat = user?.preferences.dateFormat ?? DEFAULT_DATE_FORMAT;
  const displayCurrency = user?.preferences.displayCurrency ?? 'USD';

  return {
    dateFormat,
    displayCurrency,
    date: (date) => formatDate(date, dateFormat),
    weekdayDate: (date) => formatWeekdayDate(date, dateFormat),
    month: (month) => formatMonth(month, dateFormat),
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
