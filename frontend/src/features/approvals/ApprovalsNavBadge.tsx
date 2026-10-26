import { CountBadge } from '@/ui/CountBadge/CountBadge';
import { cx } from '@/lib/cx';
import { useApprovalSummary } from './useApprovalSummary';
import styles from './ApprovalsNavBadge.module.css';

export interface ApprovalsNavBadgeProps {
  /** The sidebar is the icon rail: the badge sits on the icon. */
  collapsed: boolean;
}

/**
 * The count of activities waiting for the signed-in person on the sidebar's Approvals item
 * (FR-027): inside the link, so its name becomes "Approvals 3 waiting for you"; nothing at 0.
 */
export function ApprovalsNavBadge({ collapsed }: ApprovalsNavBadgeProps) {
  const summary = useApprovalSummary();
  const count = summary.data?.waitingForMe ?? 0;
  if (count <= 0) {
    return null;
  }
  return (
    <span className={cx(styles.badge, collapsed && styles.onIcon)}>
      <CountBadge count={count} label="waiting for you" />
    </span>
  );
}
