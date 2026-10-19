import { act, screen, waitFor, within } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { notificationOf, pageOf } from '@/features/notifications/testData';
import { mockApi } from '@/test/mockApi';
import { renderWithProviders } from '@/test/render';
import { testUser } from '@/test/users';
import { NotificationBell } from './NotificationBell';

describe('NotificationBell', () => {
  it('is a "Notifications" button explained by a tooltip (FR-013)', async () => {
    const { user } = renderWithProviders(<NotificationBell />);

    await user.hover(screen.getByRole('button', { name: 'Notifications' }));

    expect(await screen.findByRole('tooltip')).toHaveTextContent('Notifications');
  });

  it('opens "Notifications", which says "You\'re all caught up." (FR-013, US2)', async () => {
    const { user } = renderWithProviders(<NotificationBell />);

    await user.click(screen.getByRole('button', { name: 'Notifications' }));

    const popover = screen.getByRole('dialog', { name: 'Notifications' });
    expect(await within(popover).findByText("You're all caught up.")).toBeInTheDocument();
  });

  it('closes on Escape and gives the focus back to the bell (FR-013, SC-006)', async () => {
    const { user } = renderWithProviders(<NotificationBell />);
    await user.click(screen.getByRole('button', { name: 'Notifications' }));

    await user.keyboard('{Escape}');

    expect(screen.queryByRole('dialog', { name: 'Notifications' })).not.toBeInTheDocument();
    await waitFor(() =>
      expect(screen.getByRole('button', { name: 'Notifications' })).toHaveFocus(),
    );
  });

  it('writes its name out, without a tooltip, inside the menu drawer (plan "Layout")', async () => {
    const { user } = renderWithProviders(<NotificationBell withLabel />);

    const bell = screen.getByRole('button', { name: 'Notifications' });
    expect(bell).toHaveTextContent('Notifications');
    await user.hover(bell);

    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument();
  });
});

describe('NotificationBell with notifications (feature 008, US6)', () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it('shows the unread count on the bell, "9+" above 9, and in its name (FR-033)', async () => {
    const twelve = Array.from({ length: 12 }, (_, index) =>
      notificationOf({ id: `NT-${1001 + index}` }),
    );
    mockApi({ 'GET /api/notifications': { body: pageOf(twelve) } });
    renderWithProviders(<NotificationBell />, { user: testUser() });

    expect(
      await screen.findByRole('button', { name: 'Notifications, 12 unread' }),
    ).toBeInTheDocument();
    expect(screen.getByText('9+')).toBeInTheDocument();
  });

  it('lists the notifications, closes after a choice and counts one less (FR-033, FR-034)', async () => {
    let read = false;
    mockApi({
      'GET /api/notifications': () => ({ body: pageOf([notificationOf({ read })]) }),
      'POST /api/notifications/NT-1003/read': () => {
        read = true;
        return { body: notificationOf({ read: true }) };
      },
    });
    const { user, router } = renderWithProviders(<NotificationBell />, { user: testUser() });

    await user.click(await screen.findByRole('button', { name: 'Notifications, 1 unread' }));
    const popover = screen.getByRole('dialog', { name: 'Notifications' });
    await user.click(within(popover).getByRole('button', { name: /^Approval needed/ }));

    await waitFor(() =>
      expect(screen.queryByRole('dialog', { name: 'Notifications' })).not.toBeInTheDocument(),
    );
    await waitFor(() => expect(router.state.location.pathname).toBe('/activities/A-1042'));
    expect(await screen.findByRole('button', { name: 'Notifications' })).toBeInTheDocument();
  });

  it('asks again every 30 seconds, so a new one shows within a minute (FR-036)', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    let calls = 0;
    mockApi({
      'GET /api/notifications': () => {
        calls += 1;
        return { body: pageOf(calls === 1 ? [] : [notificationOf()]) };
      },
    });
    renderWithProviders(<NotificationBell />, { user: testUser() });
    await waitFor(() => expect(calls).toBe(1));
    expect(screen.getByRole('button', { name: 'Notifications' })).toBeInTheDocument();

    await act(async () => {
      await vi.advanceTimersByTimeAsync(30_000);
    });

    expect(
      await screen.findByRole('button', { name: 'Notifications, 1 unread' }),
    ).toBeInTheDocument();
    expect(calls).toBe(2);
  });
});
