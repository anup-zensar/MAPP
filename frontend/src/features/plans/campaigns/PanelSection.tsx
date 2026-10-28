import { useId } from 'react';
import type { ReactNode } from 'react';
import { FieldHelpTip } from '@/features/settings';
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
  helpKey?: string;
  children: ReactNode;
}

export function Facts({ children }: { children: ReactNode }) {
  return <dl className={styles.facts}>{children}</dl>;
}

export function Fact({ label, helpKey, children }: FactProps) {
  return (
    <div className={styles.fact}>
      <dt className={styles.label}>
        {label}
        {helpKey && typeof label === 'string' && <FieldHelpTip helpKey={helpKey} label={label} />}
      </dt>
      <dd className={styles.value}>{children}</dd>
    </div>
  );
}
