import { screen, within } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/test/render';
import { TopBar } from './TopBar';
import type { CurrentUser } from './useSession';
import { testUser } from '@/test/users';

const PRIYA: CurrentUser = testUser();

afterEach(() => {
  localStorage.clear();
  delete document.documentElement.dataset.theme;
});

describe('TopBar', () => {
  it('is the page banner with the logo, which leads Home (FR-010)', () => {
    renderWithProviders(<TopBar />, { user: PRIYA });

    const bar = screen.getByRole('banner');
    expect(within(bar).getByRole('link', { name: 'NorthStar home' })).toHaveAttribute('href', '/');
  });

  it('has the theme switch, the bell and the account button on the right (FR-010)', () => {
    renderWithProviders(<TopBar />, { user: PRIYA });

    const bar = screen.getByRole('banner');
    expect(within(bar).getByRole('button', { name: 'Switch to dark mode' })).toBeInTheDocument();
    expect(within(bar).getByRole('button', { name: 'Notifications' })).toBeInTheDocument();
    expect(within(bar).getByRole('button', { name: 'Priya — account menu' })).toHaveTextContent(
      'Priya',
    );
  });

  it('has the menu button that opens the sections on narrow screens (FR-011)', () => {
    renderWithProviders(<TopBar />, { user: PRIYA });

    expect(
      within(screen.getByRole('banner')).getByRole('button', { name: 'Open menu' }),
    ).toBeInTheDocument();
  });
});
