import type { PlanningRecordType } from '@/ui/record-links';

const PLANNING_TYPES: readonly PlanningRecordType[] = [
  'plan',
  'campaign',
  'initiative',
  'activity',
  'lineItem',
];

/** Whether a record reference's type (a string in the contract) is a level of the plan. */
export function isPlanningType(type: string): type is PlanningRecordType {
  return PLANNING_TYPES.some((known) => known === type);
}
