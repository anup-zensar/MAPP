import { useFormatters } from '@/format/useFormatters';
import type { ActivityLogEntry } from './api';
import styles from './SampleRecordsPage.module.css';

/**
 * "Priya Raman changed sample folder 'Ideas 2' (SF-1001)", then where the record was ("in Ideas")
 * and when, then each changed field: "Name: Ideas → Ideas 2".
 */
export function SampleLogEntry({ entry }: { entry: ActivityLogEntry }) {
  const { dateTime } = useFormatters();
  const place = entry.path.map((record) => record.name).join(' › ');

  return (
    <li className={styles.logEntry}>
      <p>{`${entry.actor.name} ${entry.verb} ${entry.recordTypeLabel} '${entry.recordName}' (${entry.recordId})`}</p>
      <p className={styles.meta}>
        {place && `in ${place} · `}
        <time dateTime={entry.at}>{dateTime(entry.at)}</time>
      </p>
      {entry.changes.length > 0 && (
        <ul className={styles.changes}>
          {entry.changes.map((change) => (
            <li key={change.field}>{`${change.label}: ${change.oldText} → ${change.newText}`}</li>
          ))}
        </ul>
      )}
    </li>
  );
}
