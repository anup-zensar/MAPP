import { screen, within } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import type { CurrentUser } from '@/session/useSession';
import { seedList } from '@/testing/fixtures/plans';
import { testUser } from '@/testing/fixtures/users';
import { ALLOWED, refused, VIEWER, VIEWER_PLAN_ACCESS } from '@/testing/fixtures/viewer';
import { expectNoAxeViolations } from '@/testing/axe';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { reply } from '@/testing/replies';
import type { Plan } from './api';
import { PlansPage } from './PlansPage';

afterEach(() => window.localStorage.clear());

const NO_TEAM =
  'Plans belong to teams, and you are not in a team yet. Ask an administrator to add you to one.';

/** A Planner of their own teams: may create plans there (`teams`), and may not delete plans. */
function planner(teams: { id: string; name: string }[]): CurrentUser {
  const where = teams.map((team) => ({ ...team, colour: 'blue' as const, active: true }));
  return testUser({
    access: {
      roles: [],
      permissions: [
        { key: 'plans.view', all: false, teams: where },
        { key: 'plans.create', all: false, teams: where },
      ],
      administrators: [],
      noSelfApproval: true,
    },
  });
}

function openList(access: Plan['access'], person: CurrentUser) {
  const list = seedList();
  mockApi([
    reply('GET /api/v1/plans', {
      body: { ...list, items: list.items.map((item) => ({ ...item, access })) },
    }),
  ]);
  return renderWithProviders(<PlansPage />, { user: person, route: '/plans', path: '/plans' });
}

async function cardMenu(user: ReturnType<typeof openList>['user']) {
  const card = await screen.findByRole('article', { name: 'FY2026 Lubricants Europe' });
  await user.click(within(card).getByRole('button', { name: /^Actions for/ }));
  return screen.getAllByRole('menuitem');
}

describe('014 FR-011, FR-013: the Plans list follows the reader’s access', () => {
  it('offers a Viewer no New plan and only View on each plan', async () => {
    const { user } = openList(VIEWER_PLAN_ACCESS, VIEWER);

    expect(screen.queryByRole('link', { name: 'New plan' })).not.toBeInTheDocument();
    expect((await cardMenu(user)).map((item) => item.textContent)).toEqual(['View']);
    await expectNoAxeViolations();
  });

  it('shows a planner Delete disabled with its reason (US2 scenario 3)', async () => {
    const access = {
      ...VIEWER_PLAN_ACCESS,
      edit: ALLOWED,
      delete: refused('Deleting plans', 'Planner'),
    };
    const { user } = openList(access, planner([{ id: 'T-1001', name: 'Lubricants Europe' }]));

    expect(screen.getByRole('link', { name: 'New plan' })).toBeVisible();
    await cardMenu(user);
    expect(screen.getByRole('menuitem', { name: 'Delete' })).toHaveAccessibleDescription(
      "Deleting plans isn't part of your role (Planner). Ask an administrator if you need it.",
    );
  });

  it('locks New plan and says why for a planner who is in no team (message 22)', () => {
    openList(VIEWER_PLAN_ACCESS, planner([]));

    const button = screen.getByRole('button', { name: 'New plan' });
    expect(button).toBeDisabled();
    expect(button).toHaveAccessibleDescription(NO_TEAM);
  });
});
