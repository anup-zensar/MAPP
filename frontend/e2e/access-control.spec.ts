import { expect, test, type Page } from '@playwright/test';
import { action, callApi, expectNoAxeViolations, expectToast, itemAt, signInAs } from './fixtures';
import { lubricantsActivity } from './lubricants';
import { createPlan, idOf, type ListOf, type Named } from './planning';

// The access journeys of feature 014 (plan "Tests" › End-to-end, SC-002, SC-003, SC-007) against the real API
// and the demo data: who sees which plans, view versus edit, roles given in Settings, segregation of duties
// in approvals and global budgets. Records a journey needs are prepared through the API.

const ALEX = 'alex.morgan@shell.com';
const PRIYA = 'priya.raman@shell.com';
const DANIEL = 'daniel.okoro@shell.com';
const MEI = 'mei.lin@shell.com';
const CARLOS = 'carlos.mendes@shell.com';
const SARA = 'sara.novak@shell.com';

const LUBRICANTS = 'FY2026 Lubricants Europe';
const MOBILITY = 'FY2026 Mobility China';
const GLOBAL_BRAND = 'FY2027 Global Brand';
const LUBRICANTS_WORKFLOW = 'Lubricants EU approval';

function planCard(page: Page, name: string) {
  return page.getByRole('article', { name, exact: true });
}

async function openPlans(page: Page) {
  await page.goto('/plans?scope=all');
  await expect(page.getByRole('heading', { name: 'Plans', level: 1 })).toBeVisible();
}

test('US1: each person sees the plans of their markets, and another market’s address looks missing', async ({
  page,
}) => {
  const visible: [string, string[], string[]][] = [
    [PRIYA, [LUBRICANTS], [MOBILITY, GLOBAL_BRAND]],
    [MEI, [MOBILITY], [LUBRICANTS, GLOBAL_BRAND]],
    [SARA, [LUBRICANTS, MOBILITY], [GLOBAL_BRAND]],
    [CARLOS, [LUBRICANTS, MOBILITY, GLOBAL_BRAND], []],
  ];
  for (const [email, shown, hidden] of visible) {
    await signInAs(page, email);
    await openPlans(page);
    for (const name of shown) {
      await expect(planCard(page, name)).toBeVisible();
    }
    for (const name of hidden) {
      await expect(planCard(page, name)).toHaveCount(0);
    }
  }
  await expectNoAxeViolations(page);

  await signInAs(page, PRIYA);
  const lubricantsId = await idOf(page, '/api/v1/plans', LUBRICANTS);
  await signInAs(page, MEI);
  await page.goto(`/plans/${lubricantsId}`);
  await expect(page.getByText('Plan not found')).toBeVisible();
  await expect(page.getByText(LUBRICANTS)).toHaveCount(0);
  await expectNoAxeViolations(page);
});

test('US2: a Viewer reads a plan and an activity without change actions; a planner sees why Delete is locked', async ({
  page,
}) => {
  await signInAs(page, SARA);
  const planId = await idOf(page, '/api/v1/plans', LUBRICANTS);
  const activities = await callApi<ListOf<Named>>(
    page,
    'GET',
    `/api/v1/plans/${planId}/activities`,
  );
  const activity = itemAt(activities.items, 0);

  await page.goto(`/plans/${planId}/activities/${activity.id}`);
  await expect(page.getByRole('heading', { name: activity.name, level: 1 })).toBeVisible();
  await expect(page.getByRole('note').filter({ hasText: /^View only/ })).toContainText(
    'your role (Viewer) lets you see this record but not change it.',
  );
  await expect(action(page, 'Edit')).toHaveCount(0);
  await expect(page.getByRole('button', { name: /Submit for approval/ })).toHaveCount(0);
  await page.getByRole('button', { name: `Actions for ${activity.name}` }).click();
  await expect(page.getByRole('menuitem')).toHaveText(['View history']);
  await page.keyboard.press('Escape');
  await expectNoAxeViolations(page);

  await signInAs(page, PRIYA);
  await openPlans(page);
  await planCard(page, LUBRICANTS)
    .getByRole('button', { name: `Actions for ${LUBRICANTS}` })
    .click();
  await expect(page.getByRole('menuitem', { name: 'Delete' })).toHaveAccessibleDescription(
    "Deleting plans isn't part of your role (Planner). Ask an administrator if you need it.",
  );
  await page.keyboard.press('Escape');

  // Priya plans for Lubricants Europe only: the Teams field offers no other market.
  await page.goto('/plans/new');
  await page.getByRole('combobox', { name: /^Teams/ }).click();
  await expect(page.getByRole('listbox').getByRole('option')).toHaveText(['Lubricants Europe']);
  await page.keyboard.press('Escape');
  await expectNoAxeViolations(page);
});

test.describe('US3: roles given in Settings', () => {
  // The journey changes Sara's roles; they go back to Viewer for her own teams whatever happens.
  test.afterEach(async ({ page }) => {
    await signInAs(page, ALEX);
    const sara = await idOf(page, '/api/v1/people', 'Sara Novak');
    const roles = await callApi<{ version: number }>(page, 'GET', `/api/v1/people/${sara}/roles`);
    const response = await page.request.put(`/api/v1/people/${sara}/roles`, {
      data: {
        version: roles.version,
        roles: [{ roleId: 'RL-1005', scope: { type: 'myTeams' } }],
      },
    });
    expect(response.ok()).toBe(true);
  });

  test('Alex makes Sara a Planner of Mobility China, and she may edit its activities at once', async ({
    page,
    browser,
  }) => {
    // Sara is already signed in elsewhere when her roles change.
    const saraContext = await browser.newContext();
    const saraPage = await saraContext.newPage();
    await signInAs(saraPage, SARA);
    const mobilityId = await idOf(saraPage, '/api/v1/plans', MOBILITY);
    const activities = await callApi<ListOf<Named>>(
      saraPage,
      'GET',
      `/api/v1/plans/${mobilityId}/activities`,
    );
    const activity = itemAt(activities.items, 0);

    await signInAs(page, ALEX);
    await page.goto('/settings/access');
    await page.getByRole('button', { name: 'Edit roles of Sara Novak' }).click();
    const panel = page.getByRole('dialog', { name: 'Roles of Sara Novak' });
    await panel.getByRole('checkbox', { name: 'Planner', exact: true }).check();
    const where = panel.getByRole('group', { name: 'Where Planner applies' });
    await where.getByLabel('Chosen teams').check();
    await where.getByRole('combobox', { name: /^Teams/ }).click();
    await page.getByRole('option', { name: 'Mobility China' }).click();
    await expectNoAxeViolations(page);
    await panel.getByRole('button', { name: 'Save', exact: true }).click();
    await expectToast(page, 'Roles of Sara Novak saved');

    await saraPage.goto(`/plans/${mobilityId}/activities/${activity.id}`);
    await expect(saraPage.getByRole('heading', { name: activity.name, level: 1 })).toBeVisible();
    await expect(action(saraPage, 'Edit')).toBeVisible();
    await expect(saraPage.getByRole('note').filter({ hasText: /^View only/ })).toHaveCount(0);
    await saraContext.close();
  });

  test('the only administrator cannot take the Administrator role away from himself', async ({
    page,
  }) => {
    await signInAs(page, ALEX);
    await page.goto('/settings/access');
    await page.getByRole('button', { name: 'Edit roles of Alex Morgan' }).click();
    const panel = page.getByRole('dialog', { name: 'Roles of Alex Morgan' });
    await panel.getByRole('checkbox', { name: 'Administrator', exact: true }).uncheck();
    await panel.getByRole('button', { name: 'Save', exact: true }).click();
    await page.getByRole('button', { name: 'Save changes' }).click();
    await expect(
      page
        .getByText(
          'Alex Morgan is the only administrator. Give someone else the Administrator role first.',
        )
        .first(),
    ).toBeVisible();
  });
});

test('US5: Daniel may not approve an activity he owns; Alex, the backup approver, approves it', async ({
  page,
}) => {
  test.slow();
  await signInAs(page, PRIYA);
  const prepared = await lubricantsActivity(page, 'Own approval', '20000.00');
  const daniel = await idOf(page, '/api/v1/people', 'Daniel Okoro');
  const workflows = await callApi<ListOf<Named>>(page, 'GET', '/api/v1/approval-workflows');
  const workflow = workflows.items.find((w) => w.name === LUBRICANTS_WORKFLOW);
  if (!workflow) {
    throw new Error(`There is no workflow '${LUBRICANTS_WORKFLOW}'`);
  }
  const current = await callApi<ActivityAnswer>(page, 'GET', `/api/v1/activities/${prepared.id}`);
  const saved = await ownedByDaniel(page, current, daniel, workflow.id);
  await callApi(page, 'POST', `/api/v1/activities/${prepared.id}/approval/submit`, {
    version: saved.version,
    comment: '',
  });

  await signInAs(page, DANIEL);
  await page.goto(prepared.path);
  await expect(page.getByRole('heading', { name: prepared.name, level: 1 })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Approve', exact: true })).toHaveCount(0);
  await expectNoAxeViolations(page);

  await signInAs(page, ALEX);
  await page.goto(prepared.path);
  await page.getByRole('button', { name: 'Approve', exact: true }).first().click();
  const dialog = page.getByRole('alertdialog', { name: `Approve '${prepared.name}'?` });
  await dialog.getByRole('button', { name: 'Approve', exact: true }).click();
  await expect(page.getByText('Approved').first()).toBeVisible();
});

interface ActivityAnswer {
  id: string;
  name: string;
  externalName: string;
  activityType: { id: string };
  startDate: string;
  endDate: string;
  ongoing: boolean;
  plannedCost: unknown;
  description: string;
  attributes: unknown;
  version: number;
}

/** Saves the activity with Daniel Okoro as its owner and the Lubricants EU workflow, as the edit form would. */
async function ownedByDaniel(
  page: Page,
  activity: ActivityAnswer,
  ownerId: string,
  workflowId: string,
): Promise<{ version: number }> {
  return callApi(page, 'PATCH', `/api/v1/activities/${activity.id}`, {
    name: activity.name,
    externalName: activity.externalName,
    activityTypeId: activity.activityType.id,
    ownerId,
    startDate: activity.startDate,
    endDate: activity.endDate,
    ongoing: activity.ongoing,
    plannedCost: activity.plannedCost,
    approvalWorkflowId: workflowId,
    description: activity.description,
    attributes: activity.attributes,
    goals: { inq: null, setLaterStagesMyself: false, mql: null, revenue: null },
    version: activity.version,
    restartApproval: false,
  });
}

test('US6: Carlos changes an allocation; the plan owner sees a locked budget; Mei has no Global budgets', async ({
  page,
}) => {
  await signInAs(page, CARLOS);
  const budgets = await callApi<ListOf<Named & { year: number }>>(
    page,
    'GET',
    '/api/v1/global-budgets',
  );
  const budget = itemAt(budgets.items, 0);
  await signInAs(page, MEI);
  const plan = await createPlan(page, 'Access allocation', budget.year);
  await signInAs(page, CARLOS);
  const options = await callApi<{ items: { plan: { id: string; version: number } }[] }>(
    page,
    'GET',
    `/api/v1/global-budgets/${budget.id}/allocation-options`,
  );
  const option = options.items.find((candidate) => candidate.plan.id === plan.id);
  if (!option) {
    throw new Error(`'${plan.name}' is not offered for an allocation`);
  }
  await callApi(page, 'POST', `/api/v1/global-budgets/${budget.id}/allocations`, {
    planId: plan.id,
    amount: { amount: '10000.00', currency: 'USD' },
    planVersion: option.plan.version,
  });

  await page.goto(`/budgets/${budget.id}`);
  await page.getByRole('button', { name: `Actions for ${plan.name}` }).click();
  await page.getByRole('menuitem', { name: 'Change allocation' }).click();
  const dialog = page.getByRole('dialog', { name: `Change the allocation to '${plan.name}'` });
  await dialog.getByRole('textbox', { name: /^Amount/ }).fill('12000.00');
  await expectNoAxeViolations(page);
  await dialog.getByRole('button', { name: 'Save allocation' }).click();
  await expectToast(page, `Allocation to '${plan.name}' saved`);

  await signInAs(page, MEI);
  await page.goto(`/plans/${plan.id}/edit`);
  await expect(
    page.getByText(`Allocated from '${budget.name}' — ask Carlos Mendes to change it.`).first(),
  ).toBeVisible();
  await expectNoAxeViolations(page);
  await expect(
    page.getByRole('navigation', { name: 'Main' }).getByRole('link', { name: 'Global budgets' }),
  ).toHaveCount(0);
  await page.goto('/budgets');
  await expect(page.getByRole('heading', { name: 'No access', level: 1 })).toBeVisible();
});
