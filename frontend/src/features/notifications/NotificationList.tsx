import styles from './NotificationList.module.css';

export interface NotificationListProps {
  /** The id of the popover's heading, which names it. */
  titleId: string;
  /** Closes the bell's popover (after choosing a notification). */
  onClose: () => void;
}

/**
 * The content of the notification bell's popover (008 plan, "Notification bell"): the heading,
 * Mark all as read and the notifications, newest first — STUB of the shared foundation: the
 * heading and "You're all caught up." as before. Feature 008 replaces this file; 001's
 * NotificationBell already renders it.
 */
export function NotificationList({ titleId, onClose }: NotificationListProps) {
  void onClose;
  return (
    <>
      <h2 id={titleId} className={styles.title}>
        Notifications
      </h2>
      <p className={styles.empty}>You're all caught up.</p>
    </>
  );
}
