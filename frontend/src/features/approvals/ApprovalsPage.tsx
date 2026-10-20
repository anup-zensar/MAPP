import { CircleCheckBig, SearchX } from 'lucide-react';
import { useState } from 'react';
import { useSearchParams } from 'react-router';
import {
  Button,
  Card,
  CountBadge,
  EmptyState,
  ErrorState,
  FilterChips,
  PageHeader,
  Tabs,
  SearchField,
  Skeleton,
  SummaryCard,
} from '@/components';
import type { TabItem } from '@/components';
import type { components } from '@/api/schema';
import { usePageTitle } from '@/lib/usePageTitle';
import { useFormatters } from '@/lib/useFormatters';
import {
  NO_FILTERS,
  applyFilters,
  filterChips,
  filterOptions,
  filtersInUse,
} from './approvalFilters';
import type { ApprovalFilterKey, ApprovalFilterValues } from './approvalFilters';
import { ApprovalPreviewPanel } from './ApprovalPreviewPanel';
import { useApprovalRows } from './approvalsApi';
import type { ApprovalRow, ApprovalTab } from './approvalsApi';
import { ApprovalsFilters } from './ApprovalsFilters';
import { ApprovalsTable } from './ApprovalsTable';
import { activitiesCount } from './messages';
import { useApprovalSummary } from './useApprovalSummary';
import styles from './ApprovalsPage.module.css';

type ApprovalCounts = components['schemas']['ApprovalCounts'];

const TAB_VALUES: ApprovalTab[] = ['waiting', 'open', 'decided'];

/** What an empty tab says (plan, "User-visible messages"). */
const EMPTY_TAB: Record<ApprovalTab, string> = {
  waiting: "Nothing is waiting for you. You're all caught up.",
  open: 'No activities are waiting for approval.',
  decided: "You haven't approved or declined any activities yet.",
};

/**
 * Approvals (US3, FR-027 – FR-032), at /approvals: the four summary cards, the tabs Waiting for me ·
 * All open · Decided by me (?tab=), search and filters with chips and a count, and the rows; a row
 * opens the preview (?activity=), where the approver decides without leaving the page.
 *
 * Without a chosen tab the page opens the first view that has something in it: Waiting for me
 * when anything waits for the person, otherwise All open (UI audit AP-03), so someone who only
 * sends activities never lands on a view that is always empty.
 */
export function ApprovalsPage() {
  usePageTitle('Approvals');
  const [params, setParams] = useSearchParams();
  const tabParam = params.get('tab') as ApprovalTab | null;
  const chosenTab = tabParam && TAB_VALUES.includes(tabParam) ? tabParam : null;
  const previewId = params.get('activity');
  // Search and filters stay when the tab changes (plan, "Approvals page").
  const [search, setSearch] = useState('');
  const [filters, setFilters] = useState<ApprovalFilterValues>(NO_FILTERS);
  const summary = useApprovalSummary();
  const landingTab = landingTabOf(summary.data, summary.isError);
  const tab: ApprovalTab = chosenTab ?? landingTab ?? 'waiting';
  // The rows wait for the counts when no tab is chosen, so the first view is the right one.
  const rows = useApprovalRows(tab, chosenTab !== null || landingTab !== null);
  const allRows = rows.data?.items ?? [];
  const visible = applyFilters(allRows, search, filters);

  function changeParams(change: Record<string, string | null>, replace = false) {
    const next = new URLSearchParams(params);
    for (const [key, value] of Object.entries(change)) {
      if (value === null) {
        next.delete(key);
      } else {
        next.set(key, value);
      }
    }
    setParams(next, { replace });
  }

  /** After a decision in the preview: the next row of the list as it was, or the first one. */
  function moveOn(decidedId: string, before: ApprovalRow[]) {
    const index = before.findIndex((row) => row.activity.id === decidedId);
    const others = before.filter((row) => row.activity.id !== decidedId);
    const next = before[index + 1] ?? others[0];
    changeParams({ activity: next ? next.activity.id : null }, true);
  }

  const counts = summary.data;
  // Waiting for me is something to act on (a badge); All open is only counted (design system §5.2).
  const tabs: TabItem<ApprovalTab>[] = [
    {
      value: 'waiting',
      label: 'Waiting for me',
      badge: counts && <CountBadge count={counts.waitingForMe} label="activities" />,
    },
    { value: 'open', label: counts ? `All open (${counts.allOpen})` : 'All open' },
    { value: 'decided', label: 'Decided by me' },
  ];

  return (
    <>
      <PageHeader
        title="Approvals"
        description="Approve or decline the activities waiting for you, and follow the ones you sent."
      />
      <div className={styles.sections}>
        <SummaryCards counts={counts} failed={summary.isError} />

        <Tabs
          label="Approvals"
          tabs={tabs}
          value={tab}
          onChange={(value) => changeParams({ tab: value })}
        >
          <Card>
            <div className={styles.toolbar}>
              <div className={styles.search}>
                <SearchField
                  label="Search activity name or ID"
                  placeholder="Search activity name or ID"
                  value={search}
                  onChange={setSearch}
                />
              </div>
              <ApprovalsFilters
                filters={filters}
                options={filterOptions(allRows)}
                onChange={setFilters}
              />
              {rows.data && (
                <p className={styles.count} aria-live="polite">
                  {`${visible.length} of ${activitiesCount(allRows.length)}`}
                </p>
              )}
            </div>
            <Chips
              filters={filters}
              rows={allRows}
              onChange={setFilters}
              onClearAll={() => setFilters(NO_FILTERS)}
            />
            {rows.data ? (
              visible.length > 0 ? (
                <ApprovalsTable
                  rows={visible}
                  tab={tab}
                  onOpen={(id) => changeParams({ activity: id })}
                />
              ) : allRows.length > 0 ? (
                <EmptyState
                  icon={<SearchX />}
                  title="Nothing matches these filters"
                  action={
                    <Button
                      onClick={() => {
                        setSearch('');
                        setFilters(NO_FILTERS);
                      }}
                    >
                      Clear filters
                    </Button>
                  }
                />
              ) : (
                <EmptyState icon={<CircleCheckBig />} title={EMPTY_TAB[tab]} />
              )
            ) : rows.isError ? (
              <ErrorState error={rows.error} onRetry={() => void rows.refetch()} />
            ) : (
              <Skeleton lines={6} height="18px" />
            )}
          </Card>
        </Tabs>
      </div>

      {previewId && (
        <ApprovalPreviewPanel
          activityId={previewId}
          onClose={() => changeParams({ activity: null })}
          onDecided={() => moveOn(previewId, visible)}
        />
      )}
    </>
  );
}

/**
 * The view to open when the address names none: Waiting for me when something waits for the
 * person, otherwise All open; null while the counts are still on their way.
 */
function landingTabOf(counts: ApprovalCounts | undefined, failed: boolean): ApprovalTab | null {
  if (counts) {
    return counts.waitingForMe > 0 ? 'waiting' : 'open';
  }
  return failed ? 'waiting' : null;
}

interface SummaryCardsProps {
  counts: ApprovalCounts | undefined;
  failed: boolean;
}

/** The four cards (FR-028); tabs and filters never change them. */
function SummaryCards({ counts, failed }: SummaryCardsProps) {
  if (!counts && !failed) {
    return (
      <div className={styles.summary}>
        {[1, 2, 3, 4].map((key) => (
          <Skeleton key={key} height="104px" />
        ))}
      </div>
    );
  }
  const value = (count: number | undefined) => (count === undefined ? '—' : count);
  return (
    <div className={styles.summary}>
      <SummaryCard
        label="Waiting for me"
        value={value(counts?.waitingForMe)}
        tip="Submitted activities whose current approval level lists you."
        tone={counts && counts.waitingForMe > 0 ? 'review' : undefined}
      />
      <SummaryCard
        label="All open"
        value={value(counts?.allOpen)}
        tip="Every activity waiting for approval, at any level."
      />
      <SummaryCard
        label="Approved this month"
        value={value(counts?.approvedThisMonth)}
        tip="Activities that got their final approval this calendar month."
      />
      <SummaryCard
        label="Declined this month"
        value={value(counts?.declinedThisMonth)}
        tip="Activities declined this calendar month."
      />
    </div>
  );
}

interface ChipsProps {
  filters: ApprovalFilterValues;
  rows: ApprovalRow[];
  onChange: (filters: ApprovalFilterValues) => void;
  onClearAll: () => void;
}

/** The filters in use as removable chips, and Clear filters. */
function Chips({ filters, rows, onChange, onClearAll }: ChipsProps) {
  const { date } = useFormatters();
  if (filtersInUse(filters) === 0) {
    return null;
  }
  const chips = filterChips(filters, filterOptions(rows), date);
  return (
    <div className={styles.chips}>
      <FilterChips
        chips={chips}
        onRemove={(key) => onChange({ ...filters, [key as ApprovalFilterKey]: '' })}
        onClearAll={onClearAll}
      />
    </div>
  );
}
