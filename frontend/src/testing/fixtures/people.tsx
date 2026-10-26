/**
 * People of feature 003 for the tests of Settings › People, shaped as the API answers them, and
 * the page at /settings/people with the answers every test needs.
 */
import { screen, within } from '@testing-library/react';
import type { UserEvent } from '@testing-library/user-event';
import type { HttpHandler } from 'msw';
import type { components } from '@/api/generated/schema';
import { PeoplePage } from '@/features/people';
import { mockApi } from '../mock-api';
import { renderWithProviders } from '../render';
import { reply } from '../replies';
import { testUser } from './users';

type Person = components['schemas']['Person'];
type PersonDetails = components['schemas']['PersonDetails'];
type TeamRef = components['schemas']['TeamRef'];
type Team = components['schemas']['Team'];
type FieldError = components['schemas']['FieldError'];

export const META = {
  version: 1,
  createdAt: '2026-03-01T09:00:00Z',
  createdBy: { id: 'U-1001', name: 'Alex Morgan' },
  updatedAt: '2026-03-10T14:32:00Z',
  updatedBy: { id: 'U-1002', name: 'Priya Raman' },
};
export const EUROPE: TeamRef = {
  id: 'T-1001',
  name: 'Lubricants Europe',
  colour: 'blue',
  active: true,
};
export const CHINA: TeamRef = {
  id: 'T-1002',
  name: 'Mobility China',
  colour: 'teal',
  active: true,
};

export function person(
  id: string,
  name: string,
  jobTitle: string,
  teams: TeamRef[],
  active = true,
): Person {
  const email = `${name.toLowerCase().replace(' ', '.')}@shell.com`;
  const initials = name
    .split(' ')
    .map((part) => part.charAt(0))
    .join('');
  return { id, name, email, jobTitle, initials, active, teams, ...META };
}

export const ALEX = person('U-1001', 'Alex Morgan', 'Head of marketing', [EUROPE]);
export const PRIYA = person('U-1002', 'Priya Raman', 'Marketer', [EUROPE]);
export const DANIEL = person('U-1003', 'Daniel Okoro', '', [CHINA]);
export const SARA = person('U-1004', 'Sara Novak', 'Local finance advisor', [CHINA]);
export const JANE = person('U-1006', 'Jane Doe', 'Marketer', [], false);
export const PEOPLE = { items: [ALEX, DANIEL, JANE, PRIYA, SARA], total: 5 };

export const ALEX_DETAILS: PersonDetails = {
  ...ALEX,
  usage: [
    { role: 'owner', recordType: 'plan', label: 'plans', count: 2 },
    { role: 'owner', recordType: 'activity', label: 'activities', count: 5 },
    { role: 'approver', recordType: 'approvalWorkflow', label: 'approval workflow', count: 1 },
  ],
};

function team(ref: TeamRef): Team {
  return { ...ref, description: '', members: [], plans: [], ...META };
}
const TEAMS = { items: [team(EUROPE), team(CHINA)], total: 2 };

/** Someone who may not manage access can't read a person's roles: the Roles line is left out. */
const ROLES_FORBIDDEN = {
  status: 403,
  body: {
    error: { code: 'forbidden' as const, message: "You don't have access to this.", fields: [] },
  },
};
const ROLES_OF_EVERYONE = ['U-1001', 'U-1002', 'U-1004', 'U-1006', 'U-1007'].map((personId) =>
  reply('GET /api/v1/people/{personId}/roles', ROLES_FORBIDDEN, { personId }),
);

/** A 400 with the field problems the API found. */
export function validation(fields: FieldError[]) {
  return {
    status: 400,
    body: {
      error: {
        code: 'validation_failed' as const,
        message: 'Check the highlighted fields.',
        fields,
      },
    },
  };
}

/** The page at /settings/people (or one person's address) with the answers every test needs. */
export function renderPeople(answers: HttpHandler[] = [], route = '/settings/people') {
  const api = mockApi([
    reply('GET /api/v1/me', { body: testUser() }),
    reply('GET /api/v1/people', { body: PEOPLE }),
    reply('GET /api/v1/teams', { body: TEAMS }),
    ...ROLES_OF_EVERYONE,
    ...answers,
  ]);
  return {
    api,
    ...renderWithProviders(<PeoplePage />, {
      user: testUser(),
      route,
      path: '/settings/people/:personId?',
    }),
  };
}

export async function openActions(user: UserEvent, name: string) {
  await user.click(await screen.findByRole('button', { name: `Actions for ${name}` }));
  return screen.getByRole('menu');
}

export async function chooseAction(user: UserEvent, name: string, action: string) {
  const menu = await openActions(user, name);
  await user.click(within(menu).getByRole('menuitem', { name: action }));
}

export function rowOf(name: string) {
  return screen.getByRole('row', { name: new RegExp(`^${name}`) });
}
