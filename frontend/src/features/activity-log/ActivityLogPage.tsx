import { History, SearchX } from 'lucide-react';
import { Link, useSearchParams } from 'react-router';
import { Button } from '@/ui/Button/Button';
import { EmptyState } from '@/ui/EmptyState/EmptyState';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { SummaryCard } from '@/ui/SummaryCard/SummaryCard';
import { formatWhole } from '@/format/decimal';
import { usePageTitle } from '@/ui/usePageTitle';
import { LOG_PAGE_SIZE, logEntriesKey, useLogEntries, useLogSummary } from './api';
import { LogEntryList } from './LogEntryList';
import { LogFilterBar } from './LogFilterBar';
import {
  NO_FILTERS,
  apiQuery,
  filtersFromParams,
  filtersToParams,
  isFiltered,
  todayStart,
  weekStart,
} from './log-filters';
import type { LogFilterState } from './log-filters';
import { countLine } from './log-text';
import styles from './ActivityLogPage.module.css';

/** The four summary cards (FR-002): label, tip and the filters a click applies. */
const CARDS = [
  {
    key: 'changesToday',
    label: 'Changes today',
    tip: 'Entries since midnight, in your time zone.',
    filters: '?range=today',
  },
  {
    key: 'changesThisWeek',
    label: 'Changes this week',
    tip: 'Entries since Monday.',
    filters: '?range=week',
  },
  {
    key: 'peopleActiveThisWeek',
    label: 'People active this week',
    tip: 'Different people who changed something since Monday.',
    filters: '?range=week',
  },
  {
    key: 'deletionsThisWeek',
    label: 'Deletions this week',
    tip: 'Deletes, permanent deletes and emptied recycle bins since Monday.',
    filters: '?range=week&action=deleted&action=deletedPermanently',
  },
] as const;

/**
 * The Activity log (feature 011, US2, FR-001 – FR-010): every change, newest first, grouped by
 * day, with four summary cards and filters kept in the address. Read-only: there is nothing to
 * create, edit or remove here.
 */
export function ActivityLogPage() {
  usePageTitle('Activity log');
  const [params, setParams] = useSearchParams();
  const filters = filtersFromParams(params);
  const now = new Date();
  const summary = useLogSummary(todayStart(now), weekStart(now));
  const query = apiQuery(filters, now);
  const log = useLogEntries(logEntriesKey(query), query, LOG_PAGE_SIZE);
  const filtered = isFiltered(filters);

  const changeFilters = (next: LogFilterState, typing = false) =>
    setParams(filtersToParams(next), { replace: typing });

  return (
    <>
      <PageHeader
        title="Activity log"
        description="See who changed what, and when. Entries can't be edited or removed."
      />
      <div className={styles.sections}>
        {summary.error ? (
          <ErrorState error={summary.error} onRetry={() => void summary.refetch()} />
        ) : (
          <div className={styles.cards}>
            {CARDS.map((card) =>
              summary.data ? (
                <SummaryCard
                  key={card.key}
                  label={card.label}
                  value={formatWhole(summary.data[card.key])}
                  tip={card.tip}
                  detail={
                    <Link
                      className={styles.cardLink}
                      to={card.filters}
                      aria-label={`${card.label}: ${formatWhole(summary.data[card.key])} — show these entries`}
                    >
                      Show these entries
                    </Link>
                  }
                />
              ) : (
                <div key={card.key} className={styles.cardSkeleton}>
                  <Skeleton lines={2} />
                </div>
              ),
            )}
          </div>
        )}

        {/* The list template (design system §2.4 A): the toolbar inside the card, across its width. */}
        <section className={styles.panel} aria-label="Entries">
          <div className={styles.toolbar}>
            <LogFilterBar
              filters={filters}
              onChange={changeFilters}
              summary={summary.data}
              record={log.firstPage?.record ?? null}
              count={<LogCount log={log} filtered={filtered} />}
            />
          </div>
          <div className={styles.body}>
            <LogBody log={log} filtered={filtered} onClear={() => changeFilters(NO_FILTERS)} />
          </div>
        </section>
      </div>
    </>
  );
}

/** "12 of 1,204 entries" at the end of the toolbar, once the first page has arrived (§5.2). */
function LogCount({ log, filtered }: { log: ReturnType<typeof useLogEntries>; filtered: boolean }) {
  const page = log.firstPage;
  const unfiltered = page?.unfilteredTotal ?? page?.total ?? 0;
  if (!page || log.error || unfiltered === 0 || (!filtered && page.total === 0)) {
    return null;
  }
  return (
    <p className={styles.count} aria-live="polite">
      {countLine(page.total, unfiltered)}
    </p>
  );
}

/** The list, or the state the list is in. */
function LogBody({
  log,
  filtered,
  onClear,
}: {
  log: ReturnType<typeof useLogEntries>;
  filtered: boolean;
  onClear: () => void;
}) {
  if (log.error) {
    return <ErrorState error={log.error} onRetry={log.refetch} />;
  }
  if (log.isPending) {
    return (
      <div className={styles.loading}>
        {Array.from({ length: 5 }, (_, index) => (
          <Skeleton key={index} lines={2} />
        ))}
      </div>
    );
  }
  const page = log.firstPage;
  if (!page) {
    // A custom range that ends before it starts: its field says why, and nothing is asked.
    return null;
  }
  const unfiltered = page.unfilteredTotal ?? page.total;
  if (unfiltered === 0 || (!filtered && page.total === 0)) {
    return (
      <EmptyState
        icon={<History />}
        title="No changes yet"
        text="Changes to plans, settings and people appear here."
      />
    );
  }

  return page.total === 0 ? (
    <EmptyState
      icon={<SearchX />}
      title="Nothing matches these filters"
      text="Remove a filter or search for something else."
      action={<Button onClick={onClear}>Clear filters</Button>}
    />
  ) : (
    <LogEntryList
      entries={log.entries}
      total={page.total}
      hasMore={log.hasMore}
      onLoadMore={log.loadMore}
      isLoadingMore={log.isLoadingMore}
    />
  );
}
