import { useId } from 'react';
import type { ReactNode } from 'react';
import styles from './Card.module.css';

export interface CardProps {
  /** The card's heading; it also names the card for screen readers. */
  title?: string;
  subtitle?: string;
  /** Buttons for the whole card, on the right of the title. */
  actions?: ReactNode;
  children: ReactNode;
}

/** A rounded white content card with an optional header (design system §2). */
export function Card({ title, subtitle, actions, children }: CardProps) {
  const titleId = useId();
  const hasHeader = Boolean(title || subtitle || actions);

  return (
    <section className={styles.card} aria-labelledby={title ? titleId : undefined}>
      {hasHeader && (
        <div className={styles.header}>
          <div>
            {title && (
              <h2 id={titleId} className={styles.title}>
                {title}
              </h2>
            )}
            {subtitle && <p className={styles.subtitle}>{subtitle}</p>}
          </div>
          {actions && <div className={styles.actions}>{actions}</div>}
        </div>
      )}
      <div className={styles.body}>{children}</div>
    </section>
  );
}
