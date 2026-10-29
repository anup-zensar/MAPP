import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { CurrentUser } from '@/session/useSession';
import { HISTORY_ANSWERS } from '@/testing/fixtures/activity-log';
import { planForm } from '@/testing/fixtures/attribute-forms';
import { NOTHING_INSIDE } from '@/testing/fixtures/plan-page';
import { financialsNotServed, plan } from '@/testing/fixtures/plans';
import { testUser } from '@/testing/fixtures/users';
import {
  ALLOWED,
  refused,
  VIEWER,
  VIEWER_CONTENT_ACCESS,
  VIEWER_PLAN_ACCESS,
  viewOnlyNote,
} from '@/testing/fixtures/viewer';
import { expectNoAxeViolations } from '@/testing/axe';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { apiError, reply } from '@/testing/replies';
import type { Plan } from './api';
import { PlanCreatePage } from './PlanCreatePage';
import { PlanEditPage } from './PlanEditPage';
import { PlanPage } from './PlanPage';

const PLANNER_ACCESS: NonNullable<Plan['access']> = {
  edit: ALLOWED,
  editBudget: ALLOWED,
  delete: refused('Deleting plans', 'Planner'),
  publish: refused('Publishing plans', 'Editor'),
  copy: ALLOWED,
  content: { ...VIEWER_CONTENT_ACCESS, addCampaign: ALLOWED, editCampaigns: ALLOWED },
};

function openPlan(access: Plan['access'], person: CurrentUser) {
  mockApi([
    ...HISTORY_ANSWERS,
    ...NOTHING_INSIDE,
    ...financialsNotServed('P-1001'),
    reply('GET /api/v1/plans/{planId}', { body: plan({ access }) }),
    reply('GET /api/v1/shell-attribute-forms/{level}', { body: planForm() }),
  ]);
  return renderWithProviders(<PlanPage />, {
    user: person,
    route: '/plans/P-1001',
    path: '/plans/:planId',
  });
}

async function moreItems(user: ReturnType<typeof openPlan>['user']) {
  await user.click(await screen.findByRole('button', { name: 'More' }));
  return screen.getAllByRole('menuitem');
}

describe('014 FR-011: the plan page follows the reader’s access', () => {
  it('shows a Viewer one View only note and hides every change action (US2 scenario 1)', async () => {
    const { user } = openPlan(VIEWER_PLAN_ACCESS, VIEWER);

    await screen.findByRole('heading', { level: 1, name: 'FY2026 Lubricants Europe' });
    expect(screen.getByRole('note')).toHaveTextContent(viewOnlyNote('this plan'));
    expect(screen.queryByRole('link', { name: 'Edit' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Publish' })).not.toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'Add campaign' })).not.toBeInTheDocument();
    expect((await moreItems(user)).map((item) => item.textContent)).toEqual([
      'View history',
      'Export to Excel',
      'Export to PowerPoint',
      'Export to PDF',
    ]);
    await expectNoAxeViolations();
  });

  it('lists what a planner may not do here in More, disabled with the reason (US2 scenario 3)', async () => {
    const { user } = openPlan(PLANNER_ACCESS, testUser());

    await screen.findByRole('heading', { level: 1, name: 'FY2026 Lubricants Europe' });
    expect(screen.queryByRole('note')).not.toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Edit' })).toBeVisible();
    expect(screen.queryByRole('button', { name: 'Publish' })).not.toBeInTheDocument();
    await moreItems(user);
    expect(screen.getByRole('menuitem', { name: 'Publish' })).toHaveAccessibleDescription(
      "Publishing plans isn't part of your role (Editor). Ask an administrator if you need it.",
    );
    expect(screen.getByRole('menuitem', { name: 'Delete' })).toHaveAttribute(
      'aria-disabled',
      'true',
    );
  });
});

describe('014 FR-008, FR-011: plan forms the person may not use', () => {
  it('shows the no-access page with the reason at a plan’s edit address for a Viewer', async () => {
    mockApi([reply('GET /api/v1/plans/{planId}', { body: plan({ access: VIEWER_PLAN_ACCESS }) })]);
    renderWithProviders(<PlanEditPage />, {
      user: VIEWER,
      route: '/plans/P-1001/edit',
      path: '/plans/:planId/edit',
    });

    expect(await screen.findByRole('heading', { level: 1, name: 'No access' })).toBeVisible();
    expect(
      screen.getByText(
        "Editing plans isn't part of your role (Viewer). Ask an administrator if you need it.",
      ),
    ).toBeVisible();
  });

  it('shows the no-access page when the API refuses the new-plan form', async () => {
    const message =
      "Creating plans isn't part of your role (Viewer). Ask an administrator if you need it.";
    mockApi([
      reply('GET /api/v1/plan-defaults', { status: 403, body: apiError('forbidden', message) }),
      reply('GET /api/v1/me', { body: VIEWER }),
    ]);
    renderWithProviders(<PlanCreatePage />, {
      user: VIEWER,
      route: '/plans/new',
      path: '/plans/new',
    });

    expect(await screen.findByRole('heading', { level: 1, name: 'No access' })).toBeVisible();
    expect(screen.getByText(message)).toBeVisible();
  });
});
