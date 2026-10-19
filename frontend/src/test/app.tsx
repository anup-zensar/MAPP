import { screen, within } from '@testing-library/react';
import { expect } from 'vitest';
import { routes } from '@/app/router';
import type { CurrentUser } from '@/app/useSession';
import { mockApi } from './mockApi';
import type { MockHandler } from './mockApi';
import { renderRoutes } from './render';
import { testUser } from './users';

/**
 * Opens the whole app at `path` as a signed-in person, as a page load does: GET /api/me answers
 * `user` (testUser() by default), and `answers` are the page's own requests. Returns what
 * renderRoutes returns, plus the fake API.
 *
 *   const { api } = renderApp('/plans', { 'GET /api/plans': { body: plans } });
 */
export function renderApp(
  path: string,
  answers: Record<string, MockHandler> = {},
  user: CurrentUser = testUser(),
) {
  const api = mockApi({ 'GET /api/me': { body: user }, ...answers });
  return { ...renderRoutes(routes, { route: path }), api };
}

/** The sidebar item marked as the current section. */
export function expectCurrentSection(label: string) {
  const navigation = screen.getByRole('navigation', { name: 'Main' });
  expect(within(navigation).getByRole('link', { name: label })).toHaveAttribute(
    'aria-current',
    'page',
  );
}
