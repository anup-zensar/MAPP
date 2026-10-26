import { recordHref } from '@/ui/record-links';
import type { AllWorkRow } from './allWorkTypes';

/**
 * The page a row opens (FR-016), through 004's recordHref (D-194): the record's own page or
 * panel; an unassigned planned cost row opens its activity.
 */
export function rowHref(row: Pick<AllWorkRow, 'level' | 'recordId' | 'planId'>): string {
  return row.level === 'unassigned'
    ? recordHref('activity', row.recordId, row.planId)
    : recordHref(row.level, row.recordId, row.planId);
}
