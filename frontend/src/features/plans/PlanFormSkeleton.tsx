import { Skeleton } from '@/ui/Skeleton/Skeleton';
import styles from './PlanForm.module.css';

/** The form's shape while its data loads. */
export function PlanFormSkeleton() {
  return (
    <div className={styles.main} aria-busy="true" aria-label="Loading the form">
      {[0, 1, 2, 3].map((section) => (
        <div key={section} className={styles.card}>
          <div className={styles.skeletonSection}>
            <Skeleton width="160px" height="18px" />
            <Skeleton height="38px" />
            <Skeleton height="38px" />
          </div>
        </div>
      ))}
    </div>
  );
}
