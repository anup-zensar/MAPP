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
import { cx } from '@/ui/cx';
import { subjectHref } from '@/ui/subject-links';
import { useFormatters } from '@/format/useFormatters';
import { asApiError, isSessionLost } from '@/api/errors';
import { useMarkAllRead, useMarkRead, useNotifications } from './api';
import type { AppNotification } from './api';
import styles from './NotificationList.module.css';

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
  titleId: string;
  onClose: () => void;
}

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
    const { subject } = notification;
    void navigate(
      subject.exists ? `${subjectHref(subject.kind, subject.id)}#approval` : '/recycle-bin',
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
        {!notification.subject.exists && (
          <>
            <span className={styles.deleted}>
              {`This ${notification.subject.label.toLowerCase()} was deleted — find it in the recycle bin.`}
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
