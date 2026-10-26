import { SearchX } from 'lucide-react';
import { ApiError } from '@/api/errors';
import { LinkButton } from '@/ui/Button/Button';
import { Card } from '@/ui/Card/Card';
import { EmptyState } from '@/ui/EmptyState/EmptyState';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import styles from './ActivityLoadStates.module.css';

export interface LoadFailedProps {
  error: unknown;
  /** Said when the record is gone (404): "This activity no longer exists — …". */
  notFoundText: string;
  onRetry: () => void;
}

/**
 * What an activity page shows when its data did not load (006 plan, "States of every page"): the
 * not-found sentence with **Go to Plans**, or the failure with **Try again**.
 */
export function LoadFailed({ error, notFoundText, onRetry }: LoadFailedProps) {
  if (error instanceof ApiError && error.kind === 'notFound') {
    return (
      <Card>
        <EmptyState
          icon={<SearchX />}
          title={notFoundText}
          action={
            <LinkButton to="/plans" variant="primary">
              Go to Plans
            </LinkButton>
          }
        />
      </Card>
    );
  }
  return (
    <Card>
      <ErrorState error={error} onRetry={onRetry} />
    </Card>
  );
}

/** Grey shapes in the form's layout while it loads. */
export function FormSkeleton() {
  return (
    <div className={styles.form}>
      <Skeleton width="40%" height="20px" />
      <Skeleton lines={6} height="36px" />
      <Skeleton width="40%" height="20px" />
      <Skeleton lines={3} height="36px" />
    </div>
  );
}

/** Grey shapes of the activity page while it loads: the header and two cards. */
export function PageSkeleton() {
  return (
    <div className={styles.page}>
      <Skeleton width="60%" height="28px" lines={2} />
      <Card>
        <Skeleton lines={4} />
      </Card>
      <Card>
        <Skeleton lines={4} />
      </Card>
    </div>
  );
}
