import { Skeleton } from '@/ui/Skeleton/Skeleton';
import styles from './AllWorkPage.module.css';

/** Eight grey lines while the first answer loads. */
export function GridSkeleton() {
  return (
    <div className={styles.state}>
      <Skeleton lines={8} height="18px" />
    </div>
  );
}
