import { useId } from 'react';
import type { ReactNode } from 'react';
import styles from './PanelSection.module.css';

export interface PanelSectionProps {
  title: ReactNode;
  children: ReactNode;
  /** The section's own address in the panel, for links that end in #id (Home's, feature 009). */
  id?: string;
}

/** A titled part of a details panel (design system §6.3), a region named by its heading. */
export function PanelSection({ title, children, id }: PanelSectionProps) {
  const headingId = useId();
  return (
    <section id={id} aria-labelledby={headingId} className={styles.section}>
      <h3 id={headingId} className={styles.title}>
        {title}
      </h3>
      {children}
    </section>
  );
}

export interface FactProps {
  label: ReactNode;
  children: ReactNode;
}

/** A definition list of Facts. */
export function Facts({ children }: { children: ReactNode }) {
  return <dl className={styles.facts}>{children}</dl>;
}

/** One label and value of Facts. */
export function Fact({ label, children }: FactProps) {
  return (
    <div className={styles.fact}>
      <dt className={styles.label}>{label}</dt>
      <dd className={styles.value}>{children}</dd>
    </div>
  );
}
