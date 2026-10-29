import { LoaderCircle } from 'lucide-react';
import { Button } from '@/ui/Button/Button';
import { localDate } from '@/format/intl';
import { useFormatters } from '@/format/useFormatters';
import { LogEntry } from './LogEntry';
import { dayLabel, entriesText, loadedLine } from './log-text';
import type { ActivityLogEntry } from './log-text';
import styles from './LogEntryList.module.css';

const HEADINGS = { 2: 'h2', 3: 'h3', 4: 'h4' } as const;

export interface LogEntryListProps {
  /** The entries loaded so far, newest first. */
  entries: ActivityLogEntry[];
  /** How many entries match in all. */
  total: number;
  hasMore: boolean;
  onLoadMore: () => void;
  isLoadingMore: boolean;
  /**
   * The level of the day headings: 2 on the log page, 3 inside a History card, 4 inside the History
   * part of a side panel (whose own heading is an h3).
   */
  headingLevel?: 2 | 3 | 4;
}

/**
 * The entries grouped by day in the person's time zone — "Today · 3 entries", "Yesterday",
 * "Monday 12 Oct 2026" (FR-004) — and, while more can be loaded, "Showing the last 50 of 1,204"
 * with **Load more** (FR-009).
 */
export function LogEntryList({
  entries,
  total,
  hasMore,
  onLoadMore,
  isLoadingMore,
  headingLevel = 2,
}: LogEntryListProps) {
  const today = localDate(new Date());
  const { dateFormat } = useFormatters();
  const Heading = HEADINGS[headingLevel];

  return (
    <div className={styles.list}>
      {groupByDay(entries).map(([day, dayEntries]) => (
        <section key={day} className={styles.day}>
          <Heading className={styles.heading}>
            {dayLabel(day, today, dateFormat)}
            {/* The space stays outside the span, so screen readers keep it between the words. */}{' '}
            <span className={styles.count}>· {entriesText(dayEntries.length)}</span>
          </Heading>
          <ol role="list" className={styles.entries}>
            {dayEntries.map((entry) => (
              <LogEntry key={entry.id} entry={entry} />
            ))}
          </ol>
        </section>
      ))}
      {hasMore && (
        <div className={styles.more}>
          <p className={styles.loaded}>{loadedLine(entries.length, total)}</p>
          <Button
            onClick={onLoadMore}
            aria-busy={isLoadingMore || undefined}
            icon={isLoadingMore ? <LoaderCircle className={styles.spinner} /> : undefined}
          >
            Load more
          </Button>
        </div>
      )}
    </div>
  );
}

/** The entries by local calendar day, newest day first, in their order within each day. */
function groupByDay(entries: ActivityLogEntry[]): [string, ActivityLogEntry[]][] {
  const days = new Map<string, ActivityLogEntry[]>();
  for (const entry of entries) {
    const day = localDate(entry.at);
    days.set(day, [...(days.get(day) ?? []), entry]);
  }
  return [...days];
}
