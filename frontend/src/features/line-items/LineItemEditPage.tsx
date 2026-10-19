import { useParams } from 'react-router';
import { BuildingPage } from '@/features/placeholders/BuildingPage';

/**
 * Edit line item (feature 007, US4, FR-027), at /line-items/:lineItemId/edit (?field=<field>
 * focuses a field) — STUB of the shared foundation. Feature 007 replaces this file; the route
 * stays.
 */
export function LineItemEditPage() {
  const { lineItemId = '' } = useParams();
  return (
    <BuildingPage
      eyebrow="Line item"
      title="Edit line item"
      description={lineItemId}
      breadcrumb={[
        { label: 'Plans', to: '/plans' },
        { label: lineItemId, to: `/line-items/${lineItemId}` },
        { label: 'Edit' },
      ]}
    />
  );
}
