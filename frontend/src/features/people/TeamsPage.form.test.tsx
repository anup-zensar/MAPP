import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { chooseAction } from '@/testing/fixtures/people';
import { ALEX, CHINA, EUROPE, OLD, renderTeams, SARA } from '@/testing/fixtures/teams';
import { findMessages } from '@/testing/render';
import { reply } from '@/testing/replies';

describe('TeamsPage (feature 003, US2): New team and Edit team', () => {
  it('creates a team with two members picked with the person picker (US2-1)', async () => {
    const created = { ...CHINA, id: 'T-1004', name: 'Lubricants Asia', members: [ALEX, SARA] };
    const { api, user, router } = renderTeams([
      reply('POST /api/v1/teams', { status: 201, body: created }),
      reply('GET /api/v1/teams/{teamId}', { body: created }, { teamId: 'T-1004' }),
    ]);

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
    expect(api.bodiesOf('POST /api/v1/teams')).toEqual([
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

  it('shows a taken name at the field and in the summary (US2-4)', async () => {
    const { api, user } = renderTeams([
      reply(
        'PATCH /api/v1/teams/{teamId}',
        {
          status: 400,
          body: {
            error: {
              code: 'validation_failed',
              message: 'Check the highlighted fields.',
              fields: [
                { field: 'name', message: "A team called 'Lubricants Europe' already exists." },
              ],
            },
          },
        },
        { teamId: 'T-1002' },
      ),
    ]);

    await chooseAction(user, 'Mobility China', 'Edit');
    const panel = await screen.findByRole('dialog', { name: 'Edit team' });
    const name = within(panel).getByRole('textbox', { name: /Name/ });
    await user.clear(name);
    await user.type(name, 'lubricants europe');
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(
      await within(panel).findAllByText("A team called 'Lubricants Europe' already exists."),
    ).toHaveLength(2);
    // Only the changed field and the version (D-210).
    expect(api.bodiesOf('PATCH /api/v1/teams/T-1002')).toEqual([
      { name: 'lubricants europe', version: 1 },
    ]);
    expect(api.bodiesOf('POST /api/v1/teams/T-1002/impact')).toEqual([]);
  });

  it('says who saved first after a conflict and saves with their version next time', async () => {
    let patches = 0;
    const { api, user } = renderTeams([
      reply(
        'PATCH /api/v1/teams/{teamId}',
        () => {
          patches += 1;
          return patches === 1
            ? {
                status: 409,
                body: {
                  error: {
                    code: 'version_conflict',
                    message: 'Not saved — Daniel Okoro changed this team.',
                    fields: [],
                    conflict: { changedBy: 'Daniel Okoro', changedAt: '2026-03-10T14:32:00Z' },
                  },
                },
              }
            : { body: { ...CHINA, version: 3 } };
        },
        { teamId: 'T-1002' },
      ),
      reply('GET /api/v1/teams/{teamId}', { body: { ...CHINA, version: 2 } }, { teamId: 'T-1002' }),
    ]);

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
      expect(api.requests.some((r) => r.path === '/api/v1/teams/T-1002')).toBe(true),
    );
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(await screen.findByText("Team 'Mobility China' saved")).toBeInTheDocument();
    expect(api.bodiesOf('PATCH /api/v1/teams/T-1002')).toEqual([
      expect.objectContaining({ version: 1 }),
      expect.objectContaining({ version: 2, description: 'Mobility in China and Hong Kong' }),
    ]);
  });

  it('asks before removing a member who owns a plan; Keep editing sends nothing (US2-5)', async () => {
    const warning =
      'Priya Raman owns 1 plan that uses this team: FY2026 Lubricants Europe. Priya will no longer belong to its teams, so its owner will need attention.';
    const { api, user } = renderTeams([
      reply(
        'POST /api/v1/teams/{teamId}/impact',
        {
          body: { warnings: [{ kind: 'ownerNeedsAttention', message: warning, count: 1 }] },
        },
        { teamId: 'T-1001' },
      ),
      reply(
        'PATCH /api/v1/teams/{teamId}',
        { body: { ...EUROPE, members: [ALEX, OLD], version: 2 } },
        { teamId: 'T-1001' },
      ),
    ]);

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
    expect(api.bodiesOf('PATCH /api/v1/teams/T-1001')).toEqual([]);

    await user.click(within(panel).getByRole('button', { name: 'Save' }));
    const again = await screen.findByRole('alertdialog', {
      name: 'Some plans will need a new owner',
    });
    await user.click(within(again).getByRole('button', { name: 'Save changes' }));

    expect(await screen.findByText("Team 'Lubricants Europe' saved")).toBeInTheDocument();
    const body = { memberIds: ['U-1001', 'U-1005'], version: 1 }; // only what changed (D-210)
    expect(api.bodiesOf('POST /api/v1/teams/T-1001/impact')).toEqual([body, body]);
    expect(api.bodiesOf('PATCH /api/v1/teams/T-1001')).toEqual([body]);
  });
});
