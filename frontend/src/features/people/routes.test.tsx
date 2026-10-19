import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { expectCurrentSection, renderApp } from '@/test/app';
import type { MockHandler } from '@/test/mockApi';

const META = {
  version: 1,
  createdAt: '2026-03-01T09:00:00Z',
  createdBy: { id: 'U-1001', name: 'Alex Morgan' },
  updatedAt: '2026-03-01T09:00:00Z',
  updatedBy: { id: 'U-1001', name: 'Alex Morgan' },
};
const EUROPE = {
  id: 'T-1001',
  name: 'Lubricants Europe',
  description: '',
  colour: 'blue',
  active: true,
  members: [],
  plans: [],
  ...META,
};
const TEAMS: Record<string, MockHandler> = {
  'GET /api/teams': { body: { items: [EUROPE], total: 1 } },
  'GET /api/teams/T-1001': { body: EUROPE },
};
const PRIYA = {
  id: 'U-1002',
  name: 'Priya Raman',
  email: 'priya.raman@northstar.demo',
  jobTitle: 'Marketer',
  initials: 'PR',
  active: true,
  teams: [],
  ...META,
};
const PEOPLE: Record<string, MockHandler> = {
  'GET /api/people': { body: { items: [PRIYA], total: 1 } },
  'GET /api/people/U-1002': { body: { ...PRIYA, usage: [] } },
};

// Feature 003's addresses (build guide §5). Feature 003 keeps this file up to date as it replaces
// the stub pages (their own requests go into renderApp's answers).
describe('People, teams and profile routes (feature 003)', () => {
  it.each([
    ['/settings/people', 'People', PEOPLE],
    ['/settings/teams', 'Teams', TEAMS],
  ])('opens %s: "%s", with Settings current in the sidebar', async (path, title, answers) => {
    renderApp(path, answers);

    expect(await screen.findByRole('heading', { level: 1, name: title })).toBeInTheDocument();
    expect(document.title).toBe(`${title} · NorthStar`);
    expectCurrentSection('Settings');
  });

  it.each([
    ['/settings/people/U-1002', 'People', 'Priya Raman', PEOPLE],
    ['/settings/teams/T-1001', 'Teams', 'Lubricants Europe', TEAMS],
  ])('opens %s in its details panel over the %s page', async (path, title, record, answers) => {
    renderApp(path, answers);

    expect(await screen.findByRole('dialog', { name: record })).toBeInTheDocument();
    expect(document.title).toBe(`${title} · NorthStar`);
  });

  it('opens My profile, which is not in the sidebar (FR-012 of 001)', async () => {
    renderApp('/profile');

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Your profile' }),
    ).toBeInTheDocument();
    expect(document.title).toBe('My profile · NorthStar');
  });
});
