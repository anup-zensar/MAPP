import { expect, test } from '@playwright/test';
import { expectToast, signInAs } from './fixtures';
import {
  createInitiative,
  createPlan,
  createPlanningRecords,
  idIn,
  planTree,
  treeRow,
} from './planning';

// Copying a plan (004 US6) and moving and copying campaigns, initiatives and activities
// (005 US6, 006 US5), against the real API with the demo data.

const MEI = 'mei.lin@shell.com';

test('Copy a plan: a planner copies a plan with everything inside into the next year and opens the copy (004 US6)', async ({
  page,
}) => {
  await signInAs(page, MEI);
  const records = await createPlanningRecords(page, 'Copy', {
    withWorkflow: false,
    leaveEmpty: [],
  });
  const { year } = records;
  const copyName = `${records.planName} next year`;

  await page.goto(`/plans/${records.planId}`);
  await expect(page.getByRole('heading', { name: records.planName, level: 1 })).toBeVisible();
  // The header's More menu: Copy · View history · Delete (004 plan, Plan header; design system §6.3).
  await page.getByRole('main').getByRole('button', { name: 'More', exact: true }).click();
  await page.getByRole('menuitem', { name: 'Copy' }).click();

  const dialog = page.getByRole('dialog', { name: `Copy plan '${records.planName}'` });
  const name = dialog.getByRole('textbox', { name: /^Name of the copy/ });
  await expect(name).toHaveValue(`${records.planName} (copy)`);
  await expect(dialog.getByRole('radio', { name: `Same year (${year})` })).toBeChecked();
  await expect(
    dialog.getByRole('checkbox', {
      name: 'Include campaigns, initiatives, activities and line items',
    }),
  ).toBeChecked();
  await expect(dialog.getByText('1 campaign, 1 initiative and 1 activity')).toBeVisible();
  await dialog.getByRole('radio', { name: `Next year (${year + 1})` }).check();
  await expect(dialog.getByText('Every date moves one year later.')).toBeVisible();
  await name.fill(copyName);
  await dialog.getByRole('button', { name: 'Copy plan' }).click();

  // The copy opens on its own page: a Draft of the next year with a new ID (FR-026 – FR-028).
  await expectToast(page, `Plan '${copyName}' created as a copy of '${records.planName}'`);
  await expect(page).toHaveURL(/\/plans\/P-\d+$/);
  const copyId = idIn(page.url(), /\/plans\/(P-\d+)$/);
  expect(copyId).not.toBe(records.planId);
  await expect(page.getByRole('heading', { name: copyName, level: 1 })).toBeVisible();
  // The header's Plan facts row: ID, owner and the copy's dates (design system v2).
  const facts = page.getByRole('group', { name: 'Plan facts' });
  await expect(facts).toContainText(copyId);
  await expect(facts).toContainText(`01 Jan – 31 Dec ${year + 1}`);
  await expect(facts.getByText('Mei Lin', { exact: true })).toBeVisible();
  await expect(page.getByText('Draft', { exact: true }).first()).toBeVisible();

  // Everything inside came along, one year later, with its planned cost rolled up again.
  await page.getByRole('button', { name: 'Expand all' }).click();
  await expect(treeRow(page, copyName)).toContainText(`01 Jan – 31 Dec ${year + 1}`);
  await expect(treeRow(page, copyName)).toContainText('$20,000.00');
  await expect(treeRow(page, records.campaignName)).toContainText(`01 Jan – 31 Dec ${year + 1}`);
  await expect(treeRow(page, records.initiativeName)).toContainText('$20,000.00');
  const activityRow = treeRow(page, records.activityName);
  await expect(activityRow).toContainText(`01 Mar – 31 May ${year + 1}`);
  await expect(activityRow).toContainText('Created');
  await expect(activityRow).toContainText('$20,000.00');
  // The copied activity has an ID of its own.
  await expect(
    planTree(page).getByRole('link', { name: records.activityName }),
  ).not.toHaveAttribute('href', new RegExp(`/activities/${records.activityId}$`));

  // The source keeps its year, and the copy is on the Plans list.
  await page.goto('/plans');
  await page.getByRole('searchbox', { name: 'Search plans' }).fill(copyName);
  const card = page.getByRole('article', { name: copyName });
  await expect(card).toContainText(`${copyId} · ${year + 1}`);
  await card.getByRole('link', { name: copyName, exact: true }).click();
  await expect(page).toHaveURL(new RegExp(`/plans/${copyId}$`));
  await page.goto(`/plans/${records.planId}`);
  await expect(page.getByRole('group', { name: 'Plan facts' })).toContainText(
    `01 Jan – 31 Dec ${year}`,
  );
});

test('Move and copy: an activity moves to another initiative and its campaign is copied into another plan; the roll-ups follow at source and destination (005 US6, 006 US5)', async ({
  page,
}) => {
  test.slow();
  await signInAs(page, MEI);
  const records = await createPlanningRecords(page, 'Move', {
    withWorkflow: false,
    leaveEmpty: [],
  });
  const target = await createInitiative(
    page,
    records.campaignId,
    'Target',
    records.year,
    '30000.00',
  );
  const otherPlan = await createPlan(page, 'Copy target');
  const name = records.activityName;

  // Move the activity (006 US5): the dialog lists the plan's initiatives, its own one unavailable.
  await page.goto(records.activityPath);
  await expect(page.getByRole('heading', { name, level: 1 })).toBeVisible();
  await page
    .getByRole('button', { name: `Actions for ${name}` })
    .first()
    .click();
  await page.getByRole('menuitem', { name: 'Move' }).click();
  const move = page.getByRole('dialog', { name: `Move activity '${name}'` });
  await expect(
    move.getByText(
      `Choose another initiative in '${records.planName}'. The activity keeps its status, line items and values.`,
    ),
  ).toBeVisible();
  await expect(
    move.getByRole('radio', { name: `${records.initiativeName} (this initiative)` }),
  ).toBeDisabled();
  await expect(move.getByText("It's already in this initiative.")).toBeVisible();
  await move.getByRole('radio', { name: target.name, exact: true }).check();
  await move.getByRole('button', { name: 'Move activity' }).click();
  await expectToast(page, `Activity '${name}' moved to '${target.name}'`);

  // The planned cost left the source initiative and arrived at the target; the campaign holds both.
  await page.goto(`/plans/${records.planId}`);
  await page.getByRole('button', { name: 'Expand all' }).click();
  const source = treeRow(page, records.initiativeName);
  await expect(source).toContainText('0 activities');
  await expect(source).not.toContainText('$20,000.00');
  const destination = treeRow(page, target.name);
  await expect(destination).toContainText('1 activity');
  await expect(destination).toContainText('$20,000.00');
  // 20,000 of the target's 30,000 budget: 67 % used.
  await expect(destination.getByText('67% used', { exact: true })).toBeVisible();
  await expect(treeRow(page, records.campaignName)).toContainText('$20,000.00');
  await expect(treeRow(page, records.planName)).toContainText('$20,000.00');

  // Copy the campaign into another plan of the same year (005 US6).
  await page.getByRole('button', { name: `Actions for ${records.campaignName}` }).click();
  await page.getByRole('menuitem', { name: 'Copy' }).click();
  const copy = page.getByRole('dialog', { name: `Copy campaign '${records.campaignName}'` });
  await expect(copy.getByText('It copies 2 initiatives and 1 activity.')).toBeVisible();
  await expect(
    copy.getByText(
      'Copies get new IDs. Activities start as Created, without approvals, comments or history.',
    ),
  ).toBeVisible();
  await copy.getByRole('radio', { name: otherPlan.name, exact: true }).check();
  const copyName = `${records.campaignName} (copy)`;
  await expect(copy.getByRole('textbox', { name: /^Name of the copy/ })).toHaveValue(copyName);
  await copy.getByRole('button', { name: 'Copy campaign' }).click();
  await expectToast(page, `Campaign '${copyName}' created`);

  // The destination plan opens with the copy and its planned cost rolled up there.
  await expect(page).toHaveURL(new RegExp(`/plans/${otherPlan.id}$`));
  await expect(page.getByRole('heading', { name: otherPlan.name, level: 1 })).toBeVisible();
  await page.getByRole('button', { name: 'Expand all' }).click();
  await expect(treeRow(page, copyName)).toContainText('$20,000.00');
  await expect(treeRow(page, copyName)).toContainText('2 initiatives');
  await expect(treeRow(page, target.name)).toContainText('$20,000.00');
  await expect(treeRow(page, name)).toContainText('Created');
  await expect(treeRow(page, otherPlan.name)).toContainText('$20,000.00');
  // 20,000 of the plan's 100,000 budget: 20 % used.
  await expect(treeRow(page, otherPlan.name).getByText('20% used', { exact: true })).toBeVisible();

  // The source plan is unchanged: still one campaign and 20,000.00.
  await page.goto(`/plans/${records.planId}`);
  await expect(treeRow(page, records.planName)).toContainText('$20,000.00');
  await expect(planTree(page).getByRole('row').filter({ hasText: copyName })).toHaveCount(0);
});
