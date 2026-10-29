import { screen } from '@testing-library/react';
import { SignInPage } from '@/features/sign-in';
import type { CurrentUser } from '@/session/useSession';
import { renderWithProviders } from '../render';
import type { RenderOptions } from '../render';
import { testUser } from './users';

export const PRIYA: CurrentUser = testUser();

export const DEMO_PEOPLE = {
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

export const INCORRECT = {
  error: {
    code: 'incorrect_credentials' as const,
    message: 'Incorrect e-mail or password.',
    fields: [],
  },
};

export const UNREACHABLE =
  "SP&A One can't reach its server right now. Check that it is running and try again.";

/** The sign-in page at /sign-in for a visitor who is not signed in (unless `user` says otherwise). */
export function renderSignInPage({
  route = '/sign-in?next=%2Fplans',
  user = null,
}: Pick<RenderOptions, 'route' | 'user'> = {}) {
  return renderWithProviders(<SignInPage />, { route, user, path: '/sign-in' });
}

export function emailField() {
  return screen.getByLabelText('E-mail');
}

export function passwordField() {
  return screen.getByLabelText('Password');
}

/** The form's button: "Sign in", or "Signing in…" while waiting (not the demo-person buttons). */
export function signInButton() {
  return screen.getByRole('button', { name: /^(Sign in|Signing in…)$/ });
}
