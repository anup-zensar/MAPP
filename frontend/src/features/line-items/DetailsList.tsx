import type { ReactNode } from 'react';
import styles from './DetailsList.module.css';

/** Shown for an empty value (design system §4.4). */
const EMPTY = '—';

export interface DetailsListProps {
  children: ReactNode;
  /** Names the list for screen readers when the card heading does not ("Goals"). */
  label?: string;
}

/**
 * The one key–value layout of the activity and line-item pages (design system §5.4
 * `DescriptionList`, UI audit AC-06, X-08): a 200 px label column, then the value; below 600 px the
 * label sits above its value. It lives here until the shared `DescriptionList` component exists,
 * so every value on these pages starts at the same x.
 */
export function DetailsList({ children, label }: DetailsListProps) {
  return (
    <dl className={styles.list} aria-label={label}>
      {children}
    </dl>
  );
}

export interface DetailProps {
  label: ReactNode;
  /** The value; "—" when it is empty (null, undefined or ""). */
  children?: ReactNode;
  /** Numbers and amounts: tabular figures. */
  numeric?: boolean;
}

/** One row of a DetailsList. */
export function Detail({ label, children, numeric = false }: DetailProps) {
  const empty = children === null || children === undefined || children === '';
  return (
    <div className={styles.row}>
      <dt className={styles.label}>{label}</dt>
      <dd className={numeric ? `${styles.value} ${styles.numeric}` : styles.value}>
        {empty ? EMPTY : children}
      </dd>
    </div>
  );
}
