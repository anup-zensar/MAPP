import { Bell } from 'lucide-react';
import { Popover } from 'radix-ui';
import { useId, useState } from 'react';
import { Button, CountBadge, Tooltip } from '@/components';
import { NotificationList } from '@/features/notifications/NotificationList';
import { useUnreadNotificationCount } from '@/features/notifications/useUnreadNotificationCount';
import { cx } from '@/lib/cx';
import styles from './NotificationBell.module.css';

export interface NotificationBellProps {
  /**
   * Writes "Notifications" next to the bell instead of showing a tooltip — in the menu drawer,
   * where a tooltip would swallow the first Escape (research R14).
   */
  withLabel?: boolean;
}

/**
 * The bell of the sidebar and the top bar (FR-013). Feature 008 fills it from its own folder
 * (features/notifications): the unread count on the bell and in its name ("Notifications, 3
 * unread"), and the list in the popover.
 */
export function NotificationBell({ withLabel = false }: NotificationBellProps) {
  const titleId = useId();
  const [open, setOpen] = useState(false);
  const unread = useUnreadNotificationCount();
  const name = unread > 0 ? `Notifications, ${unread} unread` : 'Notifications';

  const trigger = (
    <Popover.Trigger asChild>
      <Button
        variant="ghost"
        icon={<Bell />}
        // The written word names it in the drawer, unless there is a count to add.
        aria-label={withLabel && unread === 0 ? undefined : name}
        className={withLabel ? styles.labelled : undefined}
      >
        {withLabel ? 'Notifications' : undefined}
      </Button>
    </Popover.Trigger>
  );

  return (
    <Popover.Root open={open} onOpenChange={setOpen}>
      <span className={cx(styles.bell, withLabel && styles.labelledBell)}>
        {withLabel ? trigger : <Tooltip content="Notifications">{trigger}</Tooltip>}
        {unread > 0 && (
          // The bell's name says the count already ("Notifications, 3 unread").
          <span className={styles.count} aria-hidden="true">
            <CountBadge count={unread} label="unread" max={9} />
          </span>
        )}
      </span>
      <Popover.Portal>
        <Popover.Content
          className={styles.popover}
          aria-labelledby={titleId}
          align="start"
          sideOffset={8}
        >
          <NotificationList titleId={titleId} onClose={() => setOpen(false)} />
        </Popover.Content>
      </Popover.Portal>
    </Popover.Root>
  );
}
