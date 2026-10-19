import { screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/test/render';
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
    expect(popover).toHaveTextContent("You're all caught up.");
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
