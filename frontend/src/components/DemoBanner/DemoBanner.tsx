import styles from './DemoBanner.module.css';

/** The thin navy system message on every page of this phase (FR-016, design system §2). */
export function DemoBanner() {
  return (
    <p className={styles.banner}>Demo environment — sign-in and data are for demonstration only</p>
  );
}
