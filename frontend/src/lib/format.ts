/**
 * How dates, times and money look on screen (FR-021, design system §4, research R18).
 * Until feature 003 adds personal preferences, everyone sees these formats.
 */

/** A fixed list, because browsers' own "en-GB" short months now print "Sept". */
const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];

/** Shown instead of a missing amount. Never 0, which would look like a real value. */
const NO_AMOUNT = '—';

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
 * "12 Mar 2026" for the calendar date "2026-03-12".
 * The text is split by hand: new Date("2026-03-12") means midnight in London, which is still the
 * day before in the Americas.
 */
export function formatDate(date: string): string {
  const [year, month, day] = date.split('-');
  return `${day} ${MONTHS[Number(month) - 1]} ${year}`;
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
export function formatDateTime(timestamp: string): string {
  return `${formatDate(localDate(timestamp))}, ${formatTime(timestamp)}`;
}

/** "$12,500.00" for US dollars, "12,500.00 EUR" for other currencies, "—" without an amount. */
export function formatMoney(value: string | null | undefined, currency: string): string {
  if (!value) {
    return NO_AMOUNT;
  }
  const exact = value as Intl.StringNumericLiteral;
  return currency === 'USD' ? dollars.format(exact) : `${amount.format(exact)} ${currency}`;
}

/**
 * "$1.25M" or "12.5K EUR" for summary cards, which show formatMoney's full value in a tooltip.
 * Amounts under 1,000 are short already and are shown in full.
 */
export function formatMoneyCompact(value: string | null | undefined, currency: string): string {
  if (!value || Math.abs(Number(value)) < 1000) {
    return formatMoney(value, currency);
  }
  const exact = value as Intl.StringNumericLiteral;
  return currency === 'USD'
    ? compactDollars.format(exact)
    : `${compactAmount.format(exact)} ${currency}`;
}

function twoDigits(n: number): string {
  return String(n).padStart(2, '0');
}
