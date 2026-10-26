/**
 * Teams of feature 003 for the tests of Settings › Teams, shaped as the API answers them, and the
 * page at /settings/teams with the answers every test needs.
 */
import type { HttpHandler } from 'msw';
import type { components } from '@/api/generated/schema';
import { TeamsPage } from '@/features/people';
import { mockApi } from '../mock-api';
import { renderWithProviders } from '../render';
import { reply } from '../replies';
import { META } from './people';
import { testUser } from './users';

type PersonRef = components['schemas']['PersonRef'];
type Team = components['schemas']['Team'];

export const ALEX: PersonRef = {
  id: 'U-1001',
  name: 'Alex Morgan',
  initials: 'AM',
  jobTitle: 'Head of marketing',
  active: true,
};
const PRIYA: PersonRef = {
  id: 'U-1002',
  name: 'Priya Raman',
  initials: 'PR',
  jobTitle: 'Marketer',
  active: true,
};
const DANIEL: PersonRef = {
  id: 'U-1003',
  name: 'Daniel Okoro',
  initials: 'DO',
  jobTitle: '',
  active: true,
};
export const SARA: PersonRef = {
  id: 'U-1004',
  name: 'Sara Novak',
  initials: 'SN',
  jobTitle: 'Local finance advisor',
  active: true,
};
export const OLD: PersonRef = {
  id: 'U-1005',
  name: 'Old Timer',
  initials: 'OT',
  jobTitle: 'Retired',
  active: false,
};

export const EUROPE: Team = {
  id: 'T-1001',
  name: 'Lubricants Europe',
  description: 'Lubricants marketing for Europe',
  colour: 'blue',
  active: true,
  members: [ALEX, OLD, PRIYA],
  plans: [
    { id: 'P-1001', name: 'FY2026 Lubricants Europe', inRecycleBin: false },
    { id: 'P-1009', name: 'FY2025 Lubricants Europe', inRecycleBin: true },
  ],
  ...META,
};
export const CHINA: Team = {
  id: 'T-1002',
  name: 'Mobility China',
  description: 'Mobility in China',
  colour: 'teal',
  active: true,
  members: [DANIEL, SARA],
  plans: [],
  ...META,
};
export const OLD_MARKETS: Team = {
  id: 'T-1003',
  name: 'Old Markets',
  description: '',
  colour: 'gold',
  active: false,
  members: [],
  plans: [],
  ...META,
};
export const TEAMS = { items: [EUROPE, CHINA, OLD_MARKETS], total: 3 };

function ref(team: Team) {
  return { id: team.id, name: team.name, colour: team.colour, active: team.active };
}
const PEOPLE = {
  total: 5,
  items: [
    { ...ALEX, email: 'alex.morgan@shell.com', teams: [ref(EUROPE)], ...META },
    { ...PRIYA, email: 'priya.raman@shell.com', teams: [ref(EUROPE)], ...META },
    { ...DANIEL, email: 'daniel.okoro@shell.com', teams: [ref(CHINA)], ...META },
    { ...SARA, email: 'sara.novak@shell.com', teams: [ref(CHINA)], ...META },
    { ...OLD, email: 'old.timer@shell.com', teams: [ref(EUROPE)], ...META },
  ],
};

/** The page at /settings/teams (or one team's address) with the answers every test needs. */
export function renderTeams(answers: HttpHandler[] = [], route = '/settings/teams') {
  const api = mockApi([
    reply('GET /api/v1/me', { body: testUser() }),
    reply('GET /api/v1/teams', { body: TEAMS }),
    reply('GET /api/v1/people', { body: PEOPLE }),
    ...answers,
  ]);
  return {
    api,
    ...renderWithProviders(<TeamsPage />, {
      user: testUser(),
      route,
      path: '/settings/teams/:teamId?',
    }),
  };
}
