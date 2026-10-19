import { useParams } from 'react-router';
import { BuildingPanel } from '@/features/placeholders/BuildingPage';

/**
 * New initiative (feature 005, US2), a side panel over the plan page at
 * /plans/:planId/campaigns/:campaignId/initiatives/new — STUB of the shared foundation. Feature
 * 005 replaces this file; the route stays.
 */
export function NewInitiativePanel() {
  const { planId = '' } = useParams();
  return (
    <BuildingPanel
      title="New initiative"
      closeTo={`/plans/${planId}`}
      closeLabel="Back to the plan"
    />
  );
}
