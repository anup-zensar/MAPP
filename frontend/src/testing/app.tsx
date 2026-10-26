import { screen, within } from '@testing-library/react';
import { expect } from 'vitest';
import type { HttpHandler } from 'msw';
import { routes } from '@/app/router';
import type { CurrentUser } from '@/session/useSession';
import { mockApi } from './mock-api';
import { renderRoutes } from './render';
import { testUser } from './fixtures/users';
import { reply } from './replies';

/**
 * Opens the whole app at `path` as a signed-in person, as a page load does: GET /api/v1/me answers
 * `user` (testUser() by default), and `answers` are the page's own requests. Returns what
 * renderRoutes returns, plus the fake API.
 *
 *   const { api } = renderApp('/plans', { 'GET /api/v1/plans': { body: plans } });
 */
export function renderApp(
  path: string,
  answers: HttpHandler[] = [],
  user: CurrentUser = testUser(),
) {
  const api = mockApi([reply('GET /api/v1/me', { body: user }), ...answers]);
  return { ...renderRoutes(routes, { route: path }), api };
}

/** Opens the whole app at `options.route` with the session as the test arranged it (no GET /me answer). */
export function renderAppRoutes(options: Parameters<typeof renderRoutes>[1]) {
  return renderRoutes(routes, options);
}

/** The sidebar item marked as the current section. */
export function expectCurrentSection(label: string) {
  const navigation = screen.getByRole('navigation', { name: 'Main' });
  expect(within(navigation).getByRole('link', { name: label })).toHaveAttribute(
    'aria-current',
    'page',
  );
}
