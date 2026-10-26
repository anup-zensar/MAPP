import { useParams } from 'react-router';
import { useInitiative } from './api';
import { RecordPanel } from './RecordPanel';

export interface InitiativePanelProps {
  /** details: /plans/:planId/initiatives/:initiativeId; edit: …/edit (?focus=<field>). */
  mode: 'details' | 'edit';
}

/**
 * An initiative's details or edit form (US3), a side panel over the plan page. The details show
 * feature 006's activities of the initiative (extensions.ts).
 */
export function InitiativePanel({ mode }: InitiativePanelProps) {
  const { planId = '', initiativeId = '' } = useParams();
  const query = useInitiative(initiativeId);
  return <RecordPanel level="initiative" mode={mode} planId={planId} query={query} />;
}
