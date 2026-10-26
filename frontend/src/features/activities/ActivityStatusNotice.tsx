import type { ActivityApproval } from '@/features/approvals';
import { declineBoxText, levelOf, names } from '@/features/approvals';
import { Notice } from '@/features/line-items';
import { useFormatters } from '@/format/useFormatters';
import { activityBannerText } from '@/ui/ActivityStatusChip/activity-status';
import type { Activity } from './api';

export interface ActivityStatusNoticeProps {
  activity: Activity;
  /** The activity's approval (feature 008): the latest decline and the waiting level. */
  approval: ActivityApproval | undefined;
}

/**
 * The notice directly under an activity's header (design system §2.4 B, UI audit AC-02):
 *
 * - Declined: "Declined by Daniel Okoro on 15 Oct 2026, 14:41" and the reason, in the danger
 *   colour, visible without scrolling — never buried in the Approval section;
 * - Submitted: who it waits for, and that changing planned cost or dates restarts approval;
 * - Approved, In progress, Complete: that changing planned cost or dates restarts approval (FR-027).
 */
export function ActivityStatusNotice({ activity, approval }: ActivityStatusNoticeProps) {
  const { dateFormat } = useFormatters();
  const decline = approval?.latestDecline;

  if (activity.status === 'declined') {
    if (!decline) {
      return null;
    }
    return (
      <Notice tone="danger" title={declineBoxText(decline, dateFormat)}>
        {decline.comment && <p>{decline.comment}</p>}
        <p>
          Change what is needed, then re-submit it for approval.{' '}
          <a href="#approval">See the approval history</a>
        </p>
      </Notice>
    );
  }

  const banner = activityBannerText(activity.displayStatus);
  if (!banner) {
    return null;
  }
  if (activity.status === 'submitted') {
    const round = approval?.rounds[0];
    const level = round?.levels.find((each) => each.number === round.currentLevel);
    return (
      <Notice
        tone="warning"
        title={
          round && level
            ? `Waiting for ${levelOf(round.currentLevel, round.levelCount)}${
                level.approvers.length > 0 ? ` — ${names(level.approvers)}` : ''
              }`
            : undefined
        }
      >
        <p>{banner}</p>
      </Notice>
    );
  }
  return (
    <Notice tone="info">
      <p>{banner}</p>
    </Notice>
  );
}
