import { Link } from 'react-router';
import { Card } from '@/ui/Card/Card';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { useFormatters } from '@/format/useFormatters';
import { useAccessChanges } from './api';
import { changeText } from './access-words';
import styles from './access.module.css';

/**
 * Recent access changes (US7 scenario 3, FR-038): the latest 20 changes of roles, people's roles
 * and the rules, newest first, each linking to its record's history in the activity log.
 */
export function RecentAccessChanges() {
  const changes = useAccessChanges();
  const { dateTime } = useFormatters();

  return (
    <Card title="Recent access changes">
      {changes.data ? (
        changes.data.items.length === 0 ? (
          <p className={styles.note}>No access changes yet.</p>
        ) : (
          <ul className={styles.changes}>
            {changes.data.items.map((entry) => (
              <li key={entry.id}>
                <Link to={`/activity-log?record=${encodeURIComponent(entry.recordId)}`}>
                  {changeText(entry)}
                </Link>
                <span className={styles.when}>{dateTime(entry.at)}</span>
              </li>
            ))}
          </ul>
        )
      ) : changes.isError ? (
        <ErrorState error={changes.error} onRetry={() => void changes.refetch()} />
      ) : (
        <Skeleton lines={4} height="18px" />
      )}
    </Card>
  );
}
