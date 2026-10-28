import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { HttpHandler } from 'msw';
import { expectCurrentSection, renderApp } from '@/testing/app';
import { testUser } from '@/testing/fixtures/users';
import { reply } from '@/testing/replies';

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
  colour: 'blue' as const,
  active: true,
  members: [],
  plans: [],
  ...META,
};
const TEAMS: HttpHandler[] = [
  reply('GET /api/v1/teams', { body: { items: [EUROPE], total: 1 } }),
  reply('GET /api/v1/teams/{teamId}', { body: EUROPE }, { teamId: 'T-1001' }),
];
const PRIYA = {
  id: 'U-1002',
  name: 'Priya Raman',
  email: 'priya.raman@shell.com',
  jobTitle: 'Marketer',
  initials: 'PR',
  active: true,
  teams: [],
  ...META,
};
const PEOPLE: HttpHandler[] = [
  reply('GET /api/v1/people', { body: { items: [PRIYA], total: 1 } }),
  reply('GET /api/v1/people/{personId}', { body: { ...PRIYA, usage: [] } }, { personId: 'U-1002' }),
  reply(
    'GET /api/v1/people/{personId}/roles',
    {
      status: 403,
      body: { error: { code: 'forbidden', message: 'No access', fields: [] } },
    },
    { personId: 'U-1002' },
  ),
];
const CURRENCIES: HttpHandler[] = [
  reply('GET /api/v1/me/display-currencies', {
    body: { items: [{ code: 'USD', name: 'US dollar', active: true }], total: 1 },
  }),
];

// Feature 003's addresses (build guide §5). Feature 003 keeps this file up to date as it replaces
// the stub pages (their own requests go into renderApp's answers).
describe('People, teams and profile routes (feature 003)', () => {
  it.each([
    ['/settings/people', 'People', PEOPLE],
    ['/settings/teams', 'Teams', TEAMS],
  ])('opens %s: "%s", with Settings current in the sidebar', async (path, title, answers) => {
    renderApp(path, answers);

    expect(await screen.findByRole('heading', { level: 1, name: title })).toBeInTheDocument();
    expect(document.title).toBe(`${title} · SP&A One`);
    expectCurrentSection('Settings');
  });

  it.each([
    ['/settings/people/U-1002', 'People', 'Priya Raman', PEOPLE],
    ['/settings/teams/T-1001', 'Teams', 'Lubricants Europe', TEAMS],
  ])('opens %s in its details panel over the %s page', async (path, title, record, answers) => {
    renderApp(path, answers);

    expect(await screen.findByRole('dialog', { name: record })).toBeInTheDocument();
    expect(document.title).toBe(`${title} · SP&A One`);
  });

  it('opens My profile, which is not in the sidebar (FR-012 of 001, FR-021)', async () => {
    renderApp('/profile', CURRENCIES);

    expect(
      await screen.findByRole('heading', { level: 1, name: 'My profile' }),
    ).toBeInTheDocument();
    expect(document.title).toBe('My profile · SP&A One');
    expect(await screen.findByText('USD — US dollar')).toBeInTheDocument();
  });

  it('shows a new name on the account button as soon as My profile saves it (US4-2)', async () => {
    const renamed = testUser({
      name: 'Priya Iyer',
      firstName: 'Priya',
      initials: 'PI',
      version: 2,
    });
    let saved = false;
    const { user } = renderApp('/profile', [
      ...CURRENCIES,
      reply('PATCH /api/v1/me', () => {
        saved = true;
        return { body: renamed };
      }),
      reply('GET /api/v1/me', () => ({ body: saved ? renamed : testUser() })),
    ]);
    const accountButtons = () => screen.getAllByRole('button', { name: 'Priya — account menu' });
    const editDetails = await screen.findByRole('button', { name: 'Edit details' });
    expect(accountButtons()[0]).toHaveTextContent('PR');

    await user.click(editDetails);
    const panel = await screen.findByRole('dialog', { name: 'Edit your details' });
    const name = within(panel).getByRole('textbox', { name: /Name/ });
    await user.clear(name);
    await user.type(name, 'Priya Iyer');
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(await screen.findByText('Your details saved')).toBeInTheDocument();
    await waitFor(() =>
      expect(screen.queryByRole('dialog', { name: 'Edit your details' })).not.toBeInTheDocument(),
    );
    for (const button of accountButtons()) {
      expect(button).toHaveTextContent('PI');
    }
  });
});
