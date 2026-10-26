/**
 * Notifications for feature 008's bell tests, shaped exactly as the contract answers them. The
 * tests run at 12 Mar 2026, 09:00 UTC (the seed's clock); tests change only what they need:
 *
 *   notificationOf({ read: true })
 */
import type { AppNotification, NotificationPage } from './notificationsApi';

/** The tests' "now": 12 Mar 2026, 09:00 UTC. */
export const NOW = new Date('2026-03-12T09:00:00Z');

/** Daniel's: Priya submitted "Launch film" five minutes ago; unread. */
export function notificationOf(overrides: Partial<AppNotification> = {}): AppNotification {
  return {
    id: 'NT-1003',
    kind: 'approvalNeeded',
    title: 'Approval needed',
    text: "Priya Raman submitted 'Launch film' (A-1042)",
    activity: { id: 'A-1042', name: 'Launch film', exists: true },
    actor: { id: 'U-1002', name: 'Priya Raman' },
    at: '2026-03-12T08:55:00Z',
    read: false,
    ...overrides,
  };
}

/** Priya's: Daniel declined "Spring promo" yesterday morning; read. */
export const DECLINED = notificationOf({
  id: 'NT-1002',
  kind: 'declined',
  title: 'Activity declined',
  text: 'Daniel Okoro declined \'Spring promo\' (A-1031): "The agency fee is missing its GL code."',
  activity: { id: 'A-1031', name: 'Spring promo', exists: true },
  actor: { id: 'U-1003', name: 'Daniel Okoro' },
  at: '2026-03-11T08:00:00Z',
  read: true,
});

/** The answer of GET /api/v1/notifications for these items, counting the unread ones. */
export function pageOf(items: AppNotification[], total = items.length): NotificationPage {
  return { items, total, unread: items.filter((item) => !item.read).length };
}
