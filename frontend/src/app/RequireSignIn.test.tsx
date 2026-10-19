import { screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { mockApi } from '@/test/mockApi';
import { renderRoutes } from '@/test/render';
import { routes } from './router';
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

const DEMO_PEOPLE = { password: 'northstar', people: [] };

const SIGN_IN_REQUIRED = {
  code: 'sign_in_required',
  message: 'Please sign in to continue.',
  fields: [],
};

const BANNER = 'Demo environment — sign-in and data are for demonstration only';

afterEach(() => {
  localStorage.clear();
  delete document.documentElement.dataset.theme;
});

describe('RequireSignIn', () => {
  it('shows a loading placeholder, with the banner and no sidebar, while the session is checked (plan "Session")', async () => {
    // The check never answers in this test.
    mockApi({ 'GET /api/me': () => new Promise<never>(() => {}) });
    renderRoutes(routes, { route: '/plans' });

    expect(await screen.findByRole('status')).toHaveTextContent('Loading…');
    expect(screen.getByText(BANNER)).toBeInTheDocument();
    expect(screen.queryByRole('navigation', { name: 'Main' })).not.toBeInTheDocument();
    expect(document.title).toBe('Plans · NorthStar');
  });

  it('shows a full-page error when the API cannot be reached, without leaving the page (edge case, FR-020)', async () => {
    mockApi({ 'GET /api/me': () => Promise.reject(new TypeError('Failed to fetch')) });
    const { router } = renderRoutes(routes, { route: '/approvals' });

    expect(await screen.findByRole('alert')).toHaveTextContent(
      "NorthStar can't reach its server right now. Check that it is running and try again.",
    );
    expect(screen.getByRole('button', { name: 'Try again' })).toBeInTheDocument();
    expect(screen.getByText(BANNER)).toBeInTheDocument();
    expect(document.title).toBe('Approvals · NorthStar');
    expect(router.state.location.pathname).toBe('/approvals');
  });

  it('checks the session again on Try again, and shows the page once the API answers (edge case)', async () => {
    let checks = 0;
    mockApi({
      'GET /api/me': () =>
        ++checks === 1 ? Promise.reject(new TypeError('Failed to fetch')) : { body: PRIYA },
    });
    const { user } = renderRoutes(routes, { route: '/approvals' });
    await screen.findByRole('alert');

    await user.click(screen.getByRole('button', { name: 'Try again' }));

    expect(await screen.findByRole('heading', { level: 1, name: 'Approvals' })).toBeInTheDocument();
  });

  it('sends a visitor who is not signed in to the sign-in page, with the address to return to (FR-005, US1 scenario 1)', async () => {
    mockApi({
      'GET /api/me': { status: 401, body: SIGN_IN_REQUIRED },
      'GET /api/auth/demo-people': { body: DEMO_PEOPLE },
    });
    const { router } = renderRoutes(routes, { route: '/plans?status=draft&year=2026' });

    await waitFor(() => expect(router.state.location.pathname).toBe('/sign-in'));
    expect(router.state.location.search).toBe('?next=%2Fplans%3Fstatus%3Ddraft%26year%3D2026');
    // The protected address is replaced, so Back does not return to it.
    expect(router.state.historyAction).toBe('REPLACE');
    expect(
      await screen.findByRole('heading', { level: 1, name: 'Welcome back' }),
    ).toBeInTheDocument();
  });

  it('shows the page inside the app layout to a signed-in person (FR-010)', async () => {
    mockApi({ 'GET /api/me': { body: PRIYA } });
    renderRoutes(routes, { route: '/plans' });

    expect(await screen.findByRole('heading', { level: 1, name: 'Plans' })).toBeInTheDocument();
    expect(screen.getByRole('navigation', { name: 'Main' })).toBeInTheDocument();
    expect(screen.getByRole('main')).toHaveAttribute('id', 'main');
  });

  it('checks the session again on every navigation (research R15)', async () => {
    const api = mockApi({ 'GET /api/me': { body: PRIYA } });
    const { user } = renderRoutes(routes, { route: '/' });
    await screen.findByRole('heading', { level: 1, name: 'Welcome back, Priya.' });
    const checksBefore = api.requests.length;

    await user.click(screen.getByRole('link', { name: 'Plans' }));

    await waitFor(() => expect(api.requests).toHaveLength(checksBefore + 1));
    expect(api.requests.at(-1)).toMatchObject({ method: 'GET', path: '/api/me' });
  });
});
