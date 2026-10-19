import type { ReactNode } from 'react';
import styles from './PageHeader.module.css';

export interface PageHeaderProps {
  /** The small blue word above the title, usually the section ("Plans"). */
  eyebrow: string;
  /** The page's main heading (the only <h1> on the page). */
  title: string;
  /** An optional phrase after the title, in grey ("Welcome back," + "Priya."). */
  highlight?: string;
  /** One line that says what the page is for. */
  description?: string;
  /** The page's actions, on the right: at most one primary button. */
  actions?: ReactNode;
}

/** The header at the top of every signed-in page (design system §2, FR-010). */
export function PageHeader({ eyebrow, title, highlight, description, actions }: PageHeaderProps) {
  return (
    <header className={styles.header}>
      <div className={styles.text}>
        <p className={styles.eyebrow}>{eyebrow}</p>
        <h1 className={styles.title}>
          {title}
          {/* The space stays outside the span, so screen readers keep it between the words. */}
          {highlight && ' '}
          {highlight && <span className={styles.highlight}>{highlight}</span>}
        </h1>
        {description && <p className={styles.description}>{description}</p>}
      </div>
      {actions && <div className={styles.actions}>{actions}</div>}
    </header>
  );
}
