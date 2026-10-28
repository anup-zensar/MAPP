import type { ReactNode } from 'react';
import { InfoTip } from '@/ui/InfoTip/InfoTip';
import styles from './Details.module.css';

const EMPTY = '—';

export function DetailsList({ children, label }: { children: ReactNode; label?: string }) {
  return (
    <dl className={styles.list} aria-label={label}>
      {children}
    </dl>
  );
}

export interface DetailProps {
  label: string;
  /** "Filled by GSAP" and the like: why the value is read-only, in an ⓘ (design system §6.6). */
  tip?: string;
  /** The value; "—" when it is empty. */
  children?: ReactNode;
  numeric?: boolean;
}

export function Detail({ label, tip, children, numeric = false }: DetailProps) {
  const empty = children === null || children === undefined || children === '';
  return (
    <div className={styles.row}>
      <dt className={styles.label}>
        {label}
        {tip && <InfoTip label={label} text={tip} />}
      </dt>
      <dd className={numeric ? `${styles.value} ${styles.numeric}` : styles.value}>
        {empty ? EMPTY : children}
      </dd>
    </div>
  );
}
