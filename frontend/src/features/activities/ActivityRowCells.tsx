import { CircleAlert } from 'lucide-react';
import { Link } from 'react-router';
import { StatusChip } from '@/ui/StatusChip/StatusChip';
import { useFormatters } from '@/format/useFormatters';
import { ActivityStatusChip } from '@/ui/ActivityStatusChip/ActivityStatusChip';
import type { ActivitySummary } from './api';
import styles from './ActivityRowCells.module.css';
import { lineItemsAttentionText } from './activity-text';

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
 * (the unassigned planned cost is negative). In the reader's display currency (D-220); in a plan
 * planned in another currency, the plan-currency amount with the display one under it, muted
 * (016 FR-013).
 */
export function ActivityRowCost({ row }: { row: ActivitySummary }) {
  const { money } = useFormatters();
  const overspent = row.unassignedPlannedCost?.amount.startsWith('-') ?? false;
  const plan = row.plannedCost;
  const display = row.display.plannedCost;
  const both = Boolean(plan && display && plan.currency !== display.currency);
  return (
    <span className={overspent ? styles.overspent : undefined}>
      {both ? money(plan) : money(display)}
      {overspent && (
        <>
          {' '}
          <CircleAlert aria-hidden="true" className={styles.mark} />
          <span className="visually-hidden">Line items exceed the planned cost</span>
        </>
      )}
      {both && <span className={styles.beside}>≈ {money(display)}</span>}
    </span>
  );
}
