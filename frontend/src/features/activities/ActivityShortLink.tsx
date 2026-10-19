import { useParams } from 'react-router';
import { BuildingPage } from '@/features/placeholders/BuildingPage';

/**
 * The short address /activities/:activityId (D-194) for links that know only the activity (007 –
 * 011): feature 006 replaces this STUB of the shared foundation with a page that finds the
 * activity's plan and replaces the address with /plans/:planId/activities/:activityId, keeping the
 * #line-items or #approval part. The route stays.
 */
export function ActivityShortLink() {
  const { activityId = '' } = useParams();
  return (
    <BuildingPage
      eyebrow="Activity"
      title="Activity"
      description={activityId}
      breadcrumb={[{ label: 'Plans', to: '/plans' }, { label: activityId }]}
    />
  );
}
