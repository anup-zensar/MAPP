import type { ReactNode } from 'react';
import styles from './CentredMessage.module.css';

export interface CentredMessageProps {
  /** A lucide outline icon, drawn 48 px in grey. */
  icon: ReactNode;
  /** The page's only heading. */
  title: string;
  /** One or two sentences under the title. */
  children: ReactNode;
  /** The way on: one button. */
  action: ReactNode;
}

/**
 * The one layout of the pages that have nothing else to show — Page not found and Coming soon
 * (UI audit M-02): centred in the content column, without a card, an icon, the title, the words
 * and one way on.
 */
export function CentredMessage({ icon, title, children, action }: CentredMessageProps) {
  return (
    <div className={styles.page}>
      <div className={styles.icon} aria-hidden="true">
        {icon}
      </div>
      <h1 className={styles.title}>{title}</h1>
      <div className={styles.text}>{children}</div>
      <div className={styles.action}>{action}</div>
    </div>
  );
}
