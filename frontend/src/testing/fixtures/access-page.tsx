/**
 * Settings › Roles and access for the tests of feature 014: the page at /settings/access, opened
 * by Alex (an administrator), with the demo roles and people of fixtures/access.ts.
 */
import { screen } from '@testing-library/react';
import type { HttpHandler } from 'msw';
import { AccessPage } from '@/features/people';
import { mockApi } from '../mock-api';
import { renderWithProviders } from '../render';
import { reply } from '../replies';
import { ACCESS_ANSWERS, ALEX } from './access';
import { testUser } from './users';

export const ALEX_USER = testUser({
  id: ALEX.id,
  name: ALEX.name,
  firstName: 'Alex',
  initials: 'AM',
});

export function renderAccess(answers: HttpHandler[] = [], route = '/settings/access') {
  // A change refreshes the session too: the signed-in person's own access may have changed.
  const api = mockApi([
    reply('GET /api/v1/me', { body: ALEX_USER }),
    ...ACCESS_ANSWERS,
    ...answers,
  ]);
  return {
    api,
    ...renderWithProviders(<AccessPage />, { user: ALEX_USER, route, path: '/settings/access' }),
  };
}

export function rowOf(name: string) {
  const escaped = name.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  return screen.getByRole('row', { name: new RegExp(`^${escaped}`) });
}
