import { screen, waitFor, within } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { mockApi } from '@/test/mockApi';
import { findMessages, renderWithProviders } from '@/test/render';
import { ThemeToggle } from './ThemeToggle';
import type { CurrentUser } from './useSession';

const PRIYA: CurrentUser = {
  id: 'U-1002',
  name: 'Priya Raman',
  firstName: 'Priya',
  email: 'priya.raman@northstar.demo',
  jobTitle: 'Marketer',
  initials: 'PR',
  preferences: { theme: 'system', sidebarCollapsed: false },
};

const PRIYA_IN_DARK: CurrentUser = {
  ...PRIYA,
  preferences: { theme: 'dark', sidebarCollapsed: false },
};

function shownTheme() {
  return document.documentElement.dataset.theme;
}

afterEach(() => {
  localStorage.clear();
  delete document.documentElement.dataset.theme;
});

describe('ThemeToggle', () => {
  it('offers dark mode while the app is light, explained by a tooltip (FR-014)', async () => {
    const { user } = renderWithProviders(<ThemeToggle />, { user: PRIYA });

    await user.hover(screen.getByRole('button', { name: 'Switch to dark mode' }));

    expect(await screen.findByRole('tooltip')).toHaveTextContent('Switch to dark mode');
  });

  it('turns the whole app dark at once and saves the choice for the person (FR-014, US2 scenario 5)', async () => {
    const api = mockApi({
      'PATCH /api/me/preferences': { body: { theme: 'dark', sidebarCollapsed: false } },
    });
    const { user } = renderWithProviders(<ThemeToggle />, { user: PRIYA });

    await user.click(screen.getByRole('button', { name: 'Switch to dark mode' }));

    await waitFor(() => expect(shownTheme()).toBe('dark'));
    expect(screen.getByRole('button', { name: 'Switch to light mode' })).toBeInTheDocument();
    // Only the changed preference is sent.
    expect(api.bodiesOf('PATCH /api/me/preferences')).toEqual([{ theme: 'dark' }]);
  });

  it('switches a person in dark mode back to light (FR-014)', async () => {
    const api = mockApi({
      'PATCH /api/me/preferences': { body: { theme: 'light', sidebarCollapsed: false } },
    });
    const { user } = renderWithProviders(<ThemeToggle />, { user: PRIYA_IN_DARK });

    await user.click(screen.getByRole('button', { name: 'Switch to light mode' }));

    await waitFor(() => expect(shownTheme()).toBe('light'));
    expect(api.bodiesOf('PATCH /api/me/preferences')).toEqual([{ theme: 'light' }]);
  });

  it('changes only this browser’s choice when nobody is signed in (research R16)', async () => {
    const api = mockApi({});
    const { user } = renderWithProviders(<ThemeToggle />, { user: null });

    await user.click(screen.getByRole('button', { name: 'Switch to dark mode' }));

    await waitFor(() => expect(shownTheme()).toBe('dark'));
    expect(localStorage.getItem('northstar.theme')).toBe('dark');
    expect(api.requests).toEqual([]);
  });

  it('keeps the new choice and offers Try again when it cannot be saved (plan "Preferences and theme")', async () => {
    let saves = 0;
    const api = mockApi({
      'PATCH /api/me/preferences': () =>
        ++saves === 1
          ? {
              status: 500,
              body: {
                code: 'internal_error',
                message: 'Something went wrong on our side. Please try again.',
                fields: [],
              },
            }
          : { body: { theme: 'dark', sidebarCollapsed: false } },
    });
    const { user } = renderWithProviders(<ThemeToggle />, { user: PRIYA });

    await user.click(screen.getByRole('button', { name: 'Switch to dark mode' }));

    const messages = await findMessages();
    expect(messages).toHaveTextContent("Your choice couldn't be saved.");
    expect(shownTheme()).toBe('dark');

    await user.click(within(messages).getByRole('button', { name: 'Try again' }));

    await waitFor(() =>
      expect(api.bodiesOf('PATCH /api/me/preferences')).toEqual([
        { theme: 'dark' },
        { theme: 'dark' },
      ]),
    );
  });

  it('leaves a lost session to the sign-in page, without a message (US4 scenario 2)', async () => {
    mockApi({
      'PATCH /api/me/preferences': {
        status: 401,
        body: { code: 'session_expired', message: 'Please sign in again', fields: [] },
      },
    });
    const { user, router } = renderWithProviders(<ThemeToggle />, {
      user: PRIYA,
      route: '/plans',
    });

    await user.click(screen.getByRole('button', { name: 'Switch to dark mode' }));

    expect(await screen.findByText('Sign-in page')).toBeInTheDocument();
    expect(router.state.location.search).toBe('?next=%2Fplans');
    expect(router.state.location.state).toEqual({ notice: 'sessionLost' });
    expect(screen.getByRole('region', { name: 'Messages (F8)' })).not.toHaveTextContent(
      "Your choice couldn't be saved.",
    );
  });

  it('writes its name out, without a tooltip, inside the menu drawer (plan "Layout")', async () => {
    const { user } = renderWithProviders(<ThemeToggle withLabel />, { user: PRIYA });

    const toggle = screen.getByRole('button', { name: 'Switch to dark mode' });
    expect(toggle).toHaveTextContent('Dark mode');
    await user.hover(toggle);

    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument();
  });
});
