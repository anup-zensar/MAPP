import { screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { HOME_ANSWERS } from '@/testing/fixtures/home';
import { PRIYA, renderProbe, SIGN_IN_REQUIRED } from '@/testing/fixtures/session';
import { DEMO_PEOPLE } from '@/testing/fixtures/sign-in';
import type { CurrentUser } from '@/session/useSession';
import { mockApi } from '@/testing/mock-api';
import { renderRoutes } from '@/testing/render';
import { reply } from '@/testing/replies';
import '@/testing/pages';
import { routes } from './routes';

afterEach(() => {
  localStorage.clear();
  delete document.documentElement.dataset.theme;
});

describe('SessionProvider: who is signed in (research R15)', () => {
  it('is loading until GET /api/v1/me answers, then knows the signed-in person (FR-006)', async () => {
    mockApi([reply('GET /api/v1/me', { body: PRIYA })]);
    renderProbe();

    expect(screen.getByText('Status: loading')).toBeInTheDocument();
    expect(await screen.findByText('Status: signedIn')).toBeInTheDocument();
    expect(screen.getByText('Person: Priya Raman')).toBeInTheDocument();
  });

  it('treats a 401 answer as signed out (FR-005)', async () => {
    mockApi([reply('GET /api/v1/me', { status: 401, body: SIGN_IN_REQUIRED })]);
    renderProbe();

    expect(await screen.findByText('Status: signedOut')).toBeInTheDocument();
    expect(screen.getByText('Person: nobody')).toBeInTheDocument();
  });

  it.each([
    ['no answer at all', () => Promise.reject(new TypeError('Failed to fetch'))],
    ['an empty text/plain 502', () => new Response('', { status: 502 })],
  ])(
    'is unreachable when nobody is known and the API cannot be asked: %s (edge case)',
    async (_case, answer) => {
      mockApi([reply('GET /api/v1/me', answer)]);
      renderProbe();

      expect(await screen.findByText('Status: unreachable')).toBeInTheDocument();
    },
  );

  it('keeps the person signed in when a later check cannot reach the API (FR-006)', async () => {
    let calls = 0;
    mockApi([
      reply('GET /api/v1/me', () =>
        ++calls === 1 ? { body: PRIYA } : Promise.reject(new TypeError('Failed to fetch')),
      ),
    ]);
    const { user, queryClient } = renderProbe();
    await screen.findByText('Status: signedIn');

    await user.click(screen.getByRole('button', { name: 'Check again' }));

    await waitFor(() => expect(queryClient.getQueryState(['session'])?.status).toBe('error'));
    expect(screen.getByText('Status: signedIn')).toBeInTheDocument();
    expect(screen.getByText('Person: Priya Raman')).toBeInTheDocument();
  });

  it('shows the signed-in person’s light or dark mode, whatever this browser chose (FR-014)', async () => {
    localStorage.setItem('mapp.theme', 'light');
    mockApi([
      reply('GET /api/v1/me', {
        body: { ...PRIYA, preferences: { ...PRIYA.preferences, theme: 'dark' } },
      }),
    ]);
    renderProbe();

    await screen.findByText('Status: signedIn');

    await waitFor(() => expect(document.documentElement.dataset.theme).toBe('dark'));
    expect(localStorage.getItem('mapp.theme')).toBe('dark');
  });
});

describe('SessionProvider: signing out (US4)', () => {
  it('signs out to the sign-in page with "You have been signed out." only, forgetting the person and their data (US4 scenario 1)', async () => {
    mockApi([
      reply('GET /api/v1/me', { body: PRIYA }),
      reply('POST /api/v1/auth/sign-out', { status: 204 }),
      ...HOME_ANSWERS,
      reply('GET /api/v1/auth/demo-people', { body: DEMO_PEOPLE }),
    ]);
    const { user, router, queryClient } = renderRoutes(routes, { route: '/' });

    await user.click(await screen.findByRole('button', { name: 'Priya — account menu' }));
    queryClient.setQueryData(['sample-folders'], { items: [], total: 0 });
    await user.click(screen.getByRole('menuitem', { name: 'Sign out' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/sign-in'));
    expect(router.state.location.search).toBe('');
    expect(await screen.findByText('You have been signed out.')).toBeInTheDocument();
    expect(screen.queryByText('Please sign in again')).not.toBeInTheDocument();
    expect(queryClient.getQueryData(['session'])).toBeNull();
    expect(queryClient.getQueryData(['sample-folders'])).toBeUndefined();
  });

  it('keeps the person’s light or dark mode on the sign-in page after signing out (FR-014, research R16)', async () => {
    // This browser was light when the page opened; the person chose dark.
    localStorage.setItem('mapp.theme', 'light');
    mockApi([
      reply('GET /api/v1/me', {
        body: { ...PRIYA, preferences: { ...PRIYA.preferences, theme: 'dark' } },
      }),
      reply('POST /api/v1/auth/sign-out', { status: 204 }),
      ...HOME_ANSWERS,
      reply('GET /api/v1/auth/demo-people', { body: DEMO_PEOPLE }),
    ]);
    const { user, router } = renderRoutes(routes, { route: '/' });

    await user.click(await screen.findByRole('button', { name: 'Priya — account menu' }));
    await user.click(screen.getByRole('menuitem', { name: 'Sign out' }));

    // The sign-in page shows what a reload would show: this browser's last theme.
    await waitFor(() => expect(router.state.location.pathname).toBe('/sign-in'));
    expect(await screen.findByText('You have been signed out.')).toBeInTheDocument();
    expect(document.documentElement.dataset.theme).toBe('dark');
    expect(localStorage.getItem('mapp.theme')).toBe('dark');
  });

  it('lets the person’s saved theme take over from a visitor’s choice at sign-in (FR-014, research R16)', async () => {
    const priyaInLight: CurrentUser = {
      ...PRIYA,
      preferences: { ...PRIYA.preferences, theme: 'light' },
    };
    let signedIn = false;
    const api = mockApi([
      reply('GET /api/v1/me', () =>
        signedIn ? { body: priyaInLight } : { status: 401, body: SIGN_IN_REQUIRED },
      ),
      reply('GET /api/v1/auth/demo-people', { body: DEMO_PEOPLE }),
      reply('POST /api/v1/auth/sign-in', () => {
        signedIn = true;
        return { body: priyaInLight };
      }),
      reply('POST /api/v1/auth/sign-out', { status: 204 }),
      ...HOME_ANSWERS,
    ]);
    const { user, router } = renderRoutes(routes, { route: '/sign-in' });

    // A visitor switches the sign-in page to dark, then signs in as Priya, who chose light.
    const priya = await screen.findByRole('button', { name: 'Sign in as Priya Raman, Marketer' });
    await user.click(screen.getByRole('button', { name: 'Switch to dark mode' }));
    await waitFor(() => expect(document.documentElement.dataset.theme).toBe('dark'));
    await user.click(priya);
    await waitFor(() => expect(router.state.location.pathname).toBe('/'));
    await waitFor(() => expect(document.documentElement.dataset.theme).toBe('light'));

    // After signing out, the sign-in page keeps Priya's light mode, the last one shown.
    await user.click(await screen.findByRole('button', { name: 'Priya — account menu' }));
    await user.click(screen.getByRole('menuitem', { name: 'Sign out' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/sign-in'));
    expect(await screen.findByText('You have been signed out.')).toBeInTheDocument();
    expect(document.documentElement.dataset.theme).toBe('light');
    expect(api.bodiesOf('PATCH /api/v1/me/preferences')).toEqual([]);
  });
});
