import { CircleCheckBig, SearchX, UserRoundCog } from 'lucide-react';
import { useRef, useState } from 'react';
import { useSearchParams } from 'react-router';
import { Button } from '@/ui/Button/Button';
import { Card } from '@/ui/Card/Card';
import { CountBadge } from '@/ui/CountBadge/CountBadge';
import { EmptyState } from '@/ui/EmptyState/EmptyState';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { FilterChips } from '@/ui/FilterChips/FilterChips';
import { PageHeader } from '@/ui/PageHeader/PageHeader';
import { SearchField } from '@/ui/SearchField/SearchField';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { SummaryCard } from '@/ui/SummaryCard/SummaryCard';
import { Tabs } from '@/ui/Tabs/Tabs';
import type { TabItem } from '@/ui/Tabs/Tabs';
import type { components } from '@/api/generated/schema';
import { isPurchaseRequestId } from '@/ui/record-links';
import { usePageTitle } from '@/ui/usePageTitle';
import { useFormatters } from '@/format/useFormatters';
import {
  NO_FILTERS,
  applyFilters,
  filterChips,
  filterOptions,
  filtersInUse,
  isFilterKey,
} from './approval-filters';
import type { ApprovalFilterValues } from './approval-filters';
import { ApprovalPreviewPanel } from './ApprovalPreviewPanel';
import { useApprovalRows } from './api';
import type { ApprovalRow, ApprovalTab } from './api';
import { ApprovalsFilters } from './ApprovalsFilters';
import { ApprovalsTable } from './ApprovalsTable';
import { DelegationBanner } from './DelegationBanner';
import { DelegationPanel } from './DelegationPanel';
import { activitiesCount } from './messages';
import { MonitoringView } from './MonitoringView';
import { PurchaseRequestPreviewPanel } from './PurchaseRequestPreviewPanel';
import { approvalsAddressOf } from './schemas';
import type { PageTab } from './schemas';
import { useApprovalSummary } from './useApprovalSummary';
import { useChosenTabInView } from './useChosenTabInView';
import styles from './ApprovalsPage.module.css';

type ApprovalCounts = components['schemas']['ApprovalCounts'];

const EMPTY_TAB: Record<ApprovalTab, string> = {
  waiting: "Nothing is waiting for you. You're all caught up.",
  open: 'No activities are waiting for approval.',
  decided: "You haven't approved or declined any activities yet.",
};

export function ApprovalsPage() {
  usePageTitle('Approvals');
  const [params, setParams] = useSearchParams();
  const { tab: chosenTab, previewId, delegationsOpen } = approvalsAddressOf(params);
  const [search, setSearch] = useState('');
  const [filters, setFilters] = useState<ApprovalFilterValues>(NO_FILTERS);
  const summary = useApprovalSummary();
  const landingTab = landingTabOf(summary.data, summary.isError);
  const tab: PageTab = chosenTab ?? landingTab ?? 'waiting';
  const listTab: ApprovalTab = tab === 'monitoring' ? 'open' : tab;
  const rows = useApprovalRows(
    listTab,
    tab !== 'monitoring' && (chosenTab !== null || landingTab !== null),
  );
  const allRows = rows.data?.items ?? [];
  const visible = applyFilters(allRows, search, filters);
  const tabsArea = useRef<HTMLDivElement>(null);
  useChosenTabInView(tabsArea, tab);

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

  function moveOn(decidedId: string, before: ApprovalRow[]) {
    const ids = before.map((row) => row.subject.id);
    const index = ids.indexOf(decidedId);
    const next = ids[index + 1] ?? ids.find((id) => id !== decidedId);
    changeParams({ activity: next ?? null }, true);
  }

  const counts = summary.data;
  const tabs: TabItem<PageTab>[] = [
    {
      value: 'waiting',
      label: 'Waiting for me',
      badge: counts && <CountBadge count={counts.waitingForMe} label="activities" />,
    },
    { value: 'open', label: counts ? `All open (${counts.allOpen})` : 'All open' },
    { value: 'decided', label: 'Decided by me' },
    { value: 'monitoring', label: 'Monitoring' },
  ];
  const openPreview = (id: string) => changeParams({ activity: id });

  return (
    <>
      <PageHeader
        title="Approvals"
        description="Approve or decline the activities waiting for you, and follow the ones you sent."
        actions={
          <Button icon={<UserRoundCog />} onClick={() => changeParams({ panel: 'delegations' })}>
            Delegate my approvals
          </Button>
        }
      />
      <div className={styles.sections}>
        <DelegationBanner />
        <SummaryCards counts={counts} failed={summary.isError} />

        <div ref={tabsArea}>
          <Tabs
            label="Approvals"
            tabs={tabs}
            value={tab}
            onChange={(value) => changeParams({ tab: value })}
          >
            {tab === 'monitoring' ? (
              <MonitoringView onOpen={openPreview} />
            ) : (
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
                    <ApprovalsTable rows={visible} tab={listTab} onOpen={openPreview} />
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
                    <EmptyState icon={<CircleCheckBig />} title={EMPTY_TAB[listTab]} />
                  )
                ) : rows.isError ? (
                  <ErrorState error={rows.error} onRetry={() => void rows.refetch()} />
                ) : (
                  <Skeleton lines={6} height="18px" />
                )}
              </Card>
            )}
          </Tabs>
        </div>
      </div>

      {previewId &&
        (isPurchaseRequestId(previewId) ? (
          <PurchaseRequestPreviewPanel
            purchaseRequestId={previewId}
            onClose={() => changeParams({ activity: null })}
            onDecided={() => moveOn(previewId, visible)}
          />
        ) : (
          <ApprovalPreviewPanel
            activityId={previewId}
            onClose={() => changeParams({ activity: null })}
            onDecided={() => moveOn(previewId, visible)}
          />
        ))}
      {delegationsOpen && <DelegationPanel onClose={() => changeParams({ panel: null })} />}
    </>
  );
}

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
        onRemove={(key) => {
          if (isFilterKey(key)) {
            onChange({ ...filters, [key]: '' });
          }
        }}
        onClearAll={onClearAll}
      />
    </div>
  );
}
