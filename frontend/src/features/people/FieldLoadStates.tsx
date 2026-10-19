import { RotateCw } from 'lucide-react';
import { Button, Skeleton } from '@/components';
import styles from './FieldLoadStates.module.css';

/** A field whose choices are still loading: its label over a field-sized grey shape. */
export function FieldLoading({ label }: { label: string }) {
  return (
    <div className={styles.field}>
      <p className={styles.label}>{label}</p>
      <Skeleton height="38px" />
    </div>
  );
}

/** A field whose choices could not be loaded: what failed and Try again. */
export function FieldLoadFailed({
  label,
  message,
  onRetry,
}: {
  label: string;
  message: string;
  onRetry: () => void;
}) {
  return (
    <div className={styles.field}>
      <p className={styles.label}>{label}</p>
      <div className={styles.failed}>
        <p role="alert">{message}</p>
        <Button size="sm" icon={<RotateCw />} onClick={onRetry}>
          Try again
        </Button>
      </div>
    </div>
  );
}
