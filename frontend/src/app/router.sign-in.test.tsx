import { screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { HOME_ANSWERS } from '@/testing/fixtures/home';
import { mockApi } from '@/testing/mock-api';
import { renderRoutes } from '@/testing/render';
import type { CurrentUser } from '@/session/useSession';
import { NO_DELEGATIONS } from '@/testing/fixtures/approvals';
import { testUser } from '@/testing/fixtures/users';
import { apiError, reply } from '@/testing/replies';
import '@/testing/pages';
import { routes } from './routes';

const PRIYA: CurrentUser = testUser();

const DEMO_PEOPLE = { password: 'mapp', people: [] };

const SIGN_IN_REQUIRED = apiError('sign_in_required', 'Please sign in to continue.');

const BANNER = 'Demo environment — sign-in and data are for demonstration only';

/** The Plans page's request, answered with no plans: these tests use /plans as a signed-in page. */
const NO_PLANS = [
  reply('GET /api/v1/plans', {
    body: {
      items: [],
      total: 0,
      summary: {
        plans: 0,
        draft: 0,
        published: 0,
        plannedCost: { amount: '0.00', currency: 'USD' },
        budget: { amount: '0.00', currency: 'USD' },
      },
      years: [],
    },
  }),
];

/** The Approvals page's request, answered with no rows. */
const NO_APPROVALS = [
  reply('GET /api/v1/approvals', { body: { items: [], total: 0 } }),
  reply('GET /api/v1/approval-delegations', NO_DELEGATIONS),
];

afterEach(() => {
  localStorage.clear();
  delete document.documentElement.dataset.theme;
});

describe('RequireSignIn', () => {
  it('shows a loading placeholder, with the banner and no sidebar, while the session is checked (plan "Session")', async () => {
    // The check never answers in this test.
    mockApi([reply('GET /api/v1/me', () => new Promise<never>(() => {}))]);
    renderRoutes(routes, { route: '/plans' });

    expect(await screen.findByRole('status')).toHaveTextContent('Loading…');
    expect(screen.getByText(BANNER)).toBeInTheDocument();
    expect(screen.queryByRole('navigation', { name: 'Main' })).not.toBeInTheDocument();
    expect(document.title).toBe('Plans · MAPP');
  });

  it('shows a full-page error when the API cannot be reached, without leaving the page (edge case, FR-020)', async () => {
    mockApi([reply('GET /api/v1/me', () => Promise.reject(new TypeError('Failed to fetch')))]);
    const { router } = renderRoutes(routes, { route: '/approvals' });

    expect(await screen.findByRole('alert')).toHaveTextContent(
      "MAPP can't reach its server right now. Check that it is running and try again.",
    );
    expect(screen.getByRole('button', { name: 'Try again' })).toBeInTheDocument();
    expect(screen.getByText(BANNER)).toBeInTheDocument();
    expect(document.title).toBe('Approvals · MAPP');
    expect(router.state.location.pathname).toBe('/approvals');
  });

  it('checks the session again on Try again, and shows the page once the API answers (edge case)', async () => {
    let checks = 0;
    mockApi([
      reply('GET /api/v1/me', () =>
        ++checks === 1 ? Promise.reject(new TypeError('Failed to fetch')) : { body: PRIYA },
      ),
      ...NO_APPROVALS,
    ]);
    const { user } = renderRoutes(routes, { route: '/approvals' });
    await screen.findByRole('alert');

    await user.click(screen.getByRole('button', { name: 'Try again' }));

    expect(await screen.findByRole('heading', { level: 1, name: 'Approvals' })).toBeInTheDocument();
  });

  it('sends a visitor who is not signed in to the sign-in page, with the address to return to (FR-005, US1 scenario 1)', async () => {
    mockApi([
      reply('GET /api/v1/me', { status: 401, body: SIGN_IN_REQUIRED }),
      reply('GET /api/v1/auth/demo-people', { body: DEMO_PEOPLE }),
    ]);
    const { router } = renderRoutes(routes, { route: '/plans?status=draft&year=2026' });

    await waitFor(() => expect(router.state.location.pathname).toBe('/sign-in'));
    expect(router.state.location.search).toBe('?next=%2Fplans%3Fstatus%3Ddraft%26year%3D2026');
    // The protected address is replaced, so Back does not return to it.
    expect(router.state.historyAction).toBe('REPLACE');
    expect(
      await screen.findByRole('heading', { level: 1, name: 'Sign in to MAPP' }),
    ).toBeInTheDocument();
  });

  it('checks the session again on every navigation (research R15)', async () => {
    const api = mockApi([reply('GET /api/v1/me', { body: PRIYA }), ...NO_PLANS, ...HOME_ANSWERS]);
    const sessionChecks = () => api.requests.filter((r) => r.path === '/api/v1/me').length;
    const { user } = renderRoutes(routes, { route: '/' });
    await screen.findByRole('heading', { level: 1, name: 'Welcome back, Priya.' });
    const checksBefore = sessionChecks();

    await user.click(screen.getByRole('link', { name: 'Plans' }));

    await waitFor(() => expect(sessionChecks()).toBe(checksBefore + 1));
    await screen.findByRole('heading', { level: 1, name: 'Plans' });
  });
});
