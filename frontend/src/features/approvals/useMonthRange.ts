/** The viewer's local calendar month as two instants (ISO strings), for the summary's counts. */
export interface MonthRange {
  /** The first instant of this month, local time. */
  monthStart: string;
  /** The first instant of the next month, local time. */
  monthEnd: string;
}

/** The local calendar month that `now` falls in (plan, "Server state"). */
export function monthRange(now: Date): MonthRange {
  const start = new Date(now.getFullYear(), now.getMonth(), 1);
  const end = new Date(now.getFullYear(), now.getMonth() + 1, 1);
  return { monthStart: start.toISOString(), monthEnd: end.toISOString() };
}

/**
 * This month for "Approved this month" and "Declined this month" (FR-028): the strings stay the
 * same all month, so the summary's query key changes only when a new month starts.
 */
export function useMonthRange(): MonthRange {
  return monthRange(new Date());
}
