import { expect, test, type Page } from '@playwright/test';
import { action, callApi, expectToast, itemAt, signInAs, uniqueName } from './fixtures';
import { createPlan, createPlanningRecords, idOf, type ListOf, type Named } from './planning';

// End-to-end journeys through the client's BRD MVP gaps (features 014–017), against the real API with
// the demo data: roles and access (014), financial transactions (015), planning and currency (016:
// split activity, global budgets) and approval rules (017: approved automatically, delegation).
// Records a journey needs are prepared through the API; what it proves is done on screen.

const ALEX = 'alex.morgan@shell.com';
const PRIYA = 'priya.raman@shell.com';
const DANIEL = 'daniel.okoro@shell.com';
const MEI = 'mei.lin@shell.com';
const CARLOS = 'carlos.mendes@shell.com';
const SARA = 'sara.novak@shell.com';

const STANDARD_WORKFLOW = 'Standard marketing approval';
/** The demo plan, activity and purchase order of the GSAP story (backend/seed). */
const LUBRICANTS_PLAN = 'FY2026 Lubricants Europe';
const SOCIAL_ACTIVITY = 'A-1002';
const SOCIAL_PO = 'PO 4500001002 / 10';

interface FormAttribute {
  id: string;
  label: string;
  options: { id: string; label: string }[];
}

/**
 * Every needed-to-submit activity attribute of a Lubricants plan, filled with a demo option: the
 * shared ones and the Lubricants group (Brands, Global key accounts, Sector, Sub-sector, Product launch).
 */
async function lubricantsAttributes(page: Page): Promise<Record<string, unknown>> {
  const form = await callApi<{ sections: { attributes: FormAttribute[] }[] }>(
    page,
    'GET',
    '/api/v1/shell-attribute-forms/activity',
  );
  const all = form.sections.flatMap((section) => section.attributes);
  const attribute = (label: string) => {
    const found = all.find((a) => a.label === label);
    if (!found) {
      throw new Error(`There is no activity attribute '${label}'`);
    }
    return found;
  };
  const option = (label: string, optionLabel?: string) => {
    const options = attribute(label).options;
    const found = optionLabel ? options.find((o) => o.label === optionLabel) : options[0];
    if (!found) {
      throw new Error(`'${label}' has no option '${optionLabel}'`);
    }
    return found.id;
  };
  return {
    [attribute('Channel / audience').id]: [option('Channel / audience', 'External')],
    [attribute('Country / OU').id]: [
      { optionId: option('Country / OU', 'China'), share: '100.00' },
    ],
    [attribute('Company code').id]: option('Company code', 'CN04'),
    [attribute('Cost centre').id]: option('Cost centre', 'CN04000220'),
    [attribute('Global campaign').id]: [option('Global campaign', 'Shell Recharge launch')],
    [attribute('Growth ambition').id]: [option('Growth ambition')],
    [attribute('PIR').id]: option('PIR', 'No'),
    [attribute('FIT4 initiative').id]: option('FIT4 initiative', 'Not FIT4'),
    [attribute('FIT4 ID').id]: 'FIT4-0999',
    [attribute('Brands').id]: [{ optionId: option('Brands', 'Helix'), share: '100.00' }],
    [attribute('Global key accounts').id]: [option('Global key accounts')],
    [attribute('Sector').id]: [option('Sector', 'Automotive')],
    [attribute('Sub-sector').id]: [option('Sub-sector', 'Passenger cars')],
    [attribute('Product launch').id]: option('Product launch', 'No'),
  };
}

interface LubricantsActivity extends Named {
  version: number;
  path: string;
}

/**
 * As Priya Raman: a Lubricants plan of next year (team Lubricants Europe) with one campaign and one
 * initiative, and a Created activity of `amount` USD with every needed field filled and the workflow
 * "Standard marketing approval" (017 seed: level 1 Priya's line manager Daniel Okoro, approved
 * automatically below $10,000.00; levels 2 and 3 not needed for this activity).
 */
async function lubricantsActivity(
  page: Page,
  label: string,
  amount: string,
): Promise<LubricantsActivity> {
  const year = new Date().getUTCFullYear() + 1;
  const [modelId, teamId, ownerId, planForm] = await Promise.all([
    idOf(page, '/api/v1/models', 'Lubricants 1.0'),
    idOf(page, '/api/v1/teams', 'Lubricants Europe'),
    idOf(page, '/api/v1/people', 'Priya Raman'),
    callApi<{
      businessUnitAttributeId: string;
      businessUnitOptions: { id: string; label: string }[];
    }>(page, 'GET', '/api/v1/shell-attribute-forms/plan'),
  ]);
  const lubricants = planForm.businessUnitOptions.find((o) => o.label === 'Lubricants');
  if (!lubricants) {
    throw new Error('The plan form has no business unit Lubricants');
  }
  const plan = await callApi<Named>(page, 'POST', '/api/v1/plans', {
    name: uniqueName(`${label} plan`),
    year,
    modelId,
    teamIds: [teamId],
    ownerId,
    aop: false,
    description: '',
    budget: { amount: '100000.00', currency: 'USD' },
    defaultFinanceBudgetId: null,
    defaultSpread: 'evenly',
    bcpir: false,
    goalStage: 'inq',
    goalAmount: null,
    attributes: { [planForm.businessUnitAttributeId]: [lubricants.id] },
  });
  const wholeYear = { ownerId, startDate: `${year}-01-01`, endDate: `${year}-12-31` };
  const campaign = await callApi<Named>(page, 'POST', `/api/v1/plans/${plan.id}/campaigns`, {
    name: uniqueName(`${label} campaign`),
    description: '',
    ...wholeYear,
    budget: { amount: '50000.00', currency: 'USD' },
    attributes: {},
  });
  const initiative = await callApi<Named>(
    page,
    'POST',
    `/api/v1/campaigns/${campaign.id}/initiatives`,
    {
      name: uniqueName(`${label} initiative`),
      description: '',
      ...wholeYear,
      budget: { amount: '40000.00', currency: 'USD' },
      attributes: {},
    },
  );
  const form = await callApi<{ activityTypes: Named[]; approvalWorkflows: Named[] }>(
    page,
    'GET',
    `/api/v1/initiatives/${initiative.id}/activity-form`,
  );
  const type = form.activityTypes.find((t) => t.name === 'Content production');
  const workflow = form.approvalWorkflows.find((w) => w.name === STANDARD_WORKFLOW);
  if (!type || !workflow) {
    throw new Error(`The activity form offers no 'Content production' or '${STANDARD_WORKFLOW}'`);
  }
  const activity = await callApi<Named & { version: number }>(
    page,
    'POST',
    `/api/v1/initiatives/${initiative.id}/activities`,
    {
      name: uniqueName(`${label} activity`),
      externalName: '',
      activityTypeId: type.id,
      ownerId,
      startDate: `${year}-03-01`,
      endDate: `${year}-05-31`,
      ongoing: false,
      plannedCost: { amount, currency: 'USD' },
      approvalWorkflowId: workflow.id,
      description: '',
      attributes: await lubricantsAttributes(page),
      goals: { inq: null, setLaterStagesMyself: false, mql: null, revenue: null },
    },
  );
  return { ...activity, path: `/plans/${plan.id}/activities/${activity.id}` };
}

// ---------------------------------------------------------------------------------------------------

test('014: an administrator gives a person a role in Settings › Roles and access, and it takes effect', async ({
  page,
}) => {
  test.slow();
  await signInAs(page, ALEX);
  const name = uniqueName('Access Person');
  const email = `access.${Date.now().toString(36)}@shell.com`;
  const teamId = await idOf(page, '/api/v1/teams', 'Lubricants Europe');
  await callApi(page, 'POST', '/api/v1/people', {
    name,
    email,
    jobTitle: 'Marketer',
    teamIds: [teamId],
  });

  // Without a role the new person can't create plans.
  await signInAs(page, email);
  await page.goto('/plans');
  await expect(page.getByRole('heading', { name: 'Plans', level: 1 })).toBeVisible();
  await expect(action(page, 'New plan')).toHaveCount(0);

  // Alex gives them Planner for their own teams.
  await signInAs(page, ALEX);
  await page.goto('/settings/access');
  await expect(page.getByRole('heading', { name: 'Roles and access', level: 1 })).toBeVisible();
  await page.getByRole('searchbox', { name: 'Search name or job title' }).fill(name);
  await page.getByRole('button', { name: `Edit roles of ${name}` }).click();
  const panel = page.getByRole('dialog', { name: `Roles of ${name}` });
  await panel.getByRole('checkbox', { name: 'Planner', exact: true }).check();
  await expect(
    panel.getByRole('group', { name: 'Where Planner applies' }).getByLabel('My teams'),
  ).toBeChecked();
  await panel.getByRole('button', { name: 'Save', exact: true }).click();
  await expectToast(page, `Roles of ${name} saved`);
  await expect(page.getByRole('list', { name: `Roles of ${name}` })).toContainText('Planner');

  // Signed in again, the person may now create plans.
  await signInAs(page, email);
  await page.goto('/plans');
  await expect(action(page, 'New plan')).toBeVisible();
});

test('014: a viewer sees a plan read-only', async ({ page }) => {
  await signInAs(page, SARA);
  const planId = await idOf(page, '/api/v1/plans', LUBRICANTS_PLAN);

  await page.goto(`/plans/${planId}`);
  await expect(page.getByRole('heading', { name: LUBRICANTS_PLAN, level: 1 })).toBeVisible();
  await expect(page.getByRole('note').filter({ hasText: /^View only/ })).toContainText(
    'your role (Viewer) lets you see this plan but not change it.',
  );
  // No Edit, Publish or Add campaign: the actions follow the plan's access (014 FR-011).
  await expect(action(page, 'Edit')).toHaveCount(0);
  await expect(action(page, 'Publish')).toHaveCount(0);
  await expect(action(page, 'Add campaign')).toHaveCount(0);
});

test('015: a planner sees an activity’s GSAP financials and the Financials page', async ({
  page,
}) => {
  await signInAs(page, PRIYA);
  const planId = await idOf(page, '/api/v1/plans', LUBRICANTS_PLAN);

  await page.goto(`/plans/${planId}/activities/${SOCIAL_ACTIVITY}`);
  const financials = page.getByRole('region', { name: 'Financials', exact: true });
  await expect(financials).toBeVisible();
  await expect(financials.getByText('Committed (POs)').first()).toBeVisible();
  await expect(financials.getByText(SOCIAL_PO).first()).toBeVisible();

  await page
    .getByRole('navigation', { name: 'Main' })
    .getByRole('link', { name: 'Financials' })
    .click();
  await expect(page).toHaveTitle('Financials · MAPP');
  await expect(page.getByRole('heading', { name: 'Financials', level: 1 })).toBeVisible();
  await expect(page.getByRole('main').getByText(SOCIAL_PO).first()).toBeVisible();
});

test('017: a small activity is approved automatically on submission', async ({ page }) => {
  await signInAs(page, PRIYA);
  const activity = await lubricantsActivity(page, 'Auto', '8000.00');

  await page.goto(activity.path);
  await expect(page.getByRole('heading', { name: activity.name, level: 1 })).toBeVisible();
  await page.getByRole('button', { name: 'Submit for approval' }).first().click();
  const submit = page.getByRole('alertdialog', { name: `Submit '${activity.name}' for approval?` });
  await expect(submit.getByText(`Approval workflow: ${STANDARD_WORKFLOW}`)).toBeVisible();
  await expect(submit.getByText('This activity will be approved automatically.')).toBeVisible();
  await submit.getByRole('button', { name: 'Submit for approval' }).click();
  await expectToast(page, `Activity '${activity.name}' approved automatically`);
  await expect(page.getByText('Approved', { exact: true }).first()).toBeVisible();
});

test('017: a delegate approves for the approver who is away', async ({ page }) => {
  test.slow();
  // Daniel Okoro (Priya's line manager) delegates his approvals to Carlos Mendes for today.
  await signInAs(page, DANIEL);
  const carlosId = await idOf(page, '/api/v1/people', 'Carlos Mendes');
  const today = new Date().toISOString().slice(0, 10);
  const delegation = await callApi<{ id: string; version: number }>(
    page,
    'POST',
    '/api/v1/approval-delegations',
    { delegateId: carlosId, startDate: today, endDate: today, note: 'End-to-end test' },
  );

  try {
    // Priya submits a $30,000.00 activity: it waits for Daniel at level 1.
    await signInAs(page, PRIYA);
    const activity = await lubricantsActivity(page, 'Delegate', '30000.00');
    await callApi(page, 'POST', `/api/v1/activities/${activity.id}/approval/submit`, {
      version: activity.version,
      comment: '',
    });

    // Carlos finds it on his Approvals page and approves it as Daniel's delegate.
    await signInAs(page, CARLOS);
    await page.goto('/approvals');
    await expect(page.getByRole('heading', { name: 'Approvals', level: 1 })).toBeVisible();
    await page.getByRole('searchbox', { name: 'Search activity name or ID' }).fill(activity.name);
    await page.getByRole('button', { name: activity.name, exact: true }).click();
    const preview = page.getByRole('dialog', { name: activity.name });
    await preview.getByRole('button', { name: 'Approve', exact: true }).click();
    const approve = page.getByRole('alertdialog', { name: `Approve '${activity.name}'?` });
    await expect(approve.getByText('You approve as delegate of Daniel Okoro.')).toBeVisible();
    await approve.getByRole('button', { name: 'Approve', exact: true }).click();
    await expectToast(page, `Activity '${activity.name}' approved`);

    // Priya sees who decided, and for whom.
    await signInAs(page, PRIYA);
    await page.goto(activity.path);
    await expect(page.getByText('Approved', { exact: true }).first()).toBeVisible();
    await page.getByRole('tab', { name: 'Approval' }).click();
    await expect(
      page
        .getByRole('main')
        .getByText(/Carlos Mendes as delegate of Daniel Okoro/)
        .first(),
    ).toBeVisible();
  } finally {
    await signInAs(page, DANIEL);
    await callApi(page, 'POST', `/api/v1/approval-delegations/${delegation.id}/cancel`, {
      version: delegation.version,
    });
  }
});

test('016: a planner splits an activity in two', async ({ page }) => {
  await signInAs(page, MEI);
  const records = await createPlanningRecords(page, 'Split', {
    withWorkflow: false,
    leaveEmpty: [],
  });
  const name = records.activityName;
  const newName = uniqueName('Split part');

  await page.goto(records.activityPath);
  await expect(page.getByRole('heading', { name, level: 1 })).toBeVisible();
  await page
    .getByRole('button', { name: `Actions for ${name}` })
    .first()
    .click();
  await page.getByRole('menuitem', { name: 'Split activity' }).click();
  const dialog = page.getByRole('dialog', { name: `Split '${name}'` });
  await dialog.getByRole('textbox', { name: /^New activity name/ }).fill(newName);
  await dialog.getByRole('textbox', { name: /^Planned cost to move/ }).fill('5000.00');
  await expect(dialog.getByRole('region', { name: 'After the split' })).toContainText(
    '15,000.00 USD',
  );
  await dialog.getByRole('button', { name: 'Split activity' }).click();
  await expectToast(page, `'${name}' split — '${newName}' created.`);

  // Both activities are in the plan; the planned cost is shared between them.
  const activities = await callApi<ListOf<Named & { plannedCost: { amount: string } }>>(
    page,
    'GET',
    `/api/v1/plans/${records.planId}/activities`,
  );
  const costs = Object.fromEntries(
    activities.items.map((item) => [item.name, item.plannedCost.amount]),
  );
  expect(costs).toEqual({ [name]: '15000.00', [newName]: '5000.00' });
});

test('016: a global planner allocates part of the global budget to a plan', async ({ page }) => {
  // The demo global budget (Carlos Mendes owns it) and a new plan of Mei's of the same year.
  await signInAs(page, CARLOS);
  const list = await callApi<ListOf<Named & { year: number }>>(
    page,
    'GET',
    '/api/v1/global-budgets',
  );
  const budget = itemAt(list.items, 0);
  await signInAs(page, MEI);
  const plan = await createPlan(page, 'Allocation', budget.year);

  await signInAs(page, CARLOS);
  await page.goto(`/budgets/${budget.id}`);
  await expect(page.getByRole('heading', { name: budget.name, level: 1 })).toBeVisible();
  await page.getByRole('button', { name: `Allocate to ${plan.name}` }).click();
  const dialog = page.getByRole('dialog', { name: `Allocate to '${plan.name}'` });
  await dialog.getByRole('textbox', { name: /^Amount/ }).fill('25000.00');
  await dialog.getByRole('button', { name: 'Allocate', exact: true }).click();
  await expectToast(page, `Allocation to '${plan.name}' saved`);

  // The allocation is the plan's budget now.
  await signInAs(page, MEI);
  const saved = await callApi<{ budget: { amount: string; currency: string } | null }>(
    page,
    'GET',
    `/api/v1/plans/${plan.id}`,
  );
  expect(saved.budget).toEqual({ amount: '25000.00', currency: 'USD' });
});
