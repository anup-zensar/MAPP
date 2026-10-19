import { BuildingSection } from '@/features/placeholders/BuildingPage';

export interface InitiativeActivitiesProps {
  planId: string;
  initiativeId: string;
}

/**
 * The activities of an initiative in feature 005's initiative details panel, with **Add activity**
 * (006 plan, "Plan tree rows and Add activity") — STUB of the shared foundation. Feature 006
 * replaces this file; 005's InitiativePanel places it through extensions.ts.
 */
export function InitiativeActivities({ planId, initiativeId }: InitiativeActivitiesProps) {
  void planId;
  void initiativeId;
  return <BuildingSection title="Activities" />;
}
