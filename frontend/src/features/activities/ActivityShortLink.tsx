import { Navigate, useLocation, useParams } from 'react-router';
import { Breadcrumb, PageHeader } from '@/components';
import { usePageTitle } from '@/lib/usePageTitle';
import { useActivity } from './activitiesApi';
import { LoadFailed, PageSkeleton } from './ActivityLoadStates';
import { activityHref } from './activityText';

/**
 * The short address /activities/:activityId (D-194) for links that know only the activity (007 –
 * 011): finds the activity's plan and replaces the address with
 * /plans/:planId/activities/:activityId, keeping the #line-items or #approval part.
 */
export function ActivityShortLink() {
  const { activityId = '' } = useParams();
  const location = useLocation();
  const query = useActivity(activityId);
  usePageTitle('Activity');

  if (query.isSuccess) {
    const { plan, id } = query.data;
    return <Navigate replace to={`${activityHref(plan.id, id)}${location.hash}`} />;
  }
  return (
    <>
      <Breadcrumb items={[{ label: 'Plans', to: '/plans' }, { label: 'Activity' }]} />
      <PageHeader eyebrow="Activity" title="Activity" />
      {query.isPending ? (
        <PageSkeleton />
      ) : (
        <LoadFailed
          error={query.error}
          notFoundText="This activity no longer exists — someone may have deleted it."
          onRetry={() => void query.refetch()}
        />
      )}
    </>
  );
}
