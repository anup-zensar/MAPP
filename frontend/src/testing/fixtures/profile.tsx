/**
 * My profile of feature 003 for its tests: the display currencies, Priya as a person record, and
 * the page at /profile with a line showing what the rest of the app sees of the session.
 */
import { screen } from '@testing-library/react';
import type { HttpHandler } from 'msw';
import type { components } from '@/api/generated/schema';
import { ProfilePage } from '@/features/people';
import { mockApi } from '../mock-api';
import { renderWithProviders } from '../render';
import { reply } from '../replies';
import { ElsewhereInTheApp } from './ElsewhereInTheApp';
import { testUser } from './users';

type DisplayCurrencyOption = components['schemas']['DisplayCurrencyOption'];
type PersonDetails = components['schemas']['PersonDetails'];

export const USD: DisplayCurrencyOption = { code: 'USD', name: 'US dollar', active: true };
export const EUR: DisplayCurrencyOption = { code: 'EUR', name: 'Euro', active: true };
export const CURRENCIES = { items: [EUR, USD], total: 2 };

export const PRIYA_AS_PERSON: PersonDetails = {
  id: 'U-1002',
  name: 'Priya Raman',
  email: 'priya.raman@shell.com',
  jobTitle: 'Marketer',
  initials: 'PR',
  active: true,
  teams: [{ id: 'T-1001', name: 'Lubricants Europe', colour: 'blue', active: true }],
  usage: [],
  version: 2,
  createdAt: '2026-03-01T09:00:00Z',
  createdBy: { id: 'U-1001', name: 'Alex Morgan' },
  updatedAt: '2026-03-10T14:32:00Z',
  updatedBy: { id: 'U-1001', name: 'Alex Morgan' },
};

/**
 * My profile at /profile as `person`, with the answers every test needs (GET /api/v1/me: the
 * session is asked again after every change).
 */
export function renderProfile(answers: HttpHandler[] = [], person = testUser()) {
  const api = mockApi([
    reply('GET /api/v1/me', { body: person }),
    reply('GET /api/v1/me/display-currencies', { body: CURRENCIES }),
    ...answers,
  ]);
  return {
    api,
    ...renderWithProviders(
      <>
        <ProfilePage />
        <ElsewhereInTheApp />
      </>,
      { user: person, route: '/profile', path: '/profile' },
    ),
  };
}

export function card(name: string) {
  return screen.getByRole('region', { name });
}
