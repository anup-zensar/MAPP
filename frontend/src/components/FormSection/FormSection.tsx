import { useId } from 'react';
import type { ReactNode } from 'react';
import styles from './FormSection.module.css';

export interface FormSectionProps {
  /** The section's heading: "Basics", "Dates & cost", "Shell attributes" (design system §7). */
  title: string;
  /** One line under the heading, when needed. */
  description?: string;
  /** The section's fields. */
  children: ReactNode;
}

/**
 * One section of a full-page form (feature 004; plans, activities, line items): a heading and
 * its fields in one column, in the order people think (design system §7).
 */
export function FormSection({ title, description, children }: FormSectionProps) {
  const headingId = useId();
  return (
    <section aria-labelledby={headingId} className={styles.section}>
      <div>
        <h2 id={headingId} className={styles.title}>
          {title}
        </h2>
        {description && <p className={styles.description}>{description}</p>}
      </div>
      <div className={styles.fields}>{children}</div>
    </section>
  );
}
