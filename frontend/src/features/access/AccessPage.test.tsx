import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { mockApi } from '@/test/mockApi';
import type { MockHandler } from '@/test/mockApi';
import { findMessages, renderWithProviders } from '@/test/render';
import { testUser } from '@/test/users';
import { AccessPage } from './AccessPage';
import {
  ACCESS_ANSWERS,
  ALEX,
  ALEX_ROLES,
  CHINA,
  DANIEL_ROLES,
  FINANCE_REVIEWER,
  RULES,
  SARA_ROLES,
  VIEWER,
} from './testData';

const ALEX_USER = testUser({ id: ALEX.id, name: ALEX.name, firstName: 'Alex', initials: 'AM' });

function renderAccess(answers: Record<string, MockHandler> = {}, route = '/settings/access') {
  // A change refreshes the session too: the signed-in person's own access may have changed.
  const api = mockApi({ 'GET /api/me': { body: ALEX_USER }, ...ACCESS_ANSWERS, ...answers });
  return {
    api,
    ...renderWithProviders(<AccessPage />, { user: ALEX_USER, route, path: '/settings/access' }),
  };
}

function rowOf(name: string) {
  const escaped = name.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  return screen.getByRole('row', { name: new RegExp(`^${escaped}`) });
}

describe('AccessPage: People tab (feature 014, US3)', () => {
  it('lists everyone with teams and roles, with a search and a role filter with No role (US3-1)', async () => {
    const { user } = renderAccess({
      'GET /api/access/changes': {
        body: {
          items: [
            {
              id: 'LOG-1201',
              seq: 1201,
              at: '2026-10-20T10:42:00Z',
              actor: { id: ALEX.id, name: ALEX.name },
              action: 'changed',
              verb: 'changed',
              recordType: 'person',
              recordTypeLabel: 'person',
              recordId: 'U-1006',
              recordName: 'Sara Novak',
              path: [],
              changes: [],
              comment: '',
              details: '',
            },
          ],
          total: 1,
        },
      },
    });

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Roles and access' }),
    ).toBeInTheDocument();
    expect(screen.getByRole('navigation', { name: 'Breadcrumb' })).toHaveTextContent('Settings');
    expect(screen.getByRole('tab', { name: 'People' })).toHaveAttribute('aria-selected', 'true');
    expect(await screen.findByText('4 of 4 people')).toBeInTheDocument();
    expect(rowOf('Alex Morgan (you)')).toHaveTextContent('Administrator · All markets');
    expect(rowOf('Daniel Okoro')).toHaveTextContent('Approver · My teams');
    expect(rowOf('Daniel Okoro')).toHaveTextContent('Planner · My teams');
    expect(rowOf('Priya Raman')).toHaveTextContent('No role');

    await user.click(await screen.findByRole('button', { name: 'No role 1' }));
    expect(screen.getByText('1 of 4 people')).toBeInTheDocument();
    expect(rowOf('Priya Raman')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Planner 1' }));
    expect(rowOf('Daniel Okoro')).toBeInTheDocument();
    expect(screen.queryByRole('row', { name: /^Sara Novak/ })).not.toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'All 4' }));
    await user.type(screen.getByRole('searchbox', { name: 'Search name or job title' }), 'finance');
    expect(screen.getByText('1 of 4 people')).toBeInTheDocument();
    expect(rowOf('Sara Novak')).toBeInTheDocument();

    // The latest access changes, each linking to its history in the activity log (US7-3).
    const changes = screen.getByRole('region', { name: 'Recent access changes' });
    expect(
      await within(changes).findByRole('link', { name: "Alex Morgan changed person 'Sara Novak'" }),
    ).toHaveAttribute('href', '/activity-log?record=U-1006');
  });

  it('flags a scope that covers no team any more', async () => {
    renderAccess({
      'GET /api/access/people': {
        body: {
          items: [
            {
              ...SARA_ROLES,
              roles: [{ ...SARA_ROLES.roles[0], scopeEmpty: true }],
            },
          ],
          total: 1,
        },
      },
    });

    expect(await screen.findByText('No teams left in this scope')).toBeInTheDocument();
  });

  it('gives Sara the Planner role for Mobility China, with the summary before saving (US3-2)', async () => {
    const saved = {
      ...SARA_ROLES,
      roles: [...SARA_ROLES.roles],
      version: 3,
    };
    const { api, user, router } = renderAccess({
      'GET /api/people/U-1006/roles': { body: SARA_ROLES },
      'PUT /api/people/U-1006/roles': { body: saved },
    });

    await user.click(await screen.findByRole('button', { name: 'Edit roles of Sara Novak' }));
    await waitFor(() => expect(router.state.location.search).toBe('?person=U-1006'));
    const panel = await screen.findByRole('dialog', { name: 'Roles of Sara Novak' });
    expect(panel).toHaveTextContent('Teams: Lubricants Europe, Mobility China');
    expect(within(panel).getByRole('checkbox', { name: /^Viewer/ })).toBeChecked();
    const preview = within(panel).getByRole('region', { name: 'What Sara can do' });
    expect(preview).toHaveTextContent(
      'In Lubricants Europe and Mobility China: view plans and everything in them and see financial data.',
    );

    await user.click(within(panel).getByRole('checkbox', { name: /^Planner/ }));
    const scope = within(panel).getByRole('group', { name: 'Where Planner applies' });
    expect(within(scope).getByRole('radio', { name: 'My teams' })).toBeChecked();
    await user.click(within(scope).getByRole('radio', { name: 'Chosen teams' }));
    await user.type(within(scope).getByRole('combobox', { name: 'Teams' }), 'mobility');
    await user.keyboard('{Enter}');
    expect(preview).toHaveTextContent(
      'In Mobility China: create and copy plans, edit plan details, publish and unpublish plans',
    );

    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(await screen.findByText('Roles of Sara Novak saved')).toBeInTheDocument();
    expect(api.bodiesOf('PUT /api/people/U-1006/roles')).toEqual([
      {
        roles: [
          { roleId: 'RL-1003', scope: { type: 'teams', teamIds: [CHINA.id] } },
          { roleId: VIEWER.id, scope: { type: 'myTeams' } },
        ],
        version: 2,
      },
    ]);
    // Adding a role takes nothing away: no impact check.
    expect(api.bodiesOf('POST /api/people/U-1006/roles/impact')).toEqual([]);
  });

  it('shows the field problems of a save at their role', async () => {
    const { user } = renderAccess(
      {
        'GET /api/people/U-1006/roles': { body: SARA_ROLES },
        'PUT /api/people/U-1006/roles': {
          status: 400,
          body: {
            code: 'validation_failed',
            message: 'Check the highlighted fields.',
            fields: [{ field: 'roles.0.teamIds', message: 'Choose at least one team.' }],
          },
        },
      },
      '/settings/access?person=U-1006',
    );

    const panel = await screen.findByRole('dialog', { name: 'Roles of Sara Novak' });
    await user.click(within(panel).getByRole('checkbox', { name: /^Planner/ }));
    const scope = within(panel).getByRole('group', { name: 'Where Planner applies' });
    await user.click(within(scope).getByRole('radio', { name: 'Chosen teams' }));
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(await within(panel).findAllByText('Choose at least one team.')).toHaveLength(2);
  });

  it('checks the impact of taking away approving; Keep editing sends nothing (US3-5)', async () => {
    const warning =
      'Daniel Okoro is an approver in 1 approval workflow and has 2 activities waiting. Without a role that approves, an administrator will have to decide on their behalf.';
    const { api, user } = renderAccess(
      {
        'GET /api/people/U-1003/roles': { body: DANIEL_ROLES },
        'POST /api/people/U-1003/roles/impact': {
          body: { warnings: [{ kind: 'approverCannotApprove', message: warning, count: 2 }] },
        },
        'PUT /api/people/U-1003/roles': { body: { ...DANIEL_ROLES, version: 4 } },
      },
      '/settings/access?person=U-1003',
    );

    const panel = await screen.findByRole('dialog', { name: 'Roles of Daniel Okoro' });
    await user.click(within(panel).getByRole('checkbox', { name: /^Approver/ }));
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    const dialog = await screen.findByRole('alertdialog', {
      name: 'Save the roles of Daniel Okoro?',
    });
    expect(dialog).toHaveTextContent(warning);
    expect(within(dialog).getByRole('button', { name: 'Keep editing' })).toHaveFocus();
    await user.click(within(dialog).getByRole('button', { name: 'Keep editing' }));
    expect(api.bodiesOf('PUT /api/people/U-1003/roles')).toEqual([]);

    await user.click(within(panel).getByRole('button', { name: 'Save' }));
    await user.click(
      within(
        await screen.findByRole('alertdialog', { name: 'Save the roles of Daniel Okoro?' }),
      ).getByRole('button', { name: 'Save changes' }),
    );

    expect(await screen.findByText('Roles of Daniel Okoro saved')).toBeInTheDocument();
    const body = { roles: [{ roleId: 'RL-1003', scope: { type: 'myTeams' } }], version: 3 };
    expect(api.bodiesOf('POST /api/people/U-1003/roles/impact')).toEqual([body, body]);
    expect(api.bodiesOf('PUT /api/people/U-1003/roles')).toEqual([body]);
  });

  it('says why the only administrator keeps the role (US3-4)', async () => {
    const refusal =
      'Alex Morgan is the only administrator. Give someone else the Administrator role first.';
    const { user } = renderAccess(
      {
        'GET /api/people/U-1001/roles': { body: ALEX_ROLES },
        'POST /api/people/U-1001/roles/impact': { body: { warnings: [] } },
        'PUT /api/people/U-1001/roles': {
          status: 409,
          body: { code: 'not_allowed', message: refusal, fields: [] },
        },
      },
      '/settings/access?person=U-1001',
    );

    const panel = await screen.findByRole('dialog', { name: 'Roles of Alex Morgan' });
    expect(
      within(panel).getByText('The Administrator role always applies to all markets.'),
    ).toBeInTheDocument();
    await user.click(within(panel).getByRole('checkbox', { name: /^Administrator/ }));
    expect(within(panel).getByRole('region', { name: 'What Alex can do' })).toHaveTextContent(
      'No access yet.',
    );
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(await within(panel).findByRole('alert')).toHaveTextContent(refusal);
  });
});

describe('AccessPage: Roles tab (feature 014, US4)', () => {
  it('lists the roles with how many people hold them, Built in, and New role', async () => {
    renderAccess({}, '/settings/access?tab=roles');

    expect(screen.getByRole('tab', { name: 'Roles' })).toHaveAttribute('aria-selected', 'true');
    const planner = await screen.findByRole('row', { name: /^Planner/ });
    expect(within(planner).getByRole('link', { name: 'Planner' })).toHaveAttribute(
      'href',
      '/settings/access/roles/RL-1003',
    );
    const [description, scope, people] = within(planner).getAllByRole('cell');
    expect(description).toHaveTextContent('Plans for their own markets.');
    expect(scope).toHaveTextContent('My teams');
    expect(people).toHaveTextContent('2');
    expect(planner).toHaveTextContent('Built in');
    expect(screen.getByRole('link', { name: 'New role' })).toHaveAttribute(
      'href',
      '/settings/access/roles/new',
    );
  });

  it('says why a role people hold cannot be deleted (US4-4)', async () => {
    const { api, user } = renderAccess({}, '/settings/access?tab=roles');

    await user.click(await screen.findByRole('button', { name: 'Actions for Viewer' }));
    const remove = within(screen.getByRole('menu')).getByRole('menuitem', { name: 'Delete' });
    expect(remove).toHaveAccessibleDescription(
      "'Viewer' is given to 1 person. Take it away from them first.",
    );
    await user.click(remove);

    expect(
      await screen.findByText("'Viewer' is given to 1 person. Take it away from them first."),
    ).toBeInTheDocument();
    expect(api.requests.some((request) => request.method === 'DELETE')).toBe(false);
  });

  it('deletes an unused role with Undo (US4-5)', async () => {
    const { user } = renderAccess(
      {
        'DELETE /api/roles/RL-1006': {
          body: {
            recycleBinEntry: {
              id: 'RB-1101',
              recordType: 'role',
              recordTypeLabel: 'role',
              recordId: FINANCE_REVIEWER.id,
              recordName: FINANCE_REVIEWER.name,
              path: [],
              parent: null,
              contents: [],
              deletedAt: '2026-10-20T10:00:00Z',
              deletedBy: { id: ALEX.id, name: ALEX.name },
            },
          },
        },
        'POST /api/recycle-bin/RB-1101/undo': {
          body: {
            recordType: 'role',
            recordTypeLabel: 'role',
            recordId: FINANCE_REVIEWER.id,
            recordName: FINANCE_REVIEWER.name,
            details: '',
          },
        },
      },
      '/settings/access?tab=roles',
    );

    await user.click(await screen.findByRole('button', { name: 'Actions for Finance reviewer' }));
    await user.click(screen.getByRole('menuitem', { name: 'Delete' }));
    const dialog = await screen.findByRole('alertdialog', {
      name: "Delete role 'Finance reviewer'?",
    });
    await user.click(within(dialog).getByRole('button', { name: 'Delete role' }));

    const messages = await findMessages();
    expect(
      await within(messages).findByText("Role 'Finance reviewer' moved to the recycle bin"),
    ).toBeInTheDocument();
    await user.click(within(messages).getByRole('button', { name: 'Undo' }));
    expect(await screen.findByText("Role 'Finance reviewer' restored")).toBeInTheDocument();
  });

  it('offers only View and Copy for the Administrator role', async () => {
    const { user } = renderAccess({}, '/settings/access?tab=roles');

    await user.click(await screen.findByRole('button', { name: 'Actions for Administrator' }));

    expect(
      within(screen.getByRole('menu'))
        .getAllByRole('menuitem')
        .map((item) => item.firstChild?.textContent),
    ).toEqual(['View', 'Copy as new role']);
  });
});

describe('AccessPage: Rules tab (feature 014, FR-019)', () => {
  it('shows the rules and saves a change with their version', async () => {
    const { api, user } = renderAccess(
      {
        'PATCH /api/access-rules': {
          body: {
            ...RULES,
            noSelfApproval: false,
            version: 1,
            updatedAt: '2026-10-20T10:00:00Z',
            updatedBy: { id: ALEX.id, name: ALEX.name },
          },
        },
      },
      '/settings/access?tab=rules',
    );

    expect(await screen.findByText('On — someone else must approve.')).toBeInTheDocument();
    const card = screen.getByRole('region', { name: 'Access rules' });
    expect(card).toHaveTextContent('Nobody approves an activity they own or submitted');
    expect(card).toHaveTextContent('On — someone else must approve.');
    expect(card).toHaveTextContent('Viewer · My teams');
    await user.click(within(card).getByRole('button', { name: 'Edit rules' }));
    const panel = await screen.findByRole('dialog', { name: 'Edit access rules' });
    const role = within(panel).getByRole('combobox', { name: /Role for new people/ });
    expect(within(role).queryByRole('option', { name: 'Administrator' })).not.toBeInTheDocument();
    await user.click(
      within(panel).getByRole('checkbox', {
        name: /Nobody approves an activity they own or submitted/,
      }),
    );
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(await screen.findByText('Access rules saved')).toBeInTheDocument();
    expect(api.bodiesOf('PATCH /api/access-rules')).toEqual([
      { noSelfApproval: false, version: 0 },
    ]);
  });
});
