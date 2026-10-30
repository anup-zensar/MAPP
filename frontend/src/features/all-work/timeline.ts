import type { AllWorkRow, AllWorkStatus } from './all-work-types';
import { monthDays, quarterDays, yearDays } from './calendar';
import { statusWord } from './status-words';

const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
const QUARTERS = ['Q1', 'Q2', 'Q3', 'Q4'];

export const LEVEL_WORDS: Partial<Record<AllWorkRow['level'], string>> = {
  campaign: 'Campaign',
  initiative: 'Initiative',
  activity: 'Activity',
};

/** The statuses the bars show, each word once, for the legend (FR-025, UI audit X-14). */
export function legendOf(rows: AllWorkRow[]): AllWorkStatus[] {
  const seen = new Map<string, AllWorkStatus>();
  for (const row of rows) {
    if (row.status && LEVEL_WORDS[row.level]) {
      const label = statusWord(row.status.label);
      if (!seen.has(label)) {
        seen.set(label, { ...row.status, label });
      }
    }
  }
  return [...seen.values()];
}

/** The columns of the timeline: months for one year; the years with their quarters for a range. */
export interface Timeline {
  /** The CSS grid columns of the months or quarters, each as wide as its days. */
  columns: string;
  labels: string[];
  years?: string;
}

export function timelineOf(year: number, yearTo: number): Timeline {
  const widths = (days: number[]) => days.map((count) => `minmax(0, ${count}fr)`).join(' ');
  if (yearTo === year) {
    return { columns: widths(monthDays(year)), labels: MONTHS };
  }
  const span = Array.from({ length: yearTo - year + 1 }, (_, index) => year + index);
  return {
    columns: widths(span.flatMap(quarterDays)),
    labels: span.flatMap(() => QUARTERS),
    years: widths(yearDays(year, yearTo)),
  };
}
