import type { components } from '@/api/generated/schema';
import { localDate } from '@/format/intl';
import { useFormatters } from '@/format/useFormatters';
import styles from './SettingsTable.module.css';

type ActorRef = components['schemas']['ActorRef'];

export interface MetaLineProps {
  record: { createdAt: string; createdBy: ActorRef; updatedAt: string; updatedBy: ActorRef };
}

/** "Created by Priya Raman on 12 Mar 2026, last changed by Daniel Okoro on 14 Mar 2026". */
export function MetaLine({ record }: MetaLineProps) {
  const { date } = useFormatters();
  return (
    <p className={styles.meta}>
      {`Created by ${record.createdBy.name} on ${date(localDate(record.createdAt))}, ` +
        `last changed by ${record.updatedBy.name} on ${date(localDate(record.updatedAt))}`}
    </p>
  );
}
