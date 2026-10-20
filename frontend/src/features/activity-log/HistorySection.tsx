import { History } from 'lucide-react';
import { useId } from 'react';
import type { ReactNode } from 'react';
import { useSession } from '@/app/useSession';
import { Card, CheckboxField, EmptyState, ErrorState, LinkButton, Skeleton } from '@/components';
import type { PlanningLevel } from '@/components';
import { cx } from '@/lib/cx';
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
  /**
   * The section's id, which showHistory() looks for: HISTORY_ID on a page, panelHistoryId() in a
   * side panel that opens over a page with its own History section.
   */
  id?: string;
  /**
   * card: a card of its own with an h2 (the plan, activity and line item pages); panel: a part of
   * a side panel's details with an h3, drawn like the panel's other parts (campaigns, initiatives).
   */
  variant?: 'card' | 'panel';
}

/**
 * The History section of a plan, campaign, initiative, activity or line item page (feature 011,
 * US3, FR-011 – FR-013): the record's activity-log entries, newest first, 20 at a time, with the
 * person's choice to include the items inside (remembered for them), and **Open in activity
 * log** with the same Record filter. **⋯ More › View history** jumps here with showHistory().
 */
export function HistorySection({
  recordId,
  level,
  id = HISTORY_ID,
  variant = 'card',
}: HistorySectionProps) {
  const { user, updatePreferences } = useSession();
  const includeLabel = INCLUDE_LABELS[level];
  const include = includeLabel ? (user?.preferences.historyIncludeInside ?? false) : false;
  const query = include ? { recordId, includeInside: true } : { recordId };
  const log = useLogEntries(historyKey(recordId, include), query, HISTORY_PAGE_SIZE);
  const subtitle = `Every change to this ${planningLabel(level)}, newest first`;
  const openInLog = (
    <LinkButton to={historyHref(recordId, include)} size="sm">
      Open in activity log
    </LinkButton>
  );

  const body = (
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
      <HistoryEntries log={log} headingLevel={variant === 'card' ? 3 : 4} />
    </div>
  );

  return (
    <div id={id} className={cx(styles.history, variant === 'panel' && styles.inPanel)}>
      {variant === 'card' ? (
        <Card title="History" subtitle={subtitle} actions={openInLog}>
          {body}
        </Card>
      ) : (
        <PanelPart subtitle={subtitle} actions={openInLog}>
          {body}
        </PanelPart>
      )}
    </div>
  );
}

/** The History part of a side panel: an h3 with its subtitle and action, over a hairline. */
function PanelPart({
  subtitle,
  actions,
  children,
}: {
  subtitle: string;
  actions: ReactNode;
  children: ReactNode;
}) {
  const headingId = useId();
  return (
    <section aria-labelledby={headingId} className={styles.panelPart}>
      <div className={styles.panelHeader}>
        <div>
          <h3 id={headingId} className={styles.panelTitle}>
            History
          </h3>
          <p className={styles.panelSubtitle}>{subtitle}</p>
        </div>
        {actions}
      </div>
      {children}
    </section>
  );
}

function HistoryEntries({
  log,
  headingLevel,
}: {
  log: ReturnType<typeof useLogEntries>;
  headingLevel: 3 | 4;
}) {
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
      headingLevel={headingLevel}
    />
  );
}
