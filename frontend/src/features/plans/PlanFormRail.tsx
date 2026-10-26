import type { MouseEvent } from 'react';
import styles from './PlanForm.module.css';
import { progressText, SECTIONS } from './plan-form-sections';
import type { NeededFields } from './plan-form-sections';

/** From 1,280 px: the rail that links the sections and counts the needed fields still empty. */
export function PlanFormRail({ needed }: { needed: NeededFields }) {
  return (
    <nav className={styles.rail} aria-label="On this page">
      <p className={styles.railTitle}>On this page</p>
      <ul className={styles.railList}>
        {SECTIONS.map((section) => {
          const left = needed.bySection[section.id].missing;
          return (
            <li key={section.id}>
              <a href={`#${section.id}`} className={styles.railLink} onClick={jumpTo}>
                <span>{section.title}</span>
                <span className={left > 0 ? styles.railMissing : styles.railDone}>
                  {left > 0 ? `${left} needed` : 'Done'}
                </span>
              </a>
            </li>
          );
        })}
      </ul>
      <p className={styles.railCount}>{progressText(needed)}</p>
    </nav>
  );
}

/** The rail's link: moves the focus to the section's first field as well as scrolling to it. */
function jumpTo(event: MouseEvent<HTMLAnchorElement>) {
  const id = event.currentTarget.getAttribute('href')?.slice(1);
  const section = id ? document.getElementById(id) : null;
  if (!section) {
    return;
  }
  event.preventDefault();
  section.scrollIntoView?.({ block: 'start' });
  section.querySelector<HTMLElement>('input, select, textarea, button')?.focus({
    preventScroll: true,
  });
}
