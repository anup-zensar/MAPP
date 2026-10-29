import { PROCESSING } from './simulated-gsap-text';
import styles from './GsapProgress.module.css';

export function GsapProgress() {
  return (
    <div role="status" className={styles.progress}>
      <span className={styles.track} aria-hidden="true">
        <span className={styles.bar} />
      </span>
      <span>{PROCESSING}</span>
    </div>
  );
}
