import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { ALEX, FINANCE_REVIEWER } from '@/testing/fixtures/access';
import { renderAccess } from '@/testing/fixtures/access-page';
import { findMessages } from '@/testing/render';
import { reply } from '@/testing/replies';

describe('AccessPage: Roles tab (feature 014, US4)', () => {
  it('lists the roles with how many people hold them, Built in, and New role', async () => {
    renderAccess([], '/settings/access?tab=roles');

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
    const { api, user } = renderAccess([], '/settings/access?tab=roles');

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
      [
        reply(
          'DELETE /api/v1/roles/{roleId}',
          {
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
          { roleId: 'RL-1006' },
        ),
        reply(
          'POST /api/v1/recycle-bin/{entryId}/undo',
          {
            body: {
              recordType: 'role',
              recordTypeLabel: 'role',
              recordId: FINANCE_REVIEWER.id,
              recordName: FINANCE_REVIEWER.name,
              details: '',
            },
          },
          { entryId: 'RB-1101' },
        ),
      ],
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
    const { user } = renderAccess([], '/settings/access?tab=roles');

    await user.click(await screen.findByRole('button', { name: 'Actions for Administrator' }));

    expect(
      within(screen.getByRole('menu'))
        .getAllByRole('menuitem')
        .map((item) => item.firstChild?.textContent),
    ).toEqual(['View', 'Copy as new role']);
  });
});
