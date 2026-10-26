import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { ALEX_ROLES, CHINA, DANIEL_ROLES, SARA_ROLES, VIEWER } from '@/testing/fixtures/access';
import { renderAccess } from '@/testing/fixtures/access-page';
import { reply } from '@/testing/replies';

describe("AccessPage: a person's roles (feature 014, US3)", () => {
  it('gives Sara the Planner role for Mobility China, with the summary before saving (US3-2)', async () => {
    const saved = {
      ...SARA_ROLES,
      roles: [...SARA_ROLES.roles],
      version: 3,
    };
    const { api, user, router } = renderAccess([
      reply('GET /api/v1/people/{personId}/roles', { body: SARA_ROLES }, { personId: 'U-1006' }),
      reply('PUT /api/v1/people/{personId}/roles', { body: saved }, { personId: 'U-1006' }),
    ]);

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
    // The panel closes at once: the saved roles are not "unsaved changes".
    await waitFor(() => expect(router.state.location.search).toBe(''));
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
    expect(api.bodiesOf('PUT /api/v1/people/U-1006/roles')).toEqual([
      {
        roles: [
          { roleId: 'RL-1003', scope: { type: 'teams', teamIds: [CHINA.id] } },
          { roleId: VIEWER.id, scope: { type: 'myTeams' } },
        ],
        version: 2,
      },
    ]);
    // Adding a role takes nothing away: no impact check.
    expect(api.bodiesOf('POST /api/v1/people/U-1006/roles/impact')).toEqual([]);
  });

  it('shows the field problems of a save at their role', async () => {
    const { user } = renderAccess(
      [
        reply('GET /api/v1/people/{personId}/roles', { body: SARA_ROLES }, { personId: 'U-1006' }),
        reply(
          'PUT /api/v1/people/{personId}/roles',
          {
            status: 400,
            body: {
              error: {
                code: 'validation_failed',
                message: 'Check the highlighted fields.',
                fields: [{ field: 'roles.0.teamIds', message: 'Choose at least one team.' }],
              },
            },
          },
          { personId: 'U-1006' },
        ),
      ],
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
      [
        reply(
          'GET /api/v1/people/{personId}/roles',
          { body: DANIEL_ROLES },
          { personId: 'U-1003' },
        ),
        reply(
          'POST /api/v1/people/{personId}/roles/impact',
          {
            body: { warnings: [{ kind: 'approverCannotApprove', message: warning, count: 2 }] },
          },
          { personId: 'U-1003' },
        ),
        reply(
          'PUT /api/v1/people/{personId}/roles',
          { body: { ...DANIEL_ROLES, version: 4 } },
          { personId: 'U-1003' },
        ),
      ],
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
    expect(api.bodiesOf('PUT /api/v1/people/U-1003/roles')).toEqual([]);

    await user.click(within(panel).getByRole('button', { name: 'Save' }));
    await user.click(
      within(
        await screen.findByRole('alertdialog', { name: 'Save the roles of Daniel Okoro?' }),
      ).getByRole('button', { name: 'Save changes' }),
    );

    expect(await screen.findByText('Roles of Daniel Okoro saved')).toBeInTheDocument();
    const body = { roles: [{ roleId: 'RL-1003', scope: { type: 'myTeams' } }], version: 3 };
    expect(api.bodiesOf('POST /api/v1/people/U-1003/roles/impact')).toEqual([body, body]);
    expect(api.bodiesOf('PUT /api/v1/people/U-1003/roles')).toEqual([body]);
  });

  it('says why the only administrator keeps the role (US3-4)', async () => {
    const refusal =
      'Alex Morgan is the only administrator. Give someone else the Administrator role first.';
    const { user } = renderAccess(
      [
        reply('GET /api/v1/people/{personId}/roles', { body: ALEX_ROLES }, { personId: 'U-1001' }),
        reply(
          'POST /api/v1/people/{personId}/roles/impact',
          { body: { warnings: [] } },
          { personId: 'U-1001' },
        ),
        reply(
          'PUT /api/v1/people/{personId}/roles',
          {
            status: 409,
            body: { error: { code: 'not_allowed', message: refusal, fields: [] } },
          },
          { personId: 'U-1001' },
        ),
      ],
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
