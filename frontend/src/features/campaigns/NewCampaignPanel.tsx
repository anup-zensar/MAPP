import { useParams } from 'react-router';
import { BuildingPanel } from '@/features/placeholders/BuildingPage';

/**
 * New campaign (feature 005, US1), a side panel over the plan page at
 * /plans/:planId/campaigns/new — STUB of the shared foundation. Feature 005 replaces this file;
 * the route stays.
 */
export function NewCampaignPanel() {
  const { planId = '' } = useParams();
  return (
    <BuildingPanel
      title="New campaign"
      closeTo={`/plans/${planId}`}
      closeLabel="Back to the plan"
    />
  );
}
