import { act, screen, waitFor, within } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { routes } from '@/app/router';
import type { CurrentUser } from '@/session/useSession';
import { mockApi } from '@/testing/mockApi';
import { renderRoutes, renderWithProviders } from '@/testing/render';
import type { RenderOptions } from '@/testing/render';
import { SignInPage } from './SignInPage';
import { testUser } from '@/testing/fixtures/users';

const PRIYA: CurrentUser = testUser();

const DEMO_PEOPLE = {
  password: 'mapp',
  people: [
    {
      id: 'U-1001',
      name: 'Alex Morgan',
      email: 'alex.morgan@shell.com',
      jobTitle: 'Marketing administrator',
      initials: 'AM',
    },
    {
      id: 'U-1002',
      name: 'Priya Raman',
      email: 'priya.raman@shell.com',
      jobTitle: 'Marketer',
      initials: 'PR',
    },
  ],
};

const INCORRECT = {
  error: {
    code: 'incorrect_credentials',
    message: 'Incorrect e-mail or password.',
    fields: [],
  },
};

const UNREACHABLE =
  "MAPP can't reach its server right now. Check that it is running and try again.";

/** The sign-in page at /sign-in for a visitor who is not signed in (unless `user` says otherwise). */
function renderSignInPage({
  route = '/sign-in?next=%2Fplans',
  user = null,
}: Pick<RenderOptions, 'route' | 'user'> = {}) {
  return renderWithProviders(<SignInPage />, { route, user, path: '/sign-in' });
}

function emailField() {
  return screen.getByLabelText('E-mail');
}

function passwordField() {
  return screen.getByLabelText('Password');
}

/** The form's button: "Sign in", or "Signing in…" while waiting (not the demo-person buttons). */
function signInButton() {
  return screen.getByRole('button', { name: /^(Sign in|Signing in…)$/ });
}

afterEach(() => {
  localStorage.clear();
  delete document.documentElement.dataset.theme;
});

describe('SignInPage', () => {
  it('welcomes people and says plainly that this is a demo sign-in with the shared password (FR-001, FR-002)', async () => {
    mockApi({ 'GET /api/v1/auth/demo-people': { body: DEMO_PEOPLE } });
    renderSignInPage();

    expect(screen.getByRole('heading', { level: 1, name: 'Sign in to MAPP' })).toBeInTheDocument();
    expect(
      screen.getByText('Plan your marketing work, from plan to line item.'),
    ).toBeInTheDocument();
    expect(screen.getByText('Demo sign-in — Shell single sign-on comes later')).toBeInTheDocument();
    expect(
      await screen.findByText(
        (_text, element) =>
          element?.tagName === 'P' &&
          element.textContent === 'All demo people share the password mapp.',
      ),
    ).toBeInTheDocument();
    expect(emailField()).toHaveAttribute('type', 'email');
    expect(emailField()).toHaveAttribute('autocomplete', 'username');
    expect(passwordField()).toHaveAttribute('autocomplete', 'current-password');
    expect(screen.getByRole('button', { name: 'Show password' })).toBeInTheDocument();
  });

  it('lists every demo person with their job title under "Explore as a demo person" (FR-001)', async () => {
    mockApi({ 'GET /api/v1/auth/demo-people': { body: DEMO_PEOPLE } });
    renderSignInPage();

    const section = screen.getByRole('region', { name: 'Explore as a demo person' });
    expect(
      within(section).getByText('Choose a person to sign in as them — no typing needed.'),
    ).toBeInTheDocument();
    const alex = await within(section).findByRole('button', {
      name: 'Sign in as Alex Morgan, Marketing administrator',
    });
    expect(alex).toHaveTextContent('Alex Morgan');
    expect(alex).toHaveTextContent('Marketing administrator');
    expect(
      within(section).getByRole('button', { name: 'Sign in as Priya Raman, Marketer' }),
    ).toBeInTheDocument();
  });

  it('names a demo person without a job title by their name alone (FR-001)', async () => {
    mockApi({
      'GET /api/v1/auth/demo-people': {
        body: {
          password: 'mapp',
          people: [
            {
              id: 'U-1006',
              name: 'Sara Novak',
              email: 'sara.novak@shell.com',
              jobTitle: '',
              initials: 'SN',
            },
          ],
        },
      },
    });
    renderSignInPage();

    expect(await screen.findByRole('button', { name: 'Sign in as Sara Novak' })).toHaveTextContent(
      'Sara Novak',
    );
  });

  it('shows loading rows while the demo people are on their way (plan "Sign-in page")', () => {
    mockApi({ 'GET /api/v1/auth/demo-people': () => new Promise<never>(() => {}) });
    renderSignInPage();

    const section = screen.getByRole('region', { name: 'Explore as a demo person' });
    expect(within(section).getByRole('status')).toHaveTextContent('Loading…');
  });

  it('signs in as a demo person with one click and goes to the page asked for (FR-003, US1 scenario 3)', async () => {
    const api = mockApi({
      'GET /api/v1/auth/demo-people': { body: DEMO_PEOPLE },
      'POST /api/v1/auth/sign-in': { body: PRIYA },
    });
    const { user, router } = renderSignInPage();

    await user.click(
      await screen.findByRole('button', { name: 'Sign in as Priya Raman, Marketer' }),
    );

    await waitFor(() => expect(router.state.location.pathname).toBe('/plans'));
    expect(router.state.historyAction).toBe('REPLACE');
    expect(api.bodiesOf('POST /api/v1/auth/sign-in')).toEqual([
      { email: 'priya.raman@shell.com', password: 'mapp' },
    ]);
  });

  it('signs in with a typed e-mail and the password (FR-003, US1 scenario 2)', async () => {
    const api = mockApi({
      'GET /api/v1/auth/demo-people': { body: DEMO_PEOPLE },
      'POST /api/v1/auth/sign-in': { body: PRIYA },
    });
    const { user, router } = renderSignInPage();

    // The browser drops the spaces around an e-mail, and the API ignores its letter case.
    await user.type(emailField(), ' PRIYA.RAMAN@shell.com ');
    await user.type(passwordField(), 'mapp');
    await user.click(signInButton());

    await waitFor(() => expect(router.state.location.pathname).toBe('/plans'));
    expect(api.bodiesOf('POST /api/v1/auth/sign-in')).toEqual([
      { email: 'PRIYA.RAMAN@shell.com', password: 'mapp' },
    ]);
  });

  it('says "Incorrect e-mail or password.", keeps the e-mail and clears the password (FR-004, US1 scenario 4)', async () => {
    mockApi({
      'GET /api/v1/auth/demo-people': { body: DEMO_PEOPLE },
      'POST /api/v1/auth/sign-in': { status: 401, body: INCORRECT },
    });
    const { user, router } = renderSignInPage();

    await user.type(emailField(), 'priya.raman@shell.com');
    await user.type(passwordField(), 'wrong');
    await user.click(signInButton());

    expect(await screen.findByRole('alert')).toHaveTextContent('Incorrect e-mail or password.');
    expect(emailField()).toHaveValue('priya.raman@shell.com');
    expect(passwordField()).toHaveValue('');
    expect(router.state.location.pathname).toBe('/sign-in');
  });

  it('says in plain words when the API cannot be reached while signing in, keeping what was typed (FR-020)', async () => {
    mockApi({
      'GET /api/v1/auth/demo-people': { body: DEMO_PEOPLE },
      'POST /api/v1/auth/sign-in': () => Promise.reject(new TypeError('Failed to fetch')),
    });
    const { user, router } = renderSignInPage();

    await user.type(emailField(), 'priya.raman@shell.com');
    await user.type(passwordField(), 'mapp');
    await user.click(signInButton());

    expect(await screen.findByRole('alert')).toHaveTextContent(UNREACHABLE);
    expect(emailField()).toHaveValue('priya.raman@shell.com');
    expect(passwordField()).toHaveValue('mapp');
    expect(router.state.location.pathname).toBe('/sign-in');
  });

  it('reports every empty field at once, before asking the API, and moves the focus to the summary (FR-004, US1 scenario 5)', async () => {
    const api = mockApi({ 'GET /api/v1/auth/demo-people': { body: DEMO_PEOPLE } });
    const { user } = renderSignInPage();
    await screen.findByRole('button', { name: 'Sign in as Priya Raman, Marketer' });

    await user.click(signInButton());

    const summary = screen.getByRole('alert');
    expect(summary).toHaveTextContent('2 problems need your attention');
    expect(summary).toHaveTextContent('Enter your e-mail');
    expect(summary).toHaveTextContent('Enter the password');
    expect(summary.parentElement).toHaveFocus();
    expect(emailField()).toHaveAccessibleDescription('Enter your e-mail');
    expect(passwordField()).toHaveAccessibleDescription('Enter the password');
    expect(api.bodiesOf('POST /api/v1/auth/sign-in')).toEqual([]);
  });

  it('shows the field problems the API reports, at the fields and in the summary (FR-020)', async () => {
    const api = mockApi({
      'GET /api/v1/auth/demo-people': { body: DEMO_PEOPLE },
      'POST /api/v1/auth/sign-in': {
        status: 400,
        body: {
          error: {
            code: 'validation_failed',
            message: 'Some fields need your attention.',
            fields: [{ field: 'password', message: 'Enter the password' }],
          },
        },
      },
    });
    const { user } = renderSignInPage();

    // Only spaces: not empty for the page, but the API may still refuse it.
    await user.type(emailField(), 'priya.raman@shell.com');
    await user.type(passwordField(), '   ');
    await user.click(signInButton());

    expect(await screen.findByRole('alert')).toHaveTextContent('1 problem needs your attention');
    expect(passwordField()).toHaveAccessibleDescription('Enter the password');
    expect(api.bodiesOf('POST /api/v1/auth/sign-in')).toHaveLength(1);
  });

  it('ignores clicks while a sign-in is on its way, without greying anything out (plan "Sign-in page")', async () => {
    let answer = () => {};
    const answered = new Promise<void>((resolve) => {
      answer = resolve;
    });
    const api = mockApi({
      'GET /api/v1/auth/demo-people': { body: DEMO_PEOPLE },
      'POST /api/v1/auth/sign-in': async () => {
        await answered;
        return { body: PRIYA };
      },
    });
    const { user, router } = renderSignInPage();
    const priya = await screen.findByRole('button', { name: 'Sign in as Priya Raman, Marketer' });

    await user.click(priya);

    expect(signInButton()).toHaveTextContent('Signing in…');
    expect(signInButton().closest('form')).toHaveAttribute('aria-busy', 'true');
    await user.click(
      screen.getByRole('button', { name: 'Sign in as Alex Morgan, Marketing administrator' }),
    );
    await user.click(signInButton());
    expect(priya).toBeEnabled();
    expect(signInButton()).toBeEnabled();
    expect(api.bodiesOf('POST /api/v1/auth/sign-in')).toHaveLength(1);

    act(() => answer());
    await waitFor(() => expect(router.state.location.pathname).toBe('/plans'));
  });

  it('explains what to do when no demo people exist, and keeps the form (FR-028)', async () => {
    mockApi({ 'GET /api/v1/auth/demo-people': { body: { password: 'mapp', people: [] } } });
    renderSignInPage();

    expect(
      await screen.findByText('No demo users found — load the demo data first'),
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        (_text, element) =>
          element?.tagName === 'P' &&
          element.textContent ===
            "Stop MAPP's API, run go run ./cmd/seed in the backend folder, start the API again with go run ./cmd/api, then reload this page.",
      ),
    ).toBeInTheDocument();
    expect(emailField()).toBeInTheDocument();
  });

  it('says that demo sign-in is off, and hides the form and the people (FR-009)', async () => {
    mockApi({
      'GET /api/v1/auth/demo-people': {
        status: 404,
        body: {
          error: { code: 'not_found', message: "This page or record doesn't exist.", fields: [] },
        },
      },
    });
    renderSignInPage();

    expect(
      await screen.findByText(
        'Demo sign-in is turned off in this environment. Shell single sign-on comes later.',
      ),
    ).toBeInTheDocument();
    expect(screen.queryByLabelText('E-mail')).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /^Sign in/ })).not.toBeInTheDocument();
    expect(screen.queryByText('Explore as a demo person')).not.toBeInTheDocument();
    expect(
      screen.queryByText('Demo sign-in — Shell single sign-on comes later'),
    ).not.toBeInTheDocument();
  });

  it('offers Try again when the demo people cannot be loaded (FR-020, edge case)', async () => {
    let calls = 0;
    mockApi({
      'GET /api/v1/auth/demo-people': () =>
        ++calls === 1 ? Promise.reject(new TypeError('Failed to fetch')) : { body: DEMO_PEOPLE },
    });
    const { user } = renderSignInPage();

    expect(await screen.findByRole('alert')).toHaveTextContent(UNREACHABLE);
    await user.click(screen.getByRole('button', { name: 'Try again' }));

    expect(
      await screen.findByRole('button', { name: 'Sign in as Priya Raman, Marketer' }),
    ).toBeInTheDocument();
  });

  it.each([
    ['signedOut', 'You have been signed out.'],
    ['sessionLost', 'Please sign in again'],
  ])('announces the notice the page was opened with: %s (US4)', async (notice, text) => {
    mockApi({ 'GET /api/v1/auth/demo-people': { body: DEMO_PEOPLE } });
    renderSignInPage({ route: { pathname: '/sign-in', state: { notice } } });

    // The status region is there at once, and empty, so screen readers announce the notice.
    const status = screen.getAllByRole('status')[0];
    expect(status).toBeEmptyDOMElement();
    await waitFor(() => expect(status).toHaveTextContent(text));
  });

  it('sends a signed-in visitor on to the page asked for (FR-005)', async () => {
    const { router } = renderSignInPage({ user: PRIYA });

    await waitFor(() => expect(router.state.location.pathname).toBe('/plans'));
  });

  it('never sends anyone to another site after signing in (FR-005)', async () => {
    mockApi({
      'GET /api/v1/auth/demo-people': { body: DEMO_PEOPLE },
      'POST /api/v1/auth/sign-in': { body: PRIYA },
    });
    const { user, router } = renderSignInPage({ route: '/sign-in?next=https%3A%2F%2Fevil.com' });

    await user.click(
      await screen.findByRole('button', { name: 'Sign in as Priya Raman, Marketer' }),
    );

    await waitFor(() => expect(router.state.location.pathname).toBe('/'));
  });

  it('names the page "Sign in · MAPP" in the browser tab (FR-017)', () => {
    mockApi({ 'GET /api/v1/auth/demo-people': { body: DEMO_PEOPLE } });
    renderSignInPage();

    expect(document.title).toBe('Sign in · MAPP');
  });
});

describe('SignInPage while the session is checked (plan "Session")', () => {
  it('shows the full-page loading placeholder until the session is known', async () => {
    mockApi({ 'GET /api/v1/me': () => new Promise<never>(() => {}) });
    renderRoutes(routes, { route: '/sign-in' });

    expect(await screen.findByRole('status')).toHaveTextContent('Loading…');
    expect(screen.queryByLabelText('E-mail')).not.toBeInTheDocument();
    expect(document.title).toBe('Sign in · MAPP');
  });

  it('shows the full-page error when the API cannot be reached (edge case)', async () => {
    mockApi({ 'GET /api/v1/me': () => Promise.reject(new TypeError('Failed to fetch')) });
    renderRoutes(routes, { route: '/sign-in' });

    expect(await screen.findByRole('alert')).toHaveTextContent(UNREACHABLE);
    expect(screen.getByRole('button', { name: 'Try again' })).toBeInTheDocument();
  });
});
