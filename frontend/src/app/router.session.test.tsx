import { act, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { HOME_ANSWERS } from '@/features/home/testData';
import { mockApi } from '@/testing/mockApi';
import { renderRoutes } from '@/testing/render';
import { routes } from './router';
import { SessionProvider } from '@/session/SessionProvider';
import { useSession } from '@/session/useSession';
import type { CurrentUser } from '@/session/useSession';
import { NO_DELEGATIONS } from '@/features/approvals/testData';
import { testUser } from '@/testing/users';

const PRIYA: CurrentUser = testUser();

const DANIEL: CurrentUser = testUser({
  id: 'U-1003',
  name: 'Daniel Okoro',
  firstName: 'Daniel',
  email: 'daniel.okoro@shell.com',
  jobTitle: 'Marketing team leader',
  initials: 'DO',
});

const DEMO_PEOPLE = {
  password: 'mapp',
  people: [
    {
      id: 'U-1002',
      name: 'Priya Raman',
      email: 'priya.raman@shell.com',
      jobTitle: 'Marketer',
      initials: 'PR',
    },
  ],
};

const SIGN_IN_REQUIRED = {
  error: {
    code: 'sign_in_required',
    message: 'Please sign in to continue.',
    fields: [],
  },
};
const SESSION_EXPIRED = {
  error: { code: 'session_expired', message: 'Please sign in again', fields: [] },
};

/** The Plans page's request, answered with no plans: these tests use /plans as a signed-in page. */
const NO_PLANS = {
  'GET /api/v1/plans': {
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
  },
};

/** The Approvals page's request, answered with no rows. */
const NO_APPROVALS = {
  'GET /api/v1/approvals': { body: { items: [], total: 0 } },
  // The Approvals page lists the person's delegations too (feature 017).
  'GET /api/v1/approval-delegations': NO_DELEGATIONS,
};

/** Shows what useSession() says, with a button that checks the session again. */
function SessionProbe() {
  const session = useSession();
  return (
    <>
      <p>Status: {session.status}</p>
      <p>Person: {session.user?.name ?? 'nobody'}</p>
      <button type="button" onClick={session.refetch}>
        Check again
      </button>
    </>
  );
}

/** The probe under the session, on a fresh page load: nothing is known until GET /api/v1/me answers. */
function renderProbe() {
  return renderRoutes([
    {
      path: '*',
      element: (
        <SessionProvider>
          <SessionProbe />
        </SessionProvider>
      ),
    },
  ]);
}

/** A promise the test resolves when it wants, to keep several requests waiting at once. */
function gate() {
  let open = () => {};
  const opened = new Promise<void>((resolve) => {
    open = resolve;
  });
  return { opened, open };
}

afterEach(() => {
  localStorage.clear();
  delete document.documentElement.dataset.theme;
});

describe('SessionProvider: who is signed in (research R15)', () => {
  it('is loading until GET /api/v1/me answers, then knows the signed-in person (FR-006)', async () => {
    mockApi({ 'GET /api/v1/me': { body: PRIYA } });
    renderProbe();

    expect(screen.getByText('Status: loading')).toBeInTheDocument();
    expect(await screen.findByText('Status: signedIn')).toBeInTheDocument();
    expect(screen.getByText('Person: Priya Raman')).toBeInTheDocument();
  });

  it('treats a 401 answer as signed out (FR-005)', async () => {
    mockApi({ 'GET /api/v1/me': { status: 401, body: SIGN_IN_REQUIRED } });
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
      mockApi({ 'GET /api/v1/me': answer });
      renderProbe();

      expect(await screen.findByText('Status: unreachable')).toBeInTheDocument();
    },
  );

  it('keeps the person signed in when a later check cannot reach the API (FR-006)', async () => {
    let calls = 0;
    mockApi({
      'GET /api/v1/me': () =>
        ++calls === 1 ? { body: PRIYA } : Promise.reject(new TypeError('Failed to fetch')),
    });
    const { user, queryClient } = renderProbe();
    await screen.findByText('Status: signedIn');

    await user.click(screen.getByRole('button', { name: 'Check again' }));

    await waitFor(() => expect(queryClient.getQueryState(['session'])?.status).toBe('error'));
    expect(screen.getByText('Status: signedIn')).toBeInTheDocument();
    expect(screen.getByText('Person: Priya Raman')).toBeInTheDocument();
  });

  it('shows the signed-in person’s light or dark mode, whatever this browser chose (FR-014)', async () => {
    localStorage.setItem('mapp.theme', 'light');
    mockApi({
      'GET /api/v1/me': {
        body: { ...PRIYA, preferences: { theme: 'dark', sidebarCollapsed: false } },
      },
    });
    renderProbe();

    await screen.findByText('Status: signedIn');

    await waitFor(() => expect(document.documentElement.dataset.theme).toBe('dark'));
    expect(localStorage.getItem('mapp.theme')).toBe('dark');
  });
});

describe('SessionProvider: ending a session (US4)', () => {
  it('signs out to the sign-in page with "You have been signed out." only (US4 scenario 1)', async () => {
    mockApi({
      'GET /api/v1/me': { body: PRIYA },
      'POST /api/v1/auth/sign-out': { status: 204 },
      ...HOME_ANSWERS,
      'GET /api/v1/auth/demo-people': { body: DEMO_PEOPLE },
    });
    const { user, router } = renderRoutes(routes, { route: '/' });

    await user.click(await screen.findByRole('button', { name: 'Priya — account menu' }));
    await user.click(screen.getByRole('menuitem', { name: 'Sign out' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/sign-in'));
    expect(router.state.location.search).toBe('');
    expect(await screen.findByText('You have been signed out.')).toBeInTheDocument();
    expect(screen.queryByText('Please sign in again')).not.toBeInTheDocument();
  });

  it('forgets the person and their data when the session ends (US4 scenario 1)', async () => {
    mockApi({
      'GET /api/v1/me': { body: PRIYA },
      'POST /api/v1/auth/sign-out': { status: 204 },
      ...HOME_ANSWERS,
      'GET /api/v1/auth/demo-people': { body: DEMO_PEOPLE },
    });
    const { user, router, queryClient } = renderRoutes(routes, { route: '/' });
    await user.click(await screen.findByRole('button', { name: 'Priya — account menu' }));
    queryClient.setQueryData(['sample-folders'], { items: [], total: 0 });

    await user.click(screen.getByRole('menuitem', { name: 'Sign out' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/sign-in'));
    expect(queryClient.getQueryData(['session'])).toBeNull();
    expect(queryClient.getQueryData(['sample-folders'])).toBeUndefined();
  });

  it('says "Please sign in again" after a reload once the session has expired (FR-006)', async () => {
    mockApi({
      'GET /api/v1/me': { status: 401, body: SESSION_EXPIRED },
      'GET /api/v1/auth/demo-people': { body: DEMO_PEOPLE },
    });
    const { router } = renderRoutes(routes, { route: '/plans' });

    expect(await screen.findByText('Please sign in again')).toBeInTheDocument();
    expect(router.state.location.pathname).toBe('/sign-in');
    expect(router.state.location.search).toBe('?next=%2Fplans');
  });

  it('keeps the sign-in page’s own way back when the expired session is found there (FR-005, FR-006)', async () => {
    mockApi({
      'GET /api/v1/me': { status: 401, body: SESSION_EXPIRED },
      'GET /api/v1/auth/demo-people': { body: DEMO_PEOPLE },
    });
    const { router } = renderRoutes(routes, { route: '/sign-in?next=%2Fplans' });

    expect(await screen.findByText('Please sign in again')).toBeInTheDocument();
    expect(router.state.location.pathname).toBe('/sign-in');
    expect(router.state.location.search).toBe('?next=%2Fplans');
  });

  it('keeps the person’s light or dark mode on the sign-in page after signing out (FR-014, research R16)', async () => {
    // This browser was light when the page opened; the person chose dark.
    localStorage.setItem('mapp.theme', 'light');
    mockApi({
      'GET /api/v1/me': {
        body: { ...PRIYA, preferences: { theme: 'dark', sidebarCollapsed: false } },
      },
      'POST /api/v1/auth/sign-out': { status: 204 },
      ...HOME_ANSWERS,
      'GET /api/v1/auth/demo-people': { body: DEMO_PEOPLE },
    });
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
    const priyaInLight = { ...PRIYA, preferences: { theme: 'light', sidebarCollapsed: false } };
    let signedIn = false;
    const api = mockApi({
      'GET /api/v1/me': () =>
        signedIn ? { body: priyaInLight } : { status: 401, body: SIGN_IN_REQUIRED },
      'GET /api/v1/auth/demo-people': { body: DEMO_PEOPLE },
      'POST /api/v1/auth/sign-in': () => {
        signedIn = true;
        return { body: priyaInLight };
      },
      'POST /api/v1/auth/sign-out': { status: 204 },
      ...HOME_ANSWERS,
    });
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

  it('brings the person back to where they were after signing in again (US4 scenario 2)', async () => {
    let checks = 0;
    const api = mockApi({
      'GET /api/v1/me': () =>
        ++checks === 1 ? { status: 401, body: SESSION_EXPIRED } : { body: PRIYA },
      'GET /api/v1/auth/demo-people': { body: DEMO_PEOPLE },
      'POST /api/v1/auth/sign-in': { body: PRIYA },
      ...NO_PLANS,
    });
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
    const api = mockApi({
      'GET /api/v1/me': { body: PRIYA },
      'PATCH /api/v1/me/preferences': async () => {
        await answers.opened;
        return { status: 401, body: SESSION_EXPIRED };
      },
      'GET /api/v1/auth/demo-people': { body: DEMO_PEOPLE },
      ...NO_PLANS,
    });
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
    mockApi({
      'GET /api/v1/me': async () => {
        checks += 1;
        if (checks === 2) {
          // The check of the next navigation: the API answers it only after the sign-out.
          await lateAnswer.opened;
        }
        return checks <= 2 ? { body: PRIYA } : { status: 401, body: SIGN_IN_REQUIRED };
      },
      'POST /api/v1/auth/sign-out': { status: 204 },
      'GET /api/v1/auth/demo-people': { body: DEMO_PEOPLE },
      ...NO_PLANS,
      ...NO_APPROVALS,
      ...HOME_ANSWERS,
    });
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
    mockApi({
      'GET /api/v1/me': () => ({ body: ++checks === 1 ? PRIYA : DANIEL }),
      ...NO_PLANS,
      ...NO_APPROVALS,
      ...HOME_ANSWERS,
    });
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
