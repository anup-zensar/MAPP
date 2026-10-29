import type { ReactNode } from 'react';
import { InfoTip } from '@/ui/InfoTip/InfoTip';
import { LockedNote } from '@/ui/LockedNote/LockedNote';
import styles from './PlanForm.module.css';

/** A field shown but not editable: its label, the value and the lock with the reason (§6.6). */
export function ReadOnlyField({
  label,
  marking,
  tip,
  value,
  children,
}: {
  label: string;
  marking?: string;
  tip?: string;
  value: string;
  children: ReactNode;
}) {
  return (
    <div className={styles.readOnly}>
      <p className={styles.readOnlyLabel}>
        {label} {tip && <InfoTip label={label} text={tip} />}
        {marking && <span className={styles.marking}>{marking}</span>}
      </p>
      <p className={styles.readOnlyValue}>{value}</p>
      <LockedNote>{children}</LockedNote>
    </div>
  );
}
