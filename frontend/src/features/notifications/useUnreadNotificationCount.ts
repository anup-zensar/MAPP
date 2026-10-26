import { useNotifications } from './api';

/**
 * How many notifications the signed-in person has not read (FR-033), from the bell's list query
 * (asked again every 30 s): 001's NotificationBell shows it as a CountBadge and in the bell's name
 * ("Notifications, 3 unread"). 0 until the list has arrived.
 */
export function useUnreadNotificationCount(): number {
  return useNotifications().data?.unread ?? 0;
}
