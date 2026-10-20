import type { components } from '@/api/schema';
import styles from './AgeingBars.module.css';

type AgeingBucket = components['schemas']['ApprovalAgeingBucket'];

/** The bucket labels of 017 plan "User-visible messages", in order. */
const LABELS: Record<AgeingBucket['bucket'], string> = {
  under1: 'Under 1 day',
  days1to2: '1–2 days',
  days3to5: '3–5 days',
  days6to10: '6–10 days',
  over10: 'More than 10 days',
};

/**
 * "How long approvals have waited" (017 US6, FR-030): one row per bucket, in order — the label,
 * a bar as long as its share of the largest bucket, and the count written out. One series, one
 * colour: the words and numbers carry everything, the bars only show the shape at a glance.
 */
export function AgeingBars({ buckets }: { buckets: AgeingBucket[] }) {
  const max = Math.max(1, ...buckets.map((bucket) => bucket.count));
  return (
    <ul role="list" className={styles.bars}>
      {buckets.map((bucket) => (
        <li key={bucket.bucket} className={styles.row}>
          <span className={styles.label}>{LABELS[bucket.bucket] ?? bucket.label}</span>
          <span className={styles.track} aria-hidden="true">
            {bucket.count > 0 && (
              <span className={styles.bar} style={{ width: `${(bucket.count / max) * 100}%` }} />
            )}
          </span>
          <span className={styles.count}>{bucket.count}</span>
        </li>
      ))}
    </ul>
  );
}
