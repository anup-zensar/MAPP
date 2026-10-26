import { useFormatters } from '@/format/useFormatters';
import styles from './panels.module.css';

interface Recorded {
  createdAt: string;
  createdBy: { name: string };
  updatedAt: string;
  updatedBy: { name: string };
}

/**
 * "Created by Alex Morgan on 1 Mar 2026, 09:00 · Last changed by Priya Raman on 12 Mar 2026,
 * 14:32", in the person's date format (FR-024).
 */
export function RecordLine({ record }: { record: Recorded }) {
  const { dateTime } = useFormatters();
  return (
    <p className={styles.recordLine}>
      {`Created by ${record.createdBy.name} on ${dateTime(record.createdAt)} · ` +
        `Last changed by ${record.updatedBy.name} on ${dateTime(record.updatedAt)}`}
    </p>
  );
}
