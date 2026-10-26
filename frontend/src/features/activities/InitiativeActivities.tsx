import { Plus } from 'lucide-react';
import { Link } from 'react-router';
import { LinkButton } from '@/ui/Button/Button';
import { Card } from '@/ui/Card/Card';
import { ErrorState } from '@/ui/ErrorState/ErrorState';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { usePlanActivities } from './activitiesApi';
import { ActivityStatusChip } from '@/ui/ActivityStatusChip/ActivityStatusChip';
import { activityHref, newActivityHref } from './activityText';
import styles from './InitiativeActivities.module.css';

export interface InitiativeActivitiesProps {
  planId: string;
  initiativeId: string;
}

/**
 * The activities of an initiative in feature 005's initiative details panel (006 plan, "Plan tree
 * rows and Add activity", FR-001): each name a link to its page with its status chip, and **Add
 * activity**. 005's InitiativePanel places it through extensions.ts.
 */
export function InitiativeActivities({ planId, initiativeId }: InitiativeActivitiesProps) {
  const query = usePlanActivities(planId);
  const activities = query.data?.items.filter((item) => item.initiativeId === initiativeId);

  return (
    <Card
      title="Activities"
      actions={
        <LinkButton to={newActivityHref(planId, initiativeId)} size="sm" icon={<Plus />}>
          Add activity
        </LinkButton>
      }
    >
      {query.isPending && <Skeleton lines={3} />}
      {query.isError && <ErrorState error={query.error} onRetry={() => void query.refetch()} />}
      {activities && activities.length === 0 && <p className={styles.empty}>No activities yet.</p>}
      {activities && activities.length > 0 && (
        <ul className={styles.list}>
          {activities.map((activity) => (
            <li key={activity.id} className={styles.item}>
              <Link to={activityHref(planId, activity.id)} className={styles.name}>
                {activity.name}
              </Link>
              <ActivityStatusChip status={activity.displayStatus} />
            </li>
          ))}
        </ul>
      )}
    </Card>
  );
}
