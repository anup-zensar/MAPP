import { act, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { HOME_ANSWERS } from '@/testing/fixtures/home';
import {
  DANIEL,
  gate,
  NO_APPROVALS,
  NO_PLANS,
  PRIYA,
  SESSION_EXPIRED,
  SIGN_IN_REQUIRED,
} from '@/testing/fixtures/session';
import { DEMO_PEOPLE } from '@/testing/fixtures/sign-in';
import { mockApi } from '@/testing/mock-api';
import { renderRoutes } from '@/testing/render';
import { reply } from '@/testing/replies';
import { routes } from './router';

afterEach(() => {
  localStorage.clear();
  delete document.documentElement.dataset.theme;
});

describe('SessionProvider: a lost session (US4)', () => {
  it('says "Please sign in again" after a reload once the session has expired (FR-006)', async () => {
    mockApi([
      reply('GET /api/v1/me', { status: 401, body: SESSION_EXPIRED }),
      reply('GET /api/v1/auth/demo-people', { body: DEMO_PEOPLE }),
    ]);
    const { router } = renderRoutes(routes, { route: '/plans' });

    expect(await screen.findByText('Please sign in again')).toBeInTheDocument();
    expect(router.state.location.pathname).toBe('/sign-in');
    expect(router.state.location.search).toBe('?next=%2Fplans');
  });

  it('keeps the sign-in page’s own way back when the expired session is found there (FR-005, FR-006)', async () => {
    mockApi([
      reply('GET /api/v1/me', { status: 401, body: SESSION_EXPIRED }),
      reply('GET /api/v1/auth/demo-people', { body: DEMO_PEOPLE }),
    ]);
    const { router } = renderRoutes(routes, { route: '/sign-in?next=%2Fplans' });

    expect(await screen.findByText('Please sign in again')).toBeInTheDocument();
    expect(router.state.location.pathname).toBe('/sign-in');
    expect(router.state.location.search).toBe('?next=%2Fplans');
  });

  it('brings the person back to where they were after signing in again (US4 scenario 2)', async () => {
    let checks = 0;
    const api = mockApi([
      reply('GET /api/v1/me', () =>
        ++checks === 1 ? { status: 401, body: SESSION_EXPIRED } : { body: PRIYA },
      ),
      reply('GET /api/v1/auth/demo-people', { body: DEMO_PEOPLE }),
      reply('POST /api/v1/auth/sign-in', { body: PRIYA }),
      ...NO_PLANS,
    ]);
    const { user, router } = renderRoutes(routes, { route: '/plans?year=2026' });
    await screen.findByText('Please sign in again');

    await user.click(screen.getByRole('button', { name: 'Sign in as Priya Raman, Marketer' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/plans'));
    expect(router.state.location.search).toBe('?year=2026');
    expect(await screen.findByRole('heading', { level: 1, name: 'Plans' })).toBeInTheDocument();
    expect(api.bodiesOf('POST /api/v1/auth/sign-in')).toEqual([
      { email: 'priya.raman@shell.com', password: 'mapp' },
    ]);
  });

  it('ends a lost session once, however many answers say so at the same time (US4 scenario 2)', async () => {
    const answers = gate();
    const api = mockApi([
      reply('GET /api/v1/me', { body: PRIYA }),
      reply('PATCH /api/v1/me/preferences', async () => {
        await answers.opened;
        return { status: 401, body: SESSION_EXPIRED };
      }),
      reply('GET /api/v1/auth/demo-people', { body: DEMO_PEOPLE }),
      ...NO_PLANS,
    ]);
    const { user, router } = renderRoutes(routes, { route: '/plans?year=2026' });
    await screen.findByRole('heading', { level: 1, name: 'Plans' });
    const signInPages = new Set<string>();
    router.subscribe((state) => {
      if (state.location.pathname === '/sign-in') {
        signInPages.add(state.location.key);
      }
    });

    // Two choices whose saves are both on their way when the API says the session is gone.
    await user.click(screen.getByRole('button', { name: 'Collapse' }));
    await user.click(await screen.findByRole('button', { name: 'Expand sidebar' }));
    await waitFor(() => expect(api.bodiesOf('PATCH /api/v1/me/preferences')).toHaveLength(2));
    act(() => answers.open());

    await waitFor(() => expect(router.state.location.pathname).toBe('/sign-in'));
    expect(await screen.findByText('Please sign in again')).toBeInTheDocument();
    expect(signInPages.size).toBe(1);
    expect(router.state.location.search).toBe('?next=%2Fplans%3Fyear%3D2026');
    // A lost session is not a failed save: no "couldn't be saved" message.
    expect(screen.queryByText("Your choice couldn't be saved.")).not.toBeInTheDocument();
  });

  it('stays signed out when a session check that was on its way answers after signing out (US4 scenario 1)', async () => {
    const lateAnswer = gate();
    let checks = 0;
    mockApi([
      reply('GET /api/v1/me', async () => {
        checks += 1;
        if (checks === 2) {
          // The check of the next navigation: the API answers it only after the sign-out.
          await lateAnswer.opened;
        }
        return checks <= 2 ? { body: PRIYA } : { status: 401, body: SIGN_IN_REQUIRED };
      }),
      reply('POST /api/v1/auth/sign-out', { status: 204 }),
      reply('GET /api/v1/auth/demo-people', { body: DEMO_PEOPLE }),
      ...NO_PLANS,
      ...NO_APPROVALS,
      ...HOME_ANSWERS,
    ]);
    const { user, router, queryClient } = renderRoutes(routes, { route: '/plans' });
    await screen.findByRole('heading', { level: 1, name: 'Plans' });
    await user.click(screen.getByRole('link', { name: 'Approvals' }));
    await waitFor(() => expect(checks).toBe(2));

    await user.click(await screen.findByRole('button', { name: 'Priya — account menu' }));
    await user.click(screen.getByRole('menuitem', { name: 'Sign out' }));
    await waitFor(() => expect(router.state.location.pathname).toBe('/sign-in'));
    act(() => lateAnswer.open());
    await waitFor(() => expect(queryClient.getQueryState(['session'])?.fetchStatus).toBe('idle'));

    // The late answer is about the session that has just ended: it does not bring Priya back.
    expect(queryClient.getQueryData(['session'])).toBeNull();
    expect(router.state.location.pathname).toBe('/sign-in');
    expect(await screen.findByText('You have been signed out.')).toBeInTheDocument();
    expect(screen.queryByText('Please sign in again')).not.toBeInTheDocument();
  });

  it('notices another person signing in in another tab, and starts again on Home (edge case)', async () => {
    let checks = 0;
    mockApi([
      reply('GET /api/v1/me', () => ({ body: ++checks === 1 ? PRIYA : DANIEL })),
      ...NO_PLANS,
      ...NO_APPROVALS,
      ...HOME_ANSWERS,
    ]);
    const { user, router, queryClient } = renderRoutes(routes, { route: '/plans' });
    await screen.findByRole('heading', { level: 1, name: 'Plans' });
    queryClient.setQueryData(['sample-folders'], { items: [], total: 0 });

    // Daniel signed in in another tab; this tab notices at its next check (a navigation).
    await user.click(screen.getByRole('link', { name: 'Approvals' }));

    // Home renders all its cards here; under the full suite's load that can take more than the
    // usual three seconds, so these two waits get ten.
    await waitFor(() => expect(router.state.location.pathname).toBe('/'), { timeout: 10_000 });
    expect(
      await screen.findByRole(
        'heading',
        { level: 1, name: 'Welcome back, Daniel.' },
        { timeout: 10_000 },
      ),
    ).toBeInTheDocument();
    expect(screen.getByRole('region', { name: 'Messages (F8)' })).toHaveTextContent(
      "You're now signed in as Daniel Okoro — this was changed in another tab.",
    );
    expect(queryClient.getQueryData(['sample-folders'])).toBeUndefined();
  });
});
