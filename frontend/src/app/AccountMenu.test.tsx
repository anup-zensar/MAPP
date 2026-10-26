import { screen, waitFor, within } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { mockApi } from '@/testing/mockApi';
import { findMessages, renderWithProviders } from '@/testing/render';
import { AccountMenu } from './AccountMenu';
import type { CurrentUser } from '@/session/useSession';
import { testUser } from '@/testing/fixtures/users';

const PRIYA: CurrentUser = testUser();

/** A person whose first name and job title are 100 characters long (edge case). */
const LONG_FIRST_NAME = 'Maximiliana'.repeat(9) + 'A';
const LONG_JOB_TITLE = 'Regional marketing and communications coordinator for '.padEnd(100, 'x');
const LONG: CurrentUser = {
  ...PRIYA,
  name: `${LONG_FIRST_NAME} Raman`,
  firstName: LONG_FIRST_NAME,
  jobTitle: LONG_JOB_TITLE,
};

function accountButton(firstName = 'Priya') {
  return screen.getByRole('button', { name: `${firstName} — account menu` });
}

afterEach(() => {
  localStorage.clear();
  delete document.documentElement.dataset.theme;
});

describe('AccountMenu', () => {
  it('is the avatar and first name, named "<first name> — account menu" (FR-012)', () => {
    renderWithProviders(<AccountMenu />, { user: PRIYA });

    expect(accountButton()).toHaveTextContent('PRPriya');
  });

  it('opens a menu with the person’s name, e-mail and job title, My profile and Sign out (FR-012)', async () => {
    const { user } = renderWithProviders(<AccountMenu />, { user: PRIYA });

    await user.click(accountButton());

    const menu = screen.getByRole('menu');
    expect(menu).toHaveTextContent('Priya Raman');
    expect(menu).toHaveTextContent('priya.raman@shell.com');
    expect(menu).toHaveTextContent('Marketer');
    expect(
      within(menu)
        .getAllByRole('menuitem')
        .map((item) => item.textContent),
    ).toEqual(['My profile', 'Sign out']);
  });

  it('shows a long name and job title in full in a tooltip, on focus (edge case)', async () => {
    const { user } = renderWithProviders(<AccountMenu />, { user: LONG });

    await user.tab();

    expect(accountButton(LONG_FIRST_NAME)).toHaveFocus();
    expect(await screen.findByRole('tooltip')).toHaveTextContent(
      `${LONG_FIRST_NAME} Raman — ${LONG_JOB_TITLE}`,
    );
  });

  it('writes the full name and job title out in the menu, never cut off (edge case)', async () => {
    const { user } = renderWithProviders(<AccountMenu />, { user: LONG });

    await user.click(accountButton(LONG_FIRST_NAME));

    const menu = screen.getByRole('menu');
    expect(menu).toHaveTextContent(`${LONG_FIRST_NAME} Raman`);
    expect(menu).toHaveTextContent(LONG_JOB_TITLE);
  });

  it('opens My profile (FR-012)', async () => {
    const { user, router } = renderWithProviders(<AccountMenu />, { user: PRIYA });

    await user.click(accountButton());
    await user.click(screen.getByRole('menuitem', { name: 'My profile' }));

    expect(router.state.location.pathname).toBe('/profile');
  });

  it('signs out and opens the sign-in page with the notice (US4 scenario 1)', async () => {
    mockApi({ 'POST /api/v1/auth/sign-out': { status: 204 } });
    const { user, router } = renderWithProviders(<AccountMenu />, {
      user: PRIYA,
      route: '/plans',
    });

    await user.click(accountButton());
    await user.click(screen.getByRole('menuitem', { name: 'Sign out' }));

    expect(await screen.findByText('Sign-in page')).toBeInTheDocument();
    expect(router.state.location.search).toBe('');
    expect(router.state.location.state).toEqual({ notice: 'signedOut' });
  });

  it('keeps the person signed in and says why when signing out fails (FR-020)', async () => {
    const api = mockApi({
      'POST /api/v1/auth/sign-out': () => Promise.reject(new TypeError('Failed to fetch')),
    });
    const { user, router } = renderWithProviders(<AccountMenu />, {
      user: PRIYA,
      route: '/plans',
    });

    await user.click(accountButton());
    await user.click(screen.getByRole('menuitem', { name: 'Sign out' }));

    const messages = await findMessages();
    expect(messages).toHaveTextContent(
      "MAPP can't reach its server right now. Check that it is running and try again.",
    );
    expect(router.state.location.pathname).toBe('/plans');

    await user.click(within(messages).getByRole('button', { name: 'Try again' }));
    await waitFor(() => expect(api.requests).toHaveLength(2));
  });
});
