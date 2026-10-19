import type { ReactNode } from 'react';
import styles from './EmptyState.module.css';

export interface EmptyStateProps {
  /** A lucide icon that hints at the missing thing. */
  icon?: ReactNode;
  /** One sentence: "No sample folders yet — create your first sample folder". */
  title: string;
  /** One more sentence, when needed. */
  text?: string;
  /** Usually the create button. */
  action?: ReactNode;
}

/**
 * Shown where a list or page has nothing yet: what is missing and how to add it
 * (design system §5).
 */
export function EmptyState({ icon, title, text, action }: EmptyStateProps) {
  return (
    <div className={styles.empty}>
      {icon && <div className={styles.icon}>{icon}</div>}
      <p className={styles.title}>{title}</p>
      {text && <p className={styles.text}>{text}</p>}
      {action && <div className={styles.action}>{action}</div>}
    </div>
  );
}
