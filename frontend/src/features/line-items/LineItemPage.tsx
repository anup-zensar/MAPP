import { useParams } from 'react-router';
import { BuildingPage } from '@/features/placeholders/BuildingPage';

/**
 * A line item's details (feature 007, US4, FR-026), at /line-items/:lineItemId — STUB of the
 * shared foundation. Feature 007 replaces this file; the route stays.
 */
export function LineItemPage() {
  const { lineItemId = '' } = useParams();
  return (
    <BuildingPage
      eyebrow="Line item"
      title="Line item"
      description={lineItemId}
      breadcrumb={[{ label: 'Plans', to: '/plans' }, { label: lineItemId }]}
    />
  );
}
