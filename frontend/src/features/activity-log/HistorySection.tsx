import { History } from 'lucide-react';
import { useId } from 'react';
import type { ReactNode } from 'react';
import { useSession } from '@/session/useSession';
import { LinkButton } from '@/ui/Button/Button';
import { Card } from '@/ui/Card/Card';
import { CheckboxField } from '@/ui/CheckboxField/CheckboxField';
import { EmptyState } from '@/ui/EmptyState/EmptyState';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import type { PlanningLevel } from '@/ui/LevelTag/LevelTag';
import { cx } from '@/ui/cx';
import { HISTORY_PAGE_SIZE, historyKey, useLogEntries } from './api';
import { LogEntryList } from './LogEntryList';
import { planningLabel } from './log-text';
import { HISTORY_ID, historyHref } from './show-history';
import styles from './HistorySection.module.css';

type HistoryLevel = PlanningLevel | 'purchaseRequest';

/** The include option of each level (FR-012); purchase requests (032) have nothing inside. */
const INCLUDE_LABELS: Record<HistoryLevel, string | undefined> = {
  plan: 'Include items inside',
  campaign: 'Include items inside',
  initiative: 'Include items inside',
  activity: 'Include line items',
  lineItem: undefined,
  purchaseRequest: undefined,
};

export interface HistorySectionProps {
  recordType: string;
  recordId: string;
  recordName: string;
  level: HistoryLevel;
  /** What showHistory() looks for: HISTORY_ID on a page, panelHistoryId() in a side panel. */
  id?: string;
  /** card: its own card with an h2 (record pages); panel: a side panel's part with an h3. */
  variant?: 'card' | 'panel';
}

/** A record's History (011 FR-011 – FR-013): its log, newest first, and Open in activity log. */
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
  const noun = level === 'purchaseRequest' ? 'purchase request' : planningLabel(level);
  const subtitle = `Every change to this ${noun}, newest first`;
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
          // Saved in the background; a failed save says so with Try again.
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
