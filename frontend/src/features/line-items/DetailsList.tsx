import type { ReactNode } from 'react';
import { FieldHelpTip } from '@/features/settings';
import styles from './DetailsList.module.css';

const EMPTY = '—';

export interface DetailsListProps {
  children: ReactNode;
  label?: string;
}

/** The one key–value layout of the activity and line-item pages: a label column, then the value. */
export function DetailsList({ children, label }: DetailsListProps) {
  return (
    <dl className={styles.list} aria-label={label}>
      {children}
    </dl>
  );
}

export interface DetailProps {
  label: string;
  /** The Field help entry explaining the value, shown as an ⓘ after the label. */
  helpKey?: string;
  children?: ReactNode;
  numeric?: boolean;
}

export function Detail({ label, helpKey, children, numeric = false }: DetailProps) {
  const empty = children === null || children === undefined || children === '';
  return (
    <div className={styles.row}>
      <dt className={styles.label}>
        {label}
        {helpKey && <FieldHelpTip helpKey={helpKey} label={label} />}
      </dt>
      <dd className={numeric ? `${styles.value} ${styles.numeric}` : styles.value}>
        {empty ? EMPTY : children}
      </dd>
    </div>
  );
}
