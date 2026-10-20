import { History } from 'lucide-react';
import { useSession } from '@/app/useSession';
import { Card, CheckboxField, EmptyState, ErrorState, LinkButton, Skeleton } from '@/components';
import type { PlanningLevel } from '@/components';
import { HISTORY_PAGE_SIZE, historyKey, useLogEntries } from './activityLogApi';
import { LogEntryList } from './LogEntryList';
import { planningLabel } from './logText';
import { HISTORY_ID, historyHref } from './showHistory';
import styles from './HistorySection.module.css';

/** The include option of each level (FR-012): line items have nothing inside them. */
const INCLUDE_LABELS: Record<PlanningLevel, string | undefined> = {
  plan: 'Include items inside',
  campaign: 'Include items inside',
  initiative: 'Include items inside',
  activity: 'Include line items',
  lineItem: undefined,
};

export interface HistorySectionProps {
  /** The record's type as the activity log names it ("activity"). */
  recordType: string;
  recordId: string;
  recordName: string;
  /** The planning level, which decides the include option. */
  level: PlanningLevel;
}

/**
 * The History section of a plan, campaign, initiative, activity or line item page (feature 011,
 * US3, FR-011 – FR-013): the record's activity-log entries, newest first, 20 at a time, with the
 * person's choice to include the items inside (remembered for them), and **Open in activity
 * log** with the same Record filter. **⋯ More › View history** jumps here with showHistory().
 */
export function HistorySection({ recordId, level }: HistorySectionProps) {
  const { user, updatePreferences } = useSession();
  const includeLabel = INCLUDE_LABELS[level];
  const include = includeLabel ? (user?.preferences.historyIncludeInside ?? false) : false;
  const query = include ? { recordId, includeInside: true } : { recordId };
  const log = useLogEntries(historyKey(recordId, include), query, HISTORY_PAGE_SIZE);
  const logAddress = historyHref(recordId, include);

  return (
    <div id={HISTORY_ID} className={styles.history}>
      <Card
        title="History"
        subtitle={`Every change to this ${planningLabel(level)}, newest first`}
        actions={
          <LinkButton to={logAddress} size="sm">
            Open in activity log
          </LinkButton>
        }
      >
        <div className={styles.body}>
          {includeLabel && (
            <CheckboxField
              name="historyIncludeInside"
              label={includeLabel}
              checked={include}
              // The screen changes at once; the choice is saved for the person in the background,
              // and a failed save says "Your choice couldn't be saved." with Try again.
              onChange={(checked) => updatePreferences({ historyIncludeInside: checked })}
            />
          )}
          <HistoryEntries log={log} />
        </div>
      </Card>
    </div>
  );
}

function HistoryEntries({ log }: { log: ReturnType<typeof useLogEntries> }) {
  if (log.error) {
    return <ErrorState error={log.error} onRetry={log.refetch} />;
  }
  if (log.isPending || !log.firstPage) {
    return <Skeleton lines={3} />;
  }
  if (log.firstPage.total === 0) {
    return <EmptyState icon={<History />} title="No changes recorded yet." />;
  }
  return (
    <LogEntryList
      entries={log.entries}
      total={log.firstPage.total}
      hasMore={log.hasMore}
      onLoadMore={log.loadMore}
      isLoadingMore={log.isLoadingMore}
      headingLevel={3}
    />
  );
}
