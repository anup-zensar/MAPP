import { useParams } from 'react-router';
import { BuildingPanel } from '@/features/placeholders/BuildingPage';
import { InitiativeActivitiesSection } from './extensions';

export interface InitiativePanelProps {
  /** details: /plans/:planId/initiatives/:initiativeId; edit: …/edit. */
  mode: 'details' | 'edit';
}

/**
 * An initiative's details or edit form (feature 005, US3), a side panel over the plan page — STUB
 * of the shared foundation. Feature 005 replaces this file; the routes stay. The details show
 * feature 006's activities of the initiative (extensions.ts).
 */
export function InitiativePanel({ mode }: InitiativePanelProps) {
  const { planId = '', initiativeId = '' } = useParams();
  return (
    <BuildingPanel
      title={mode === 'edit' ? 'Edit initiative' : 'Initiative'}
      description={initiativeId}
      closeTo={`/plans/${planId}`}
      closeLabel="Back to the plan"
    >
      {mode === 'details' && (
        <InitiativeActivitiesSection planId={planId} initiativeId={initiativeId} />
      )}
    </BuildingPanel>
  );
}
