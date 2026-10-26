import { useParams } from 'react-router';
import { useCampaign } from './api';
import { RecordPanel } from './RecordPanel';

export interface CampaignPanelProps {
  /** details: /plans/:planId/campaigns/:campaignId; edit: …/edit (?focus=<field>). */
  mode: 'details' | 'edit';
}

/** A campaign's details or edit form (US3, FR-016 – FR-018), a side panel over the plan page. */
export function CampaignPanel({ mode }: CampaignPanelProps) {
  const { planId = '', campaignId = '' } = useParams();
  const query = useCampaign(campaignId);
  return <RecordPanel level="campaign" mode={mode} planId={planId} query={query} />;
}
