import {
  Bell,
  CircleArrowUp,
  CircleCheckBig,
  CircleX,
  Clock,
  RotateCcw,
  RotateCw,
  Undo2,
  Zap,
} from 'lucide-react';
import type { LucideIcon } from 'lucide-react';
import { useNavigate } from 'react-router';
import { Button } from '@/ui/Button/Button';
import { Skeleton } from '@/ui/Skeleton/Skeleton';
import { useToast } from '@/ui/Toast/useToast';
import { asApiError, isSessionLost } from '@/features/approvals';
import { cx } from '@/ui/cx';
import { recordHref } from '@/ui/record-links';
import { useFormatters } from '@/format/useFormatters';
import { useMarkAllRead, useMarkRead, useNotifications } from './notificationsApi';
import type { AppNotification } from './notificationsApi';
import styles from './NotificationList.module.css';

/**
 * The icon of each kind of notification (008 plan, "Notification bell"; 017 plan, "Notifications":
 * approved automatically, a reminder, an escalation).
 */
const KIND_ICONS: Record<AppNotification['kind'], LucideIcon> = {
  approvalNeeded: Bell,
  levelApproved: CircleCheckBig,
  approved: CircleCheckBig,
  declined: CircleX,
  recalled: Undo2,
  restarted: RotateCcw,
  autoApproved: Zap,
  escalated: CircleArrowUp,
  reminder: Clock,
};

export interface NotificationListProps {
  /** The id of the popover's heading, which names it. */
  titleId: string;
  /** Closes the bell's popover (after choosing a notification). */
  onClose: () => void;
}

/**
 * The content of the notification bell's popover (US6, FR-033, FR-034): the heading, **Mark all
 * as read** while some are unread, and the newest 50 notifications, newest first. Choosing one
 * marks it read and opens the activity's Approval section — or the recycle bin when the activity
 * was deleted.
 */
export function NotificationList({ titleId, onClose }: NotificationListProps) {
  const notifications = useNotifications();
  const markAllRead = useMarkAllRead();
  const markRead = useMarkRead();
  const navigate = useNavigate();
  const { show } = useToast();
  const page = notifications.data;

  function readAll() {
    if (markAllRead.isPending) {
      return;
    }
    markAllRead.mutate(undefined, {
      onError: (error) => {
        const failure = asApiError(error);
        if (!isSessionLost(failure)) {
          show({ message: failure.message });
        }
      },
    });
  }

  function choose(notification: AppNotification) {
    if (!notification.read) {
      markRead.mutate(notification.id);
    }
    onClose();
    void navigate(
      notification.activity.exists
        ? `${recordHref('activity', notification.activity.id)}#approval`
        : '/recycle-bin',
    );
  }

  return (
    <>
      <div className={styles.header}>
        <h2 id={titleId} className={styles.title}>
          Notifications
        </h2>
        {page && page.unread > 0 && (
          <Button
            variant="tertiary"
            size="sm"
            aria-busy={markAllRead.isPending || undefined}
            onClick={readAll}
          >
            Mark all as read
          </Button>
        )}
      </div>
      {page ? (
        page.items.length === 0 ? (
          <p className={styles.empty}>You're all caught up.</p>
        ) : (
          <ul className={styles.list}>
            {page.items.map((notification) => (
              <li key={notification.id}>
                <NotificationItem notification={notification} onChoose={choose} />
              </li>
            ))}
          </ul>
        )
      ) : notifications.isError ? (
        <div className={styles.error}>
          <p role="alert">Notifications couldn't be loaded.</p>
          <Button size="sm" icon={<RotateCw />} onClick={() => void notifications.refetch()}>
            Try again
          </Button>
        </div>
      ) : (
        <div className={styles.loading}>
          <Skeleton lines={3} height="16px" />
        </div>
      )}
    </>
  );
}

interface NotificationItemProps {
  notification: AppNotification;
  onChoose: (notification: AppNotification) => void;
}

/**
 * One notification as a button: its icon, title, the line naming the activity and the actor, the
 * relative time (the full date and time on hover) and, while unread, a dot with the hidden word
 * "Unread". The spaces between the parts keep the words apart in the button's name.
 */
function NotificationItem({ notification, onChoose }: NotificationItemProps) {
  const { dateTime, relativeTime } = useFormatters();
  const Icon = KIND_ICONS[notification.kind];

  return (
    <button
      type="button"
      className={cx(styles.item, !notification.read && styles.unread)}
      onClick={() => onChoose(notification)}
    >
      <Icon className={cx(styles.icon, styles[notification.kind])} aria-hidden="true" />
      <span className={styles.main}>
        <span className={styles.itemTitle}>{notification.title}</span>{' '}
        <span className={styles.text}>{notification.text}</span>{' '}
        {!notification.activity.exists && (
          <>
            <span className={styles.deleted}>
              This activity was deleted — find it in the recycle bin.
            </span>{' '}
          </>
        )}
        <time className={styles.time} dateTime={notification.at} title={dateTime(notification.at)}>
          {relativeTime(notification.at)}
        </time>
      </span>
      {!notification.read && (
        <>
          {' '}
          <span className={styles.dot}>
            <span className="visually-hidden">Unread</span>
          </span>
        </>
      )}
    </button>
  );
}
