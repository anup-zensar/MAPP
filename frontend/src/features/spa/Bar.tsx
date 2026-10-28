import styles from './Spa.module.css';

/** A bar as long as the API says (0–100); decorative, since the amount beside it is written. */
export function Bar({
  length,
  series,
}: {
  length: number;
  series: 'planned' | 'actual' | 'share';
}) {
  return (
    <span className={styles.barTrack} aria-hidden="true">
      <span className={`${styles.bar} ${styles[series]}`} style={{ width: `${length}%` }} />
    </span>
  );
}
