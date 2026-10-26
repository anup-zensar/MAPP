import { LockedNote } from '@/ui/LockedNote/LockedNote';
import styles from '@/ui/settings-table.module.css';

export interface ReadOnlyFieldProps {
  label: string;
  value: string;
  /** Why it can't change. */
  children: string;
}

export function ReadOnlyField({ label, value, children }: ReadOnlyFieldProps) {
  return (
    <div className={styles.readOnly}>
      <span className={styles.readOnlyLabel}>{label}</span>
      <span className={styles.readOnlyValue}>{value}</span>
      <LockedNote>{children}</LockedNote>
    </div>
  );
}
