import { useParams } from 'react-router';
import { BuildingPage } from '@/features/placeholders/BuildingPage';

/**
 * New line item (feature 007, US2), at /activities/:activityId/line-items/new — STUB of the
 * shared foundation. Feature 007 replaces this file; the route stays.
 */
export function LineItemCreatePage() {
  const { activityId = '' } = useParams();
  return (
    <BuildingPage
      eyebrow="Line item"
      title="New line item"
      description={`In activity ${activityId}`}
      breadcrumb={[
        { label: 'Plans', to: '/plans' },
        { label: activityId, to: `/activities/${activityId}` },
        { label: 'New line item' },
      ]}
    />
  );
}
