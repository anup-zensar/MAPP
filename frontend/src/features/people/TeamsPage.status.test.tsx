import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { chooseAction } from '@/testing/fixtures/people';
import { CHINA, EUROPE, OLD_MARKETS, renderTeams } from '@/testing/fixtures/teams';
import { findMessages } from '@/testing/render';
import { reply } from '@/testing/replies';

const RESTORED = {
  recordType: 'team',
  recordTypeLabel: 'team',
  recordId: 'T-1002',
  recordName: 'Mobility China',
  details: '',
};

describe('TeamsPage (feature 003, US2): Delete, Set team inactive and Reactivate', () => {
  it('refuses to delete a used team and offers Set team inactive (US2-6)', async () => {
    const { api, user } = renderTeams([
      reply(
        'POST /api/v1/teams/{teamId}/status',
        { body: { ...EUROPE, active: false, version: 2 } },
        { teamId: 'T-1001' },
      ),
    ]);

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
    expect(api.bodiesOf('POST /api/v1/teams/T-1001/status')).toEqual([
      { active: false, version: 1 },
    ]);
    expect(api.requests.some((request) => request.method === 'DELETE')).toBe(false);
  });

  it('deletes an unused team, and Undo brings it back (US2-7)', async () => {
    const { api, user } = renderTeams([
      reply(
        'DELETE /api/v1/teams/{teamId}',
        {
          body: {
            recycleBinEntry: {
              id: 'RB-1001',
              recordType: 'team',
              recordTypeLabel: 'team',
              recordId: 'T-1002',
              recordName: 'Mobility China',
              path: [],
              parent: null,
              deletedAt: '2026-03-12T09:00:00Z',
              deletedBy: { id: 'U-1002', name: 'Priya Raman' },
              contents: [],
            },
          },
        },
        { teamId: 'T-1002' },
      ),
      reply('POST /api/v1/recycle-bin/{entryId}/undo', { body: RESTORED }, { entryId: 'RB-1001' }),
    ]);

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
    expect(api.requests.filter((r) => r.path === '/api/v1/recycle-bin/RB-1001/undo')).toHaveLength(
      1,
    );
  });

  it('shows the plans when one started using the team meanwhile (in_use)', async () => {
    const used = {
      ...CHINA,
      plans: [{ id: 'P-1002', name: 'FY2026 Mobility China', inRecycleBin: false }],
    };
    const { user } = renderTeams([
      reply(
        'DELETE /api/v1/teams/{teamId}',
        {
          status: 409,
          body: {
            error: {
              code: 'in_use',
              message:
                "Mobility China is used by 1 plan, so it can't be deleted. Set it inactive instead.",
              fields: [],
            },
          },
        },
        { teamId: 'T-1002' },
      ),
      reply('GET /api/v1/teams/{teamId}', { body: used }, { teamId: 'T-1002' }),
    ]);

    await chooseAction(user, 'Mobility China', 'Delete');
    await user.click(await screen.findByRole('button', { name: 'Delete team' }));

    const dialog = await screen.findByRole('alertdialog', {
      name: "'Mobility China' can't be deleted",
    });
    expect(dialog).toHaveTextContent('It is used by 1 plan: FY2026 Mobility China.');
  });

  it('sets an unused team inactive at once, with Undo (design system §6.7)', async () => {
    let posts = 0;
    const { api, user } = renderTeams([
      reply(
        'POST /api/v1/teams/{teamId}/status',
        () => {
          posts += 1;
          return { body: { ...CHINA, active: posts !== 1, version: posts + 1 } };
        },
        { teamId: 'T-1002' },
      ),
    ]);

    await chooseAction(user, 'Mobility China', 'Set team inactive');

    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
    const messages = await findMessages();
    expect(
      await within(messages).findByText("Team 'Mobility China' set inactive"),
    ).toBeInTheDocument();
    await user.click(within(messages).getByRole('button', { name: 'Undo' }));

    expect(await screen.findByText("Team 'Mobility China' reactivated")).toBeInTheDocument();
    expect(api.bodiesOf('POST /api/v1/teams/T-1002/status')).toEqual([
      { active: false, version: 1 },
      { active: true, version: 2 },
    ]);
  });

  it('asks before setting a team that plans use inactive, naming the plans', async () => {
    const { api, user } = renderTeams([
      reply(
        'POST /api/v1/teams/{teamId}/status',
        { body: { ...EUROPE, active: false, version: 2 } },
        { teamId: 'T-1001' },
      ),
    ]);

    await chooseAction(user, 'Lubricants Europe', 'Set team inactive');
    const dialog = await screen.findByRole('alertdialog', {
      name: "Set team 'Lubricants Europe' inactive?",
    });
    expect(dialog).toHaveTextContent(
      'Used by 2 plans: FY2026 Lubricants Europe, FY2025 Lubricants Europe.',
    );
    expect(within(dialog).getByRole('button', { name: 'Cancel' })).toHaveFocus();
    await user.click(within(dialog).getByRole('button', { name: 'Set inactive' }));

    expect(await screen.findByText("Team 'Lubricants Europe' set inactive")).toBeInTheDocument();
    expect(api.bodiesOf('POST /api/v1/teams/T-1001/status')).toEqual([
      { active: false, version: 1 },
    ]);
  });

  it('reactivates an inactive team', async () => {
    const { api, user } = renderTeams(
      [
        reply(
          'POST /api/v1/teams/{teamId}/status',
          { body: { ...OLD_MARKETS, active: true, version: 2 } },
          { teamId: 'T-1003' },
        ),
      ],
      '/settings/teams?status=inactive',
    );

    await chooseAction(user, 'Old Markets', 'Reactivate');

    expect(await screen.findByText("Team 'Old Markets' reactivated")).toBeInTheDocument();
    expect(api.bodiesOf('POST /api/v1/teams/T-1003/status')).toEqual([
      { active: true, version: 1 },
    ]);
  });
});
