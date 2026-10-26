import { StatusChip } from '@/ui/StatusChip/StatusChip';
import { activityStatusTone, activityStatusWord } from './activityStatus';
import type { ActivityDisplayStatus } from './activityStatus';

export interface ActivityStatusChipProps {
  /** The display status the API answers (an approved activity may be In progress or Complete). */
  status: ActivityDisplayStatus;
}

/**
 * An activity's status chip (feature 006, FR-021): its word on its colour family — on the
 * activity page, the tree rows and feature 008's approvals pages.
 */
export function ActivityStatusChip({ status }: ActivityStatusChipProps) {
  return <StatusChip tone={activityStatusTone(status)}>{activityStatusWord(status)}</StatusChip>;
}
