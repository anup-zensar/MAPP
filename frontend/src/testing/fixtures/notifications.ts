import type { AppNotification, NotificationPage } from '@/features/notifications';

export const NOW = new Date('2026-03-12T09:00:00Z');

const LAUNCH_FILM = { id: 'A-1042', name: 'Launch film', exists: true };

/** An activity notification: its subject follows the activity a test gives, unless the test gives one. */
export function notificationOf(overrides: Partial<AppNotification> = {}): AppNotification {
  const activity = overrides.activity === undefined ? LAUNCH_FILM : overrides.activity;
  const subject = activity
    ? { kind: 'activity' as const, label: 'Activity', ...activity }
    : { kind: 'purchaseRequest' as const, label: 'Purchase request', ...LAUNCH_FILM };
  return {
    id: 'NT-1003',
    kind: 'approvalNeeded',
    title: 'Approval needed',
    text: "Priya Raman submitted 'Launch film' (A-1042)",
    actor: { id: 'U-1002', name: 'Priya Raman' },
    at: '2026-03-12T08:55:00Z',
    read: false,
    ...overrides,
    activity,
    subject: overrides.subject ?? subject,
  };
}

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

export function pageOf(items: AppNotification[], total = items.length): NotificationPage {
  return { items, total, unread: items.filter((item) => !item.read).length };
}
