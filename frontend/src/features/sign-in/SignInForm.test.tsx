import { screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import {
  DEMO_PEOPLE,
  emailField,
  INCORRECT,
  passwordField,
  PRIYA,
  renderSignInPage,
  signInButton,
  UNREACHABLE,
} from '@/testing/fixtures/sign-in';
import { mockApi } from '@/testing/mock-api';
import { reply } from '@/testing/replies';

afterEach(() => {
  localStorage.clear();
  delete document.documentElement.dataset.theme;
});

describe('SignInForm (FR-003, FR-004, FR-020)', () => {
  it('signs in with a typed e-mail and the password (FR-003, US1 scenario 2)', async () => {
    const api = mockApi([
      reply('GET /api/v1/auth/demo-people', { body: DEMO_PEOPLE }),
      reply('POST /api/v1/auth/sign-in', { body: PRIYA }),
    ]);
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
    mockApi([
      reply('GET /api/v1/auth/demo-people', { body: DEMO_PEOPLE }),
      reply('POST /api/v1/auth/sign-in', { status: 401, body: INCORRECT }),
    ]);
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
    mockApi([
      reply('GET /api/v1/auth/demo-people', { body: DEMO_PEOPLE }),
      reply('POST /api/v1/auth/sign-in', () => Promise.reject(new TypeError('Failed to fetch'))),
    ]);
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
    const api = mockApi([reply('GET /api/v1/auth/demo-people', { body: DEMO_PEOPLE })]);
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
    const api = mockApi([
      reply('GET /api/v1/auth/demo-people', { body: DEMO_PEOPLE }),
      reply('POST /api/v1/auth/sign-in', {
        status: 400,
        body: {
          error: {
            code: 'validation_failed',
            message: 'Some fields need your attention.',
            fields: [{ field: 'password', message: 'Enter the password' }],
          },
        },
      }),
    ]);
    const { user } = renderSignInPage();

    // Only spaces: not empty for the page, but the API may still refuse it.
    await user.type(emailField(), 'priya.raman@shell.com');
    await user.type(passwordField(), '   ');
    await user.click(signInButton());

    expect(await screen.findByRole('alert')).toHaveTextContent('1 problem needs your attention');
    expect(passwordField()).toHaveAccessibleDescription('Enter the password');
    expect(api.bodiesOf('POST /api/v1/auth/sign-in')).toHaveLength(1);
  });
});
