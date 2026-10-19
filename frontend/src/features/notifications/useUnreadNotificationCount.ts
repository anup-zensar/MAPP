/**
 * How many notifications the signed-in person has not read (008 plan, "Notification bell") —
 * STUB of the shared foundation: 0, until feature 008 replaces this file with the count from its
 * notifications query (GET /api/notifications, polled every 30 s). 001's NotificationBell already
 * shows it as a CountBadge and in the bell's name ("Notifications, 3 unread").
 */
export function useUnreadNotificationCount(): number {
  return 0;
}
