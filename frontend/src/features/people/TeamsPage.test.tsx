import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { mockApi } from '@/test/mockApi';
import type { MockHandler } from '@/test/mockApi';
import { findMessages, renderWithProviders } from '@/test/render';
import { testUser } from '@/test/users';
import { TeamsPage } from './TeamsPage';

const META = {
  version: 1,
  createdAt: '2026-03-01T09:00:00Z',
  createdBy: { id: 'U-1001', name: 'Alex Morgan' },
  updatedAt: '2026-03-10T14:32:00Z',
  updatedBy: { id: 'U-1002', name: 'Priya Raman' },
};

const ALEX = {
  id: 'U-1001',
  name: 'Alex Morgan',
  initials: 'AM',
  jobTitle: 'Head of marketing',
  active: true,
};
const PRIYA = {
  id: 'U-1002',
  name: 'Priya Raman',
  initials: 'PR',
  jobTitle: 'Marketer',
  active: true,
};
const DANIEL = { id: 'U-1003', name: 'Daniel Okoro', initials: 'DO', jobTitle: '', active: true };
const SARA = {
  id: 'U-1004',
  name: 'Sara Novak',
  initials: 'SN',
  jobTitle: 'Local finance advisor',
  active: true,
};
const OLD = { id: 'U-1005', name: 'Old Timer', initials: 'OT', jobTitle: 'Retired', active: false };

const EUROPE = {
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
const CHINA = {
  id: 'T-1002',
  name: 'Mobility China',
  description: 'Mobility in China',
  colour: 'teal',
  active: true,
  members: [DANIEL, SARA],
  plans: [],
  ...META,
};
const OLD_MARKETS = {
  id: 'T-1003',
  name: 'Old Markets',
  description: '',
  colour: 'gold',
  active: false,
  members: [],
  plans: [],
  ...META,
};
const TEAMS = { items: [EUROPE, CHINA, OLD_MARKETS], total: 3 };

function ref(team: typeof EUROPE | typeof CHINA) {
  return { id: team.id, name: team.name, colour: team.colour, active: team.active };
}
const PEOPLE = {
  total: 5,
  items: [
    { ...ALEX, email: 'alex.morgan@northstar.demo', teams: [ref(EUROPE)], ...META },
    { ...PRIYA, email: 'priya.raman@northstar.demo', teams: [ref(EUROPE)], ...META },
    { ...DANIEL, email: 'daniel.okoro@northstar.demo', teams: [ref(CHINA)], ...META },
    { ...SARA, email: 'sara.novak@northstar.demo', teams: [ref(CHINA)], ...META },
    { ...OLD, email: 'old.timer@northstar.demo', teams: [ref(EUROPE)], ...META },
  ],
};

/** The page at /settings/teams (or one team's address) with the answers every test needs. */
function renderTeams(answers: Record<string, MockHandler> = {}, route = '/settings/teams') {
  const api = mockApi({
    'GET /api/me': { body: testUser() },
    'GET /api/teams': { body: TEAMS },
    'GET /api/people': { body: PEOPLE },
    ...answers,
  });
  return {
    api,
    ...renderWithProviders(<TeamsPage />, {
      user: testUser(),
      route,
      path: '/settings/teams/:teamId?',
    }),
  };
}

async function chooseAction(
  user: ReturnType<typeof renderTeams>['user'],
  team: string,
  action: string,
) {
  await user.click(await screen.findByRole('button', { name: `Actions for ${team}` }));
  await user.click(screen.getByRole('menuitem', { name: action }));
}

function rowOf(name: string) {
  return screen.getByRole('row', { name: new RegExp(`^${name}`) });
}

describe('TeamsPage (feature 003, US2)', () => {
  it('lists every team with members, plans and status (FR-010)', async () => {
    renderTeams();

    expect(await screen.findByText('3 teams')).toBeInTheDocument();
    expect(screen.getByRole('heading', { level: 1, name: 'Teams' })).toBeInTheDocument();
    expect(screen.getByRole('navigation', { name: 'Breadcrumb' })).toHaveTextContent('Settings');
    const europe = rowOf('Lubricants Europe');
    expect(within(europe).getByRole('link', { name: 'Lubricants Europe' })).toHaveAttribute(
      'href',
      '/settings/teams/T-1001',
    );
    expect(europe).toHaveTextContent('Lubricants marketing for Europe');
    expect(europe).toHaveTextContent('3 members');
    expect(europe).toHaveTextContent('2 plans');
    expect(europe).toHaveTextContent('Active');
    const old = rowOf('Old Markets');
    expect(old).toHaveTextContent('Old Markets (inactive)');
    expect(old).toHaveTextContent('No members');
    expect(old).toHaveTextContent('Not used');
    expect(old).toHaveTextContent('Inactive');
  });

  it('searches names and descriptions, filters by status, and clears the filters', async () => {
    const { user } = renderTeams();
    const search = await screen.findByRole('searchbox', {
      name: 'Search team name or description',
    });

    await user.type(search, 'china');
    expect(screen.getByText('1 of 3 teams')).toBeInTheDocument();
    expect(screen.getAllByRole('row')).toHaveLength(2);

    await user.clear(search);
    await user.type(search, 'MARKETING');
    expect(rowOf('Lubricants Europe')).toBeInTheDocument();
    expect(screen.getByText('1 of 3 teams')).toBeInTheDocument();

    await user.clear(search);
    await user.click(screen.getByRole('button', { name: 'Inactive' }));
    expect(screen.getByRole('button', { name: 'Inactive' })).toHaveAttribute(
      'aria-pressed',
      'true',
    );
    expect(rowOf('Old Markets')).toBeInTheDocument();
    expect(screen.getByText('1 of 3 teams')).toBeInTheDocument();

    await user.type(search, 'china');
    expect(screen.getByText('Nothing matches these filters')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Clear filters' }));
    expect(screen.getByText('3 teams')).toBeInTheDocument();
    expect(search).toHaveValue('');
  });

  it('shows how to create the first team when there is none', async () => {
    const { user } = renderTeams({ 'GET /api/teams': { body: { items: [], total: 0 } } });

    expect(await screen.findByText('No teams yet — create your first team')).toBeInTheDocument();
    const buttons = screen.getAllByRole('button', { name: 'New team' });
    await user.click(buttons[buttons.length - 1]);

    expect(await screen.findByRole('dialog', { name: 'New team' })).toBeInTheDocument();
  });

  it('says when the teams could not be loaded, with Try again', async () => {
    let calls = 0;
    const { user } = renderTeams({
      'GET /api/teams': () => {
        calls += 1;
        return calls === 1 ? { status: 500 } : { body: TEAMS };
      },
    });

    await user.click(await screen.findByRole('button', { name: 'Try again' }));

    expect(await screen.findByText('3 teams')).toBeInTheDocument();
  });

  it('creates a team with two members picked with the person picker (US2-1)', async () => {
    const created = { ...CHINA, id: 'T-1004', name: 'Lubricants Asia', members: [ALEX, SARA] };
    const { api, user, router } = renderTeams({
      'POST /api/teams': { status: 201, body: created },
      'GET /api/teams/T-1004': { body: created },
    });

    await user.click(await screen.findByRole('button', { name: 'New team' }));
    const panel = await screen.findByRole('dialog', { name: 'New team' });
    await user.type(within(panel).getByRole('textbox', { name: /Name/ }), 'Lubricants Asia');
    await user.type(
      within(panel).getByRole('textbox', { name: /Description/ }),
      'Lubricants in Asia',
    );
    const members = await within(panel).findByRole('combobox', { name: 'Members' });
    await user.type(members, 'alex');
    await user.keyboard('{Enter}');
    await user.type(members, 'finance');
    await user.keyboard('{Enter}');
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(await screen.findByText("Team 'Lubricants Asia' created")).toBeInTheDocument();
    expect(api.bodiesOf('POST /api/teams')).toEqual([
      {
        name: 'Lubricants Asia',
        description: 'Lubricants in Asia',
        memberIds: ['U-1001', 'U-1004'],
      },
    ]);
    expect(screen.queryByRole('dialog', { name: 'New team' })).not.toBeInTheDocument();

    const messages = await findMessages();
    await user.click(within(messages).getByRole('button', { name: 'View' }));
    await waitFor(() => expect(router.state.location.pathname).toBe('/settings/teams/T-1004'));
  });

  it('shows a team with its members and the plans using it (US2-3)', async () => {
    const { user, router } = renderTeams(
      { 'GET /api/teams/T-1001': { body: EUROPE } },
      '/settings/teams/T-1001',
    );

    const panel = await screen.findByRole('dialog', { name: 'Lubricants Europe' });
    expect(await within(panel).findByText('Members (3)')).toBeInTheDocument();
    expect(within(panel).getByText('Head of marketing')).toBeInTheDocument();
    expect(within(panel).getByText('Old Timer (inactive)')).toBeInTheDocument();
    expect(within(panel).getByText('Used by 2 plans')).toBeInTheDocument();
    expect(within(panel).getByRole('link', { name: 'FY2026 Lubricants Europe' })).toHaveAttribute(
      'href',
      '/plans/P-1001',
    );
    expect(
      within(panel).getByText('FY2025 Lubricants Europe (in the recycle bin)'),
    ).toBeInTheDocument();
    expect(
      within(panel).getByText(/^Created by Alex Morgan on .* · Last changed by Priya Raman on /),
    ).toBeInTheDocument();
    expect(within(panel).getByText('Blue')).toBeInTheDocument();

    await user.click(within(panel).getByRole('button', { name: 'Close' }));
    await waitFor(() => expect(router.state.location.pathname).toBe('/settings/teams'));
  });

  it('says in the panel when the team no longer exists', async () => {
    renderTeams(
      {
        'GET /api/teams/T-1009': {
          status: 404,
          body: {
            code: 'not_found',
            message: 'This team no longer exists — someone may have deleted it.',
            fields: [],
          },
        },
      },
      '/settings/teams/T-1009',
    );

    const panel = await screen.findByRole('dialog', { name: 'Team' });
    expect(
      await within(panel).findByText('This team no longer exists — someone may have deleted it.'),
    ).toBeInTheDocument();
  });

  it('shows a taken name at the field and in the summary (US2-4)', async () => {
    const { api, user } = renderTeams({
      'PATCH /api/teams/T-1002': {
        status: 400,
        body: {
          code: 'validation_failed',
          message: 'Check the highlighted fields.',
          fields: [{ field: 'name', message: "A team called 'Lubricants Europe' already exists." }],
        },
      },
    });

    await chooseAction(user, 'Mobility China', 'Edit');
    const panel = await screen.findByRole('dialog', { name: 'Edit team' });
    const name = within(panel).getByRole('textbox', { name: /Name/ });
    await user.clear(name);
    await user.type(name, 'lubricants europe');
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(
      await within(panel).findAllByText("A team called 'Lubricants Europe' already exists."),
    ).toHaveLength(2);
    expect(api.bodiesOf('PATCH /api/teams/T-1002')).toEqual([
      {
        name: 'lubricants europe',
        description: 'Mobility in China',
        colour: 'teal',
        memberIds: ['U-1003', 'U-1004'],
        version: 1,
      },
    ]);
    expect(api.bodiesOf('POST /api/teams/T-1002/impact')).toEqual([]);
  });

  it('says who saved first after a conflict and saves with their version next time', async () => {
    let patches = 0;
    const { api, user } = renderTeams({
      'PATCH /api/teams/T-1002': () => {
        patches += 1;
        return patches === 1
          ? {
              status: 409,
              body: {
                code: 'version_conflict',
                message: 'Not saved — Daniel Okoro changed this team.',
                fields: [],
                conflict: { changedBy: 'Daniel Okoro', changedAt: '2026-03-10T14:32:00Z' },
              },
            }
          : { body: { ...CHINA, version: 3 } };
      },
      'GET /api/teams/T-1002': { body: { ...CHINA, version: 2 } },
    });

    await chooseAction(user, 'Mobility China', 'Edit');
    const panel = await screen.findByRole('dialog', { name: 'Edit team' });
    await user.type(within(panel).getByRole('textbox', { name: /Description/ }), ' and Hong Kong');
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(
      await within(panel).findByText(
        'Not saved — Daniel Okoro changed this team on 10 Mar 2026 at 14:32',
      ),
    ).toBeInTheDocument();
    expect(
      within(panel).getByText(
        'Your changes are still here. Save again to replace their changes, or Cancel to see theirs.',
      ),
    ).toBeInTheDocument();
    await waitFor(() =>
      expect(api.requests.some((r) => r.path === '/api/teams/T-1002')).toBe(true),
    );
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(await screen.findByText("Team 'Mobility China' saved")).toBeInTheDocument();
    const bodies = api.bodiesOf('PATCH /api/teams/T-1002') as {
      version: number;
      description: string;
    }[];
    expect(bodies.map((body) => body.version)).toEqual([1, 2]);
    expect(bodies[1].description).toBe('Mobility in China and Hong Kong');
  });

  it('asks before removing a member who owns a plan; Keep editing sends nothing (US2-5)', async () => {
    const warning =
      'Priya Raman owns 1 plan that uses this team: FY2026 Lubricants Europe. Priya will no longer belong to its teams, so its owner will need attention.';
    const { api, user } = renderTeams({
      'POST /api/teams/T-1001/impact': {
        body: { warnings: [{ kind: 'ownerNeedsAttention', message: warning, count: 1 }] },
      },
      'PATCH /api/teams/T-1001': { body: { ...EUROPE, members: [ALEX, OLD], version: 2 } },
    });

    await chooseAction(user, 'Lubricants Europe', 'Edit');
    const panel = await screen.findByRole('dialog', { name: 'Edit team' });
    await user.click(await within(panel).findByRole('button', { name: 'Remove Priya Raman' }));
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    const dialog = await screen.findByRole('alertdialog', {
      name: 'Some plans will need a new owner',
    });
    expect(dialog).toHaveTextContent(warning);
    await user.click(within(dialog).getByRole('button', { name: 'Keep editing' }));
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
    expect(api.bodiesOf('PATCH /api/teams/T-1001')).toEqual([]);

    await user.click(within(panel).getByRole('button', { name: 'Save' }));
    const again = await screen.findByRole('alertdialog', {
      name: 'Some plans will need a new owner',
    });
    await user.click(within(again).getByRole('button', { name: 'Save changes' }));

    expect(await screen.findByText("Team 'Lubricants Europe' saved")).toBeInTheDocument();
    const body = {
      name: 'Lubricants Europe',
      description: 'Lubricants marketing for Europe',
      colour: 'blue',
      memberIds: ['U-1001', 'U-1005'],
      version: 1,
    };
    expect(api.bodiesOf('POST /api/teams/T-1001/impact')).toEqual([body, body]);
    expect(api.bodiesOf('PATCH /api/teams/T-1001')).toEqual([body]);
  });

  it('refuses to delete a used team and offers Set team inactive (US2-6)', async () => {
    const { api, user } = renderTeams({
      'POST /api/teams/T-1001/status': { body: { ...EUROPE, active: false, version: 2 } },
    });

    await chooseAction(user, 'Lubricants Europe', 'Delete');
    const dialog = await screen.findByRole('alertdialog', {
      name: "'Lubricants Europe' can't be deleted",
    });
    expect(dialog).toHaveTextContent(
      "It is used by 2 plans: FY2026 Lubricants Europe, FY2025 Lubricants Europe (in the recycle bin). Set it inactive instead: it won't be offered for new plans, and these plans keep it, shown as 'Lubricants Europe (inactive)'.",
    );
    expect(within(dialog).getByRole('button', { name: 'Cancel' })).toHaveFocus();
    await user.click(within(dialog).getByRole('button', { name: 'Set team inactive' }));

    expect(await screen.findByText("Team 'Lubricants Europe' set inactive")).toBeInTheDocument();
    expect(api.bodiesOf('POST /api/teams/T-1001/status')).toEqual([{ active: false, version: 1 }]);
    expect(api.requests.some((request) => request.method === 'DELETE')).toBe(false);
  });

  it('deletes an unused team, and Undo brings it back (US2-7)', async () => {
    const { api, user } = renderTeams({
      'DELETE /api/teams/T-1002': {
        body: {
          recycleBinEntry: { id: 'RB-1001', recordTypeLabel: 'team', recordName: 'Mobility China' },
        },
      },
      'POST /api/recycle-bin/RB-1001/undo': { body: {} },
    });

    await chooseAction(user, 'Mobility China', 'Delete');
    const dialog = await screen.findByRole('alertdialog', {
      name: "Delete team 'Mobility China'?",
    });
    expect(dialog).toHaveTextContent(
      'It goes to the recycle bin together with its 2 team memberships. You can restore it.',
    );
    await user.click(within(dialog).getByRole('button', { name: 'Delete team' }));

    expect(
      await screen.findByText("Team 'Mobility China' moved to the recycle bin"),
    ).toBeInTheDocument();
    const messages = await findMessages();
    await user.click(within(messages).getByRole('button', { name: 'Undo' }));

    expect(await screen.findByText("Team 'Mobility China' restored")).toBeInTheDocument();
    expect(api.requests.filter((r) => r.path === '/api/recycle-bin/RB-1001/undo')).toHaveLength(1);
  });

  it('shows the plans when one started using the team meanwhile (in_use)', async () => {
    const used = {
      ...CHINA,
      plans: [{ id: 'P-1002', name: 'FY2026 Mobility China', inRecycleBin: false }],
    };
    const { user } = renderTeams({
      'DELETE /api/teams/T-1002': {
        status: 409,
        body: {
          code: 'in_use',
          message:
            "Mobility China is used by 1 plan, so it can't be deleted. Set it inactive instead.",
          fields: [],
        },
      },
      'GET /api/teams/T-1002': { body: used },
    });

    await chooseAction(user, 'Mobility China', 'Delete');
    await user.click(await screen.findByRole('button', { name: 'Delete team' }));

    const dialog = await screen.findByRole('alertdialog', {
      name: "'Mobility China' can't be deleted",
    });
    expect(dialog).toHaveTextContent('It is used by 1 plan: FY2026 Mobility China.');
  });

  it('reactivates an inactive team', async () => {
    const { api, user } = renderTeams({
      'POST /api/teams/T-1003/status': { body: { ...OLD_MARKETS, active: true, version: 2 } },
    });

    await chooseAction(user, 'Old Markets', 'Reactivate');

    expect(await screen.findByText("Team 'Old Markets' reactivated")).toBeInTheDocument();
    expect(api.bodiesOf('POST /api/teams/T-1003/status')).toEqual([{ active: true, version: 1 }]);
  });
});
