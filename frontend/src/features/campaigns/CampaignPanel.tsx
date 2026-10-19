import { useParams } from 'react-router';
import { BuildingPanel } from '@/features/placeholders/BuildingPage';

export interface CampaignPanelProps {
  /** details: /plans/:planId/campaigns/:campaignId; edit: …/edit (?focus=<field>). */
  mode: 'details' | 'edit';
}

/**
 * A campaign's details or edit form (feature 005, US3), a side panel over the plan page — STUB of
 * the shared foundation. Feature 005 replaces this file; the routes stay.
 */
export function CampaignPanel({ mode }: CampaignPanelProps) {
  const { planId = '', campaignId = '' } = useParams();
  return (
    <BuildingPanel
      title={mode === 'edit' ? 'Edit campaign' : 'Campaign'}
      description={campaignId}
      closeTo={`/plans/${planId}`}
      closeLabel="Back to the plan"
    />
  );
}
