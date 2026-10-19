import styles from './Skeleton.module.css';

export interface SkeletonProps {
  /** The width of the placeholder, as CSS ("100%", "240px"). */
  width?: string;
  /** The height of each line, as CSS. */
  height?: string;
  /** How many lines of text it stands for; the last one is shorter. */
  lines?: number;
}

/**
 * Grey shapes in the place of content that is still loading (design system §5). Screen readers
 * hear "Loading…" instead. When people prefer less motion, the shapes keep still.
 */
export function Skeleton({ width = '100%', height = '14px', lines = 1 }: SkeletonProps) {
  return (
    <div role="status" className={styles.skeleton} style={{ width }}>
      <span className="visually-hidden">Loading…</span>
      <div aria-hidden="true" className={styles.lines}>
        {Array.from({ length: lines }, (_, index) => (
          <span key={index} className={styles.line} style={{ height }} />
        ))}
      </div>
    </div>
  );
}
