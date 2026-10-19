import { Bell } from 'lucide-react';
import { Popover } from 'radix-ui';
import { useId } from 'react';
import { Button, Tooltip } from '@/components';
import styles from './NotificationBell.module.css';

export interface NotificationBellProps {
  /**
   * Writes "Notifications" next to the bell instead of showing a tooltip — in the menu drawer,
   * where a tooltip would swallow the first Escape (research R14).
   */
  withLabel?: boolean;
}

/**
 * The bell of the sidebar and the top bar (FR-013). Until feature 008 delivers notifications,
 * its popover only says "You're all caught up.".
 */
export function NotificationBell({ withLabel = false }: NotificationBellProps) {
  const titleId = useId();

  const trigger = (
    <Popover.Trigger asChild>
      <Button
        variant="ghost"
        icon={<Bell />}
        aria-label={withLabel ? undefined : 'Notifications'}
        className={withLabel ? styles.labelled : undefined}
      >
        {withLabel ? 'Notifications' : undefined}
      </Button>
    </Popover.Trigger>
  );

  return (
    <Popover.Root>
      {withLabel ? trigger : <Tooltip content="Notifications">{trigger}</Tooltip>}
      <Popover.Portal>
        <Popover.Content
          className={styles.popover}
          aria-labelledby={titleId}
          align="start"
          sideOffset={8}
        >
          <h2 id={titleId} className={styles.title}>
            Notifications
          </h2>
          <p className={styles.empty}>You're all caught up.</p>
        </Popover.Content>
      </Popover.Portal>
    </Popover.Root>
  );
}
