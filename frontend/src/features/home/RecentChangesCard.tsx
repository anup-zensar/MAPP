import { History } from 'lucide-react';
import { Link } from 'react-router';
import { COMING_SOON_SECTIONS } from '@/app/navigation';
import { Avatar, Card, EmptyState, LinkButton, Skeleton, Tooltip } from '@/components';
import { recordHref } from '@/lib/recordLinks';
import type { PlanningRecordType } from '@/lib/recordLinks';
import { useFormatters } from '@/lib/useFormatters';
import { useHomeRecentChanges } from './homeApi';
import type { ActivityLogEntry, HomeScope } from './homeApi';
import { RECENT_CHANGES, initialsOf, recentSentence } from './homeText';
import styles from './RecentChangesCard.module.css';
import { SectionError } from './SectionError';

/**
 * Recent changes come from the activity log (feature 011). Until the Activity log is built — while
 * it is still a "Coming soon" section — the API does not serve them, and the card says so instead
 * of asking.
 */
function activityLogBuilt(): boolean {
  return !COMING_SOON_SECTIONS.some((section) => section.path === '/activity-log');
}

/**
 * Recent changes (FR-015, FR-016): the 10 newest activity-log entries about the plans in scope —
 * who changed what, and when — each naming its record with a link while it exists.
 */
export function RecentChangesCard({ scope }: { scope: HomeScope }) {
  const built = activityLogBuilt();
  const changes = useHomeRecentChanges(scope, built);

  return (
    <Card
      title="Recent changes"
      subtitle={RECENT_CHANGES.subtitle[scope]}
      actions={
        built ? (
          <LinkButton size="sm" to="/activity-log">
            View activity log
          </LinkButton>
        ) : undefined
      }
    >
      {!built ? (
        <EmptyState
          icon={<History />}
          title={RECENT_CHANGES.comingSoon.title}
          text={RECENT_CHANGES.comingSoon.text}
        />
      ) : changes.isPending ? (
        <Skeleton lines={4} height="18px" />
      ) : changes.isError ? (
        <SectionError onRetry={() => void changes.refetch()} />
      ) : changes.data.items.length === 0 ? (
        <EmptyState icon={<History />} title={RECENT_CHANGES.empty} />
      ) : (
        <ul className={styles.entries}>
          {changes.data.items.map((entry) => (
            <RecentChange key={entry.id} entry={entry} />
          ))}
        </ul>
      )}
    </Card>
  );
}

const PLANNING_TYPES: PlanningRecordType[] = [
  'plan',
  'campaign',
  'initiative',
  'activity',
  'lineItem',
];

/** Where an entry's record is now: its page while it exists (D-194), otherwise no address. */
function hrefOf(entry: ActivityLogEntry): string | null {
  const now = entry.recordNow;
  if (now && now.state !== 'exists') {
    return null;
  }
  const type = entry.recordType as PlanningRecordType;
  if (!PLANNING_TYPES.includes(type)) {
    return null;
  }
  const planId = (now?.path ?? entry.path).find((ancestor) => ancestor.type === 'plan')?.id;
  if ((type === 'campaign' || type === 'initiative') && !planId) {
    return null;
  }
  return recordHref(type, entry.recordId, planId);
}

/** What follows the name of a record that no longer exists. */
const GONE: Partial<Record<string, string>> = {
  inRecycleBin: ' (in recycle bin)',
  deletedPermanently: ' (permanently deleted)',
};

function RecentChange({ entry }: { entry: ActivityLogEntry }) {
  const { dateTime, relativeTime } = useFormatters();
  const sentence = recentSentence(entry);
  const href = hrefOf(entry);
  const fullTime = dateTime(entry.at);

  return (
    <li className={styles.entry}>
      <Avatar initials={initialsOf(entry.actor.name)} size="sm" />
      <div className={styles.body}>
        <p className={styles.sentence}>
          {`${sentence.lead} '`}
          {href ? <Link to={href}>{sentence.name}</Link> : sentence.name}
          {`' (${sentence.id})${GONE[entry.recordNow?.state ?? 'exists'] ?? ''}`}
        </p>
        {/* A button, so keyboard and screen-reader users reach the full date too. */}
        <Tooltip content={fullTime}>
          <button type="button" className={styles.time}>
            <time dateTime={entry.at}>{relativeTime(entry.at)}</time>{' '}
            <span className="visually-hidden">({fullTime})</span>
          </button>
        </Tooltip>
      </div>
    </li>
  );
}
