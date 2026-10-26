import { Card } from '@/ui/Card/Card';
import type { ActivityLogEntry } from './api';
import { SampleLogEntry } from './SampleLogEntry';
import styles from './SampleRecordsPage.module.css';

/** The newest activity-log entries (FR-022): who did what to which record, where and when. */
export function ActivityLogCard({ entries }: { entries: ActivityLogEntry[] }) {
  return (
    <Card title="Activity log">
      <ol role="list" className={styles.list}>
        {entries.map((entry) => (
          <SampleLogEntry key={entry.id} entry={entry} />
        ))}
      </ol>
    </Card>
  );
}
