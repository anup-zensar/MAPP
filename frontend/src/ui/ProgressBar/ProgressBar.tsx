import { cx } from '../cx';
import styles from './ProgressBar.module.css';

export interface ProgressBarProps {
  /** The percentage (125 for 125 %); null when there is nothing to measure (the caller says why). */
  value: number | null;
  /** Names the bar for screen readers: "Budget used". */
  label: string;
  /** problem: a red fill, for a budget used above 100 %. */
  tone?: 'problem';
}

/**
 * A thin progress bar (design system §4): budget used, approval progress. The fill stops at the
 * end of the track above 100 %; screen readers hear the real percentage.
 */
export function ProgressBar({ value, label, tone }: ProgressBarProps) {
  const shown = value === null ? 0 : Math.min(100, Math.max(0, value));
  return (
    <div
      role="progressbar"
      aria-label={label}
      aria-valuemin={0}
      aria-valuemax={100}
      aria-valuenow={value === null ? undefined : shown}
      aria-valuetext={value === null ? undefined : `${value} %`}
      className={styles.track}
    >
      <div
        className={cx(styles.fill, tone === 'problem' && styles.problem)}
        style={{ width: `${shown}%` }}
      />
    </div>
  );
}
