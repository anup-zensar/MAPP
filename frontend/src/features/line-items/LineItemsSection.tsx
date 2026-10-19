import { BuildingSection } from '@/features/placeholders/BuildingPage';

export interface LineItemsSectionProps {
  activityId: string;
}

/**
 * The Line items section of feature 006's activity page (007 plan, "Line items section"): the
 * rows, the unassigned planned cost and the spread grid — STUB of the shared foundation. Feature
 * 007 replaces this file; 006's ActivityPage already places it inside the element with id
 * "line-items".
 */
export function LineItemsSection({ activityId }: LineItemsSectionProps) {
  void activityId;
  return <BuildingSection title="Line items" />;
}
