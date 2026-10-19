import { CircleAlert } from 'lucide-react';
import { Link } from 'react-router';
import { StatusChip } from '@/components';
import { useFormatters } from '@/lib/useFormatters';
import type { ActivitySummary } from './activitiesApi';
import styles from './ActivityRowCells.module.css';
import { ActivityStatusChip } from './ActivityStatusChip';
import { lineItemsAttentionText } from './activityText';

/**
 * The status cell of an activity row (FR-033): the status chip and the completeness — "All filled
 * in", or "3 fields needed" and "1 line item needs attention" as links to the activity's Ready to
 * submit panel.
 */
export function ActivityRowStatus({ row, href }: { row: ActivitySummary; href: string }) {
  const { missingFields, lineItemsNeedingAttention } = row.completeness;
  return (
    <span className={styles.chips}>
      <ActivityStatusChip status={row.displayStatus} />
      {missingFields === 0 && lineItemsNeedingAttention === 0 && (
        <StatusChip tone="positive">All filled in</StatusChip>
      )}
      {missingFields > 0 && (
        <Link to={`${href}#ready`} className={styles.chipLink}>
          <StatusChip tone="problem">
            {missingFields === 1 ? '1 field needed' : `${missingFields} fields needed`}
          </StatusChip>
        </Link>
      )}
      {lineItemsNeedingAttention > 0 && (
        <Link to={`${href}#ready`} className={styles.chipLink}>
          <StatusChip tone="problem">
            {lineItemsAttentionText(lineItemsNeedingAttention)}
          </StatusChip>
        </Link>
      )}
    </span>
  );
}

/**
 * The planned-cost cell of an activity row, with a red mark when its line items exceed it
 * (the unassigned planned cost is negative).
 */
export function ActivityRowCost({ row }: { row: ActivitySummary }) {
  const { money } = useFormatters();
  const overspent = row.unassignedPlannedCost?.amount.startsWith('-') ?? false;
  return (
    <span className={overspent ? styles.overspent : undefined}>
      {money(row.plannedCost)}
      {overspent && (
        <>
          {' '}
          <CircleAlert aria-hidden="true" className={styles.mark} />
          <span className="visually-hidden">Line items exceed the planned cost</span>
        </>
      )}
    </span>
  );
}
