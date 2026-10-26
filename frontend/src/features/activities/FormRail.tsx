import { CircleCheck } from 'lucide-react';
import type { ReactNode } from 'react';
import { FormSection } from '@/ui/FormSection/FormSection';
import { filledText } from './formProgress';
import type { RailSection } from './formProgress';
import styles from './FormRail.module.css';
import { useWideScreen } from './useWideScreen';

/**
 * The right rail of a form page (design system §2.4 C, functional review P-04): "On this page",
 * each section as a link with how many needed fields it still lacks (✓ when none), and the
 * overall count. Sticky at ≥ 1280 px; below that only the count shows, as one line above the
 * form (FormProgress).
 */
export function FormRail({
  sections,
  children,
}: {
  sections: RailSection[];
  children?: ReactNode;
}) {
  return (
    <aside className={styles.rail} aria-label="On this page">
      <div className={styles.card}>
        <h2 className={styles.title}>On this page</h2>
        <ul className={styles.list}>
          {sections.map((section) => (
            <li key={section.id}>
              <a href={`#${section.id}`} className={styles.link}>
                <span>{section.title}</span>
                {section.needed > 0 &&
                  (section.missing === 0 ? (
                    <CircleCheck aria-label="all filled in" className={styles.done} />
                  ) : (
                    <span className={styles.count}>{section.missing} needed</span>
                  ))}
              </a>
            </li>
          ))}
        </ul>
        <p className={styles.total}>{filledText(sections)}</p>
      </div>
      {children}
    </aside>
  );
}

/**
 * The form column (at most 720 px, left-aligned) and from 1280 px the rail beside it (design system
 * §2.4 C); below 1280 px the rail's count shows as one line above the form instead.
 */
export function FormLayout({
  sections,
  railExtra,
  narrowExtra,
  children,
}: {
  sections: RailSection[];
  /** More cards in the rail (the line item's summary strip). */
  railExtra?: ReactNode;
  /** What shows above the form instead, below 1280 px. */
  narrowExtra?: ReactNode;
  children: ReactNode;
}) {
  const wide = useWideScreen();
  return (
    <div className={styles.layout}>
      <div className={styles.column}>
        {!wide && narrowExtra}
        {!wide && <FormProgress sections={sections} />}
        {children}
      </div>
      {wide && <FormRail sections={sections}>{railExtra}</FormRail>}
    </div>
  );
}

/** The overall count as one line, for widths without the rail (below 1280 px). */
export function FormProgress({ sections }: { sections: RailSection[] }) {
  return <p className={styles.progress}>{filledText(sections)}</p>;
}

/**
 * One section of a form page as a white card (design system §5.5 `FormSection`): the shared
 * section inside a card with the id the rail links to.
 */
export function FormCard({
  id,
  title,
  description,
  children,
}: {
  id: string;
  title: string;
  description?: string;
  children: ReactNode;
}) {
  return (
    <div id={id} className={styles.formCard}>
      <FormSection title={title} description={description}>
        {children}
      </FormSection>
    </div>
  );
}

/**
 * A form section's white card around content that draws its own headings (Goals, the Shell
 * attribute groups); with a label it is a named region.
 */
export function FormCardFrame({
  id,
  label,
  className,
  children,
}: {
  id: string;
  label?: string;
  className?: string;
  children: ReactNode;
}) {
  const classes = className ? `${styles.formCard} ${className}` : styles.formCard;
  return label ? (
    <section id={id} aria-label={label} className={classes}>
      {children}
    </section>
  ) : (
    <div id={id} className={classes}>
      {children}
    </div>
  );
}
