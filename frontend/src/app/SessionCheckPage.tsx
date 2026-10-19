import { ErrorState, Skeleton } from '@/components';
import { usePageTitle } from '@/lib/usePageTitle';
import { PublicLayout } from './PublicLayout';
import styles from './SessionCheckPage.module.css';
import { useSession } from './useSession';

export interface SessionCheckPageProps {
  /** The name of the page that was asked for, kept in the browser tab meanwhile. */
  title: string;
}

/**
 * The whole page while the session is being checked, or when the check could not reach the API
 * (plan "Session"): the public frame — banner, no sidebar — with a loading placeholder, or the
 * error with Try again, which checks again. There is no redirect: the address stays.
 */
export function SessionCheckPage({ title }: SessionCheckPageProps) {
  const { status, error, refetch } = useSession();
  usePageTitle(title);

  return (
    <PublicLayout>
      <div className={styles.page}>
        {status === 'unreachable' ? (
          <ErrorState error={error} onRetry={refetch} />
        ) : (
          <Skeleton lines={4} height="18px" />
        )}
      </div>
    </PublicLayout>
  );
}
