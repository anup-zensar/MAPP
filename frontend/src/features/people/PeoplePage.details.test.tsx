import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { nextOf } from '@/testing/dom';
import { ALEX_DETAILS, EUROPE, PRIYA, renderPeople } from '@/testing/fixtures/people';
import { reply } from '@/testing/replies';

/** The value next to a label of the details' fact list. */
function factOf(panel: HTMLElement, label: string) {
  return nextOf(within(panel).getByText(label, { selector: 'dt' }));
}

describe("PeoplePage (feature 003, US1): a person's details", () => {
  it('shows a person with their teams, what they own and approve, and the record line (US1-3)', async () => {
    const { user, router } = renderPeople(
      [
        reply('GET /api/v1/people/{personId}', { body: ALEX_DETAILS }, { personId: 'U-1001' }),
        reply(
          'GET /api/v1/people/{personId}/roles',
          {
            body: {
              person: {
                id: 'U-1001',
                name: 'Alex Morgan',
                initials: 'AM',
                jobTitle: 'Head of marketing',
                active: true,
              },
              teams: [EUROPE],
              roles: [
                {
                  id: 'RA-1001',
                  role: { id: 'RL-1003', name: 'Planner', builtIn: 'planner' },
                  scope: { type: 'myTeams', teams: [EUROPE], text: 'My teams' },
                  scopeEmpty: false,
                },
              ],
              permissions: [],
              version: 1,
            },
          },
          { personId: 'U-1001' },
        ),
      ],
      '/settings/people/U-1001',
    );

    const panel = await screen.findByRole('dialog', { name: 'Alex Morgan' });
    expect(await within(panel).findByText('alex.morgan@shell.com')).toBeInTheDocument();
    expect(within(panel).getByText('Head of marketing')).toBeInTheDocument();
    expect(within(panel).getByRole('link', { name: 'Lubricants Europe' })).toHaveAttribute(
      'href',
      '/settings/teams/T-1001',
    );
    // Two plain labels (functional review S-02): what they own, and where they approve.
    expect(factOf(panel, 'Owns')).toHaveTextContent('2 plans and 5 activities');
    expect(factOf(panel, 'Approves in')).toHaveTextContent('1 approval workflow');
    expect(await within(panel).findByText('Planner · My teams')).toBeInTheDocument();
    expect(within(panel).getByRole('link', { name: 'Change roles' })).toHaveAttribute(
      'href',
      '/settings/access?tab=people&person=U-1001',
    );
    expect(
      within(panel).getByText(/^Created by Alex Morgan on .* · Last changed by Priya Raman on /),
    ).toBeInTheDocument();
    // The one panel footer: Edit, then ⋯ with Deactivate and its reason.
    expect(within(panel).getByRole('button', { name: 'Edit' })).toBeInTheDocument();
    await user.click(within(panel).getByRole('button', { name: 'Actions for Alex Morgan' }));
    expect(
      within(screen.getByRole('menu')).getByRole('menuitem', { name: 'Deactivate' }),
    ).toHaveAccessibleDescription('People are never deleted, so the history keeps their name.');
    await user.keyboard('{Escape}');

    await user.click(within(panel).getByRole('button', { name: 'Close' }));
    await waitFor(() => expect(router.state.location.pathname).toBe('/settings/people'));
  });

  it('says in the panel when the person does not exist', async () => {
    renderPeople(
      [
        reply(
          'GET /api/v1/people/{personId}',
          {
            status: 404,
            body: {
              error: { code: 'not_found', message: "This person doesn't exist.", fields: [] },
            },
          },
          { personId: 'U-1999' },
        ),
      ],
      '/settings/people/U-1999',
    );

    const panel = await screen.findByRole('dialog', { name: 'Person' });
    expect(await within(panel).findByText("This person doesn't exist.")).toBeInTheDocument();
  });

  it('keeps Deactivate in the details for yourself, saying why it does nothing (FR-006)', async () => {
    const { api, user } = renderPeople(
      [
        reply(
          'GET /api/v1/people/{personId}',
          { body: { ...PRIYA, usage: [] } },
          { personId: 'U-1002' },
        ),
      ],
      '/settings/people/U-1002',
    );

    const panel = await screen.findByRole('dialog', { name: 'Priya Raman' });
    expect(await within(panel).findByText('Nothing yet')).toBeInTheDocument();
    expect(factOf(panel, 'Approves in')).toHaveTextContent('No approval workflows');
    await user.click(within(panel).getByRole('button', { name: 'Actions for Priya Raman' }));
    const deactivate = within(screen.getByRole('menu')).getByRole('menuitem', {
      name: 'Deactivate',
    });
    expect(deactivate).toHaveAccessibleDescription(
      "You can't deactivate yourself — ask someone else to do it.",
    );
    await user.click(deactivate);

    expect(
      await screen.findByText("You can't deactivate yourself — ask someone else to do it."),
    ).toBeInTheDocument();
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
    expect(api.requests.some((request) => request.method === 'POST')).toBe(false);
  });
});
