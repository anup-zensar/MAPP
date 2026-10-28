import { screen, within } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/testing/render';
import type { CurrentUser } from '@/session/useSession';
import { testUser } from '@/testing/fixtures/users';
import { TopBar } from './TopBar';

const PRIYA: CurrentUser = testUser();

afterEach(() => {
  localStorage.clear();
  delete document.documentElement.dataset.theme;
});

describe('TopBar', () => {
  it('is the page banner with the logo, which leads Home (FR-010)', () => {
    renderWithProviders(<TopBar />, { user: PRIYA });

    const bar = screen.getByRole('banner');
    expect(within(bar).getByRole('link', { name: 'SP&A One home' })).toHaveAttribute('href', '/');
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
});
