/**
 * What the notification bell asks the API (feature 008 plan, "Server state"; FR-033 – FR-036):
 * the signed-in person's newest 50 notifications with the unread count, asked again every 30
 * seconds and when the window gets the focus back — so a new one shows within a minute without a
 * push channel (D-160) — and the two ways to mark them read.
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api, call } from '@/api/client';
import type { components } from '@/api/generated/schema';

/** One notification (named so, because the browser already has a Notification type). */
export type AppNotification = components['schemas']['Notification'];
export type NotificationPage = components['schemas']['NotificationPage'];

/** The query key of the bell's list; the unread badge shares it. */
export const NOTIFICATIONS_KEY = ['notifications'];

/** How often the list is asked again: within FR-036's minute. */
const POLL_MS = 30_000;

/** The newest 50 notifications, newest first, with the total and the unread count. */
export function useNotifications() {
  return useQuery({
    queryKey: NOTIFICATIONS_KEY,
    queryFn: () => call(api.GET('/api/v1/notifications')),
    refetchInterval: POLL_MS,
    refetchOnWindowFocus: true,
  });
}

/**
 * Marks one notification read. The list shows it read at once (the person is already on their
 * way to the activity); if the API refuses, it shows unread again.
 */
export function useMarkRead() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (notificationId: string) =>
      call(
        api.POST('/api/v1/notifications/{notificationId}/read', {
          params: { path: { notificationId } },
        }),
      ),
    onMutate: async (notificationId) => {
      await queryClient.cancelQueries({ queryKey: NOTIFICATIONS_KEY });
      const before = queryClient.getQueryData<NotificationPage>(NOTIFICATIONS_KEY);
      if (before) {
        queryClient.setQueryData(NOTIFICATIONS_KEY, withOneRead(before, notificationId));
      }
      return { before };
    },
    onError: (_error, _notificationId, context) => {
      if (context?.before) {
        queryClient.setQueryData(NOTIFICATIONS_KEY, context.before);
      }
    },
    onSettled: () => queryClient.invalidateQueries({ queryKey: NOTIFICATIONS_KEY }),
  });
}

/** Marks every notification of the person read; the items stay listed. */
export function useMarkAllRead() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => call(api.POST('/api/v1/notifications/read-all')),
    onSuccess: () => {
      queryClient.setQueryData<NotificationPage>(NOTIFICATIONS_KEY, (page) =>
        page ? withAllRead(page) : page,
      );
    },
    onSettled: () => queryClient.invalidateQueries({ queryKey: NOTIFICATIONS_KEY }),
  });
}

/** The page with one notification read and the unread count one lower (if it was unread). */
function withOneRead(page: NotificationPage, notificationId: string): NotificationPage {
  const target = page.items.find((item) => item.id === notificationId);
  if (!target || target.read) {
    return page;
  }
  return {
    ...page,
    unread: Math.max(0, page.unread - 1),
    items: page.items.map((item) => (item.id === notificationId ? { ...item, read: true } : item)),
  };
}

function withAllRead(page: NotificationPage): NotificationPage {
  return { ...page, unread: 0, items: page.items.map((item) => ({ ...item, read: true })) };
}
