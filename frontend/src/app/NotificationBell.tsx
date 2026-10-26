import { Bell } from 'lucide-react';
import { Popover } from 'radix-ui';
import { useId, useState } from 'react';
import { Button } from '@/ui/Button/Button';
import { CountBadge } from '@/ui/CountBadge/CountBadge';
import { Tooltip } from '@/ui/Tooltip/Tooltip';
import { NotificationList } from '@/features/notifications/NotificationList';
import { useUnreadNotificationCount } from '@/features/notifications/useUnreadNotificationCount';
import styles from './NotificationBell.module.css';

/**
 * The bell of the top bar (FR-013). Feature 008 fills it from its own folder
 * (features/notifications): the unread count on the bell and in its name ("Notifications, 3
 * unread"), and the list in the popover.
 */
export function NotificationBell() {
  const titleId = useId();
  const [open, setOpen] = useState(false);
  const unread = useUnreadNotificationCount();
  const name = unread > 0 ? `Notifications, ${unread} unread` : 'Notifications';

  return (
    <Popover.Root open={open} onOpenChange={setOpen}>
      <span className={styles.bell}>
        <Tooltip content="Notifications">
          <Popover.Trigger asChild>
            <Button variant="tertiary" icon={<Bell />} aria-label={name} />
          </Popover.Trigger>
        </Tooltip>
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
          align="end"
          sideOffset={8}
        >
          <NotificationList titleId={titleId} onClose={() => setOpen(false)} />
        </Popover.Content>
      </Popover.Portal>
    </Popover.Root>
  );
}
