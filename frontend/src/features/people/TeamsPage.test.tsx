import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { nth } from '@/testing/dom';
import { rowOf } from '@/testing/fixtures/people';
import { EUROPE, renderTeams, TEAMS } from '@/testing/fixtures/teams';
import { reply } from '@/testing/replies';

describe("TeamsPage (feature 003, US2): the list and a team's details", () => {
  it('lists every team with members, plans and status (FR-010)', async () => {
    renderTeams([], '/settings/teams?status=all');

    expect(await screen.findByText('3 of 3 teams')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'All 3' })).toHaveAttribute('aria-pressed', 'true');
    expect(screen.getByRole('heading', { level: 1, name: 'Teams' })).toBeInTheDocument();
    expect(screen.getByRole('navigation', { name: 'Breadcrumb' })).toHaveTextContent('Settings');
    const europe = rowOf('Lubricants Europe');
    expect(within(europe).getByRole('link', { name: 'Lubricants Europe' })).toHaveAttribute(
      'href',
      '/settings/teams/T-1001',
    );
    // Description, Members, Plans (counts, right-aligned under their headers), Status.
    const [description, members, plans, status] = within(europe).getAllByRole('cell');
    expect(description).toHaveTextContent('Lubricants marketing for Europe');
    expect(members).toHaveTextContent(/3$/);
    expect(plans).toHaveTextContent(/^2$/);
    expect(status).toHaveTextContent('Active');
    const old = rowOf('Old Markets');
    expect(old).toHaveTextContent('Old Markets (inactive)');
    const [, oldMembers, oldPlans, oldStatus] = within(old).getAllByRole('cell');
    expect(oldMembers).toHaveTextContent(/^0$/);
    expect(oldPlans).toHaveTextContent(/^0$/);
    expect(oldStatus).toHaveTextContent('Inactive');
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
    await user.click(screen.getByRole('button', { name: /^Inactive/ }));
    expect(screen.getByRole('button', { name: /^Inactive/ })).toHaveAttribute(
      'aria-pressed',
      'true',
    );
    expect(rowOf('Old Markets')).toBeInTheDocument();
    expect(screen.getByText('1 of 3 teams')).toBeInTheDocument();

    await user.type(search, 'china');
    expect(screen.getByText('Nothing matches these filters')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Clear filters' }));
    expect(screen.getByText('3 of 3 teams')).toBeInTheDocument();
    expect(search).toHaveValue('');
  });

  it('shows how to create the first team when there is none', async () => {
    const { user } = renderTeams([reply('GET /api/v1/teams', { body: { items: [], total: 0 } })]);

    expect(await screen.findByText('No teams yet — create your first team')).toBeInTheDocument();
    const buttons = screen.getAllByRole('button', { name: 'New team' });
    await user.click(nth(buttons, -1));

    expect(await screen.findByRole('dialog', { name: 'New team' })).toBeInTheDocument();
  });

  it('says when the teams could not be loaded, with Try again', async () => {
    let calls = 0;
    const { user } = renderTeams([
      reply('GET /api/v1/teams', () => {
        calls += 1;
        return calls === 1 ? { status: 500 } : { body: TEAMS };
      }),
    ]);

    await user.click(await screen.findByRole('button', { name: 'Try again' }));

    expect(await screen.findByText('2 of 3 teams')).toBeInTheDocument();
  });

  it('shows a team with its members and the plans using it (US2-3)', async () => {
    const { user, router } = renderTeams(
      [reply('GET /api/v1/teams/{teamId}', { body: EUROPE }, { teamId: 'T-1001' })],
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
      [
        reply(
          'GET /api/v1/teams/{teamId}',
          {
            status: 404,
            body: {
              error: {
                code: 'not_found',
                message: 'This team no longer exists — someone may have deleted it.',
                fields: [],
              },
            },
          },
          { teamId: 'T-1009' },
        ),
      ],
      '/settings/teams/T-1009',
    );

    const panel = await screen.findByRole('dialog', { name: 'Team' });
    expect(
      await within(panel).findByText('This team no longer exists — someone may have deleted it.'),
    ).toBeInTheDocument();
  });
});
