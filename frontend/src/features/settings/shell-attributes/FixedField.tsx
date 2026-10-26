import type { ReactNode } from 'react';
import styles from './ShellAttributes.module.css';

export interface FixedFieldProps {
  label: string;
  value: string;
  children: ReactNode;
}

/** A field that can't be changed here, shown as its value with the reason (design system §6.6). */
export function FixedField({ label, value, children }: FixedFieldProps) {
  return (
    <div className={styles.lockedField}>
      <span className={styles.lockedLabel}>{label}</span>
      <span className={styles.lockedValue}>{value}</span>
      {children}
    </div>
  );
}
