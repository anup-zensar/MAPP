import { screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '@/test/render';
import { NotificationBell } from './NotificationBell';

// Feature 008 counts the unread notifications from its own folder; here there are three.
vi.mock('@/features/notifications/useUnreadNotificationCount', () => ({
  useUnreadNotificationCount: () => 3,
}));

describe('NotificationBell with unread notifications (feature 008 extension point)', () => {
  it('names the count and shows it on the bell', () => {
    renderWithProviders(<NotificationBell />);

    expect(screen.getByRole('button', { name: 'Notifications, 3 unread' })).toBeInTheDocument();
    expect(screen.getByText('3')).toBeInTheDocument();
  });
});
