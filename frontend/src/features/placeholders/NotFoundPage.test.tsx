import { screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import type { CurrentUser } from '@/session/useSession';
import { mockApi } from '@/testing/mock-api';
import { renderAppRoutes } from '@/testing/app';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { NotFoundPage } from './NotFoundPage';

const PRIYA: CurrentUser = testUser();

const BANNER = 'Demo environment — sign-in and data are for demonstration only';

afterEach(() => {
  localStorage.clear();
  delete document.documentElement.dataset.theme;
});

describe('NotFoundPage', () => {
  it('says "Page not found" and offers a way back to Home (FR-015, US2 scenario 6)', async () => {
    const { user, router } = renderWithProviders(<NotFoundPage />, { route: '/nowhere' });

    expect(screen.getByRole('heading', { level: 1, name: 'Page not found' })).toBeInTheDocument();
    expect(
      screen.getByText("There's no page at this address. Check the address, or go back to Home."),
    ).toBeInTheDocument();
    expect(document.title).toBe('Page not found · MAPP');

    await user.click(screen.getByRole('link', { name: 'Go to Home' }));
    expect(router.state.location.pathname).toBe('/');
  });

  it('is shown inside the app layout to a signed-in person (US2 scenario 6)', async () => {
    mockApi({ 'GET /api/v1/me': { body: PRIYA } });
    const { router } = renderAppRoutes({ route: '/nowhere' });

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Page not found' }),
    ).toBeInTheDocument();
    expect(screen.getByRole('navigation', { name: 'Main' })).toBeInTheDocument();
    expect(screen.getByText(BANNER)).toBeInTheDocument();
    expect(router.state.location.pathname).toBe('/nowhere');
  });

  it('is shown in the public layout to a visitor who is not signed in, without a redirect (US2 scenario 6)', async () => {
    mockApi({
      'GET /api/v1/me': {
        status: 401,
        body: {
          error: { code: 'sign_in_required', message: 'Please sign in to continue.', fields: [] },
        },
      },
    });
    const { router } = renderAppRoutes({ route: '/nowhere' });

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Page not found' }),
    ).toBeInTheDocument();
    expect(screen.queryByRole('navigation', { name: 'Main' })).not.toBeInTheDocument();
    expect(screen.getByText(BANNER)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Switch to dark mode' })).toBeInTheDocument();
    expect(router.state.location.pathname).toBe('/nowhere');
  });

  it('shows the full-page loading placeholder while the session is checked (plan "Session")', async () => {
    mockApi({ 'GET /api/v1/me': () => new Promise<never>(() => {}) });
    renderAppRoutes({ route: '/nowhere' });

    expect(await screen.findByRole('status')).toHaveTextContent('Loading…');
    expect(document.title).toBe('Page not found · MAPP');
  });
});
