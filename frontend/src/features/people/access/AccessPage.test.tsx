import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { nth } from '@/testing/dom';
import { ALEX, SARA_ROLES } from '@/testing/fixtures/access';
import { renderAccess, rowOf } from '@/testing/fixtures/access-page';
import { reply } from '@/testing/replies';

describe('AccessPage: People tab (feature 014, US3)', () => {
  it('lists everyone with teams and roles, with a search and a role filter with No role (US3-1)', async () => {
    const { user } = renderAccess([
      reply('GET /api/v1/access/changes', {
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
      }),
    ]);

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
    renderAccess([
      reply('GET /api/v1/access/people', {
        body: {
          items: [
            {
              ...SARA_ROLES,
              roles: [{ ...nth(SARA_ROLES.roles, 0), scopeEmpty: true }],
            },
          ],
          total: 1,
        },
      }),
    ]);

    expect(await screen.findByText('No teams left in this scope')).toBeInTheDocument();
  });
});
