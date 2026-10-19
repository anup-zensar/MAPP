import { useParams } from 'react-router';
import { BuildingPage } from '@/features/placeholders/BuildingPage';

/**
 * Edit plan (feature 004, US4), at /plans/:planId/edit (?focus=<field> focuses a field) — STUB of
 * the shared foundation. Feature 004 replaces this file with the real page; the route stays.
 */
export function PlanEditPage() {
  const { planId = '' } = useParams();
  return (
    <BuildingPage
      eyebrow="Plans"
      title="Edit plan"
      description={planId}
      breadcrumb={[
        { label: 'Plans', to: '/plans' },
        { label: planId, to: `/plans/${planId}` },
        { label: 'Edit' },
      ]}
    />
  );
}
