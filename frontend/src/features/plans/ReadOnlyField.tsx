import type { ReactNode } from 'react';
import { LockedNote } from '@/ui/LockedNote/LockedNote';
import styles from './PlanForm.module.css';

/** A field shown but not editable: its label, the value and the lock with the reason (§6.6). */
export function ReadOnlyField({
  label,
  marking,
  value,
  children,
}: {
  label: string;
  marking?: string;
  value: string;
  children: ReactNode;
}) {
  return (
    <div className={styles.readOnly}>
      <p className={styles.readOnlyLabel}>
        {label} {marking && <span className={styles.marking}>{marking}</span>}
      </p>
      <p className={styles.readOnlyValue}>{value}</p>
      <LockedNote>{children}</LockedNote>
    </div>
  );
}
