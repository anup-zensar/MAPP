import styles from './CountBadge.module.css';

export interface CountBadgeProps {
  count: number;
  /** Read with the number by screen readers, not shown: "waiting for you", "unread". */
  label: string;
  /** Above this, "<max>+" (99 when left out). */
  max?: number;
}

/**
 * A small amber counter of things that need action (feature 008): the Approvals sidebar item, the
 * notification bell, the approvals tabs. Nothing shows for 0.
 */
export function CountBadge({ count, label, max = 99 }: CountBadgeProps) {
  if (count <= 0) {
    return null;
  }
  return (
    <span className={styles.badge}>
      {count > max ? `${max}+` : count} <span className="visually-hidden">{label}</span>
    </span>
  );
}
