import { expect, test, type Page } from '@playwright/test';
import { callApi, expectToast, signInAs } from './fixtures';
import { lubricantsActivity, STANDARD_WORKFLOW, type LubricantsActivity } from './lubricants';
import { createPlanningRecords } from './planning';
import { cancelCurrentDelegations, decideAs } from './planning-changes';

// Feature 017's journeys 1–4 (plan, Tests › End-to-end) against the real API, on activities of the tests'
// own. The approved-automatically case is proven in brd-gaps.spec.ts; journeys 5 and 6 are in
// approval-monitoring.spec.ts.

const PRIYA = 'priya.raman@shell.com';
const DANIEL = 'daniel.okoro@shell.com';
const MEI = 'mei.lin@shell.com';
const CARLOS = 'carlos.mendes@shell.com';
const ALEX = 'alex.morgan@shell.com';

async function submitThroughApi(page: Page, id: string, version: number): Promise<void> {
  await callApi(page, 'POST', `/api/v1/activities/${id}/approval/submit`, { version, comment: '' });
}

/** As `email`: opens the activity in Approvals (Waiting for me, or `tab`) and approves its level. */
async function approveOnScreen(page: Page, email: string, name: string, tab?: RegExp) {
  await signInAs(page, email);
  await page.goto('/approvals');
  if (tab) {
    await page.getByRole('tab', { name: tab }).click();
  }
  await page.getByRole('searchbox', { name: 'Search activity name or ID' }).fill(name);
  await page.getByRole('button', { name, exact: true }).click();
  await page
    .getByRole('dialog', { name })
    .getByRole('button', { name: 'Approve', exact: true })
    .click();
  return page.getByRole('alertdialog', { name: `Approve '${name}'?` });
}

async function openApprovalSection(page: Page, activity: { path: string; name: string }) {
  await page.goto(activity.path);
  await expect(page.getByRole('heading', { name: activity.name, level: 1 })).toBeVisible();
  await page.getByRole('tab', { name: 'Approval' }).click();
  return page.locator('#approval');
}

test('017 US1 (T047 journey 1): the workflow reads in words, and an activity from $10,000 waits for the owner’s line manager', async ({
  page,
}) => {
  await signInAs(page, ALEX);
  await page.goto('/settings/approval-workflows');
  await page.getByRole('link', { name: STANDARD_WORKFLOW }).click();
  const details = page.getByRole('dialog', { name: STANDARD_WORKFLOW });
  await expect(
    details.getByText(
      "Approved automatically below 10,000 USD, otherwise by the owner's line manager.",
      { exact: false },
    ),
  ).toBeVisible();
  const levels = details.getByRole('region', { name: 'Levels' });
  await expect(levels).toContainText('Line manager (level 1)');
  await expect(levels).toContainText(
    'Approved automatically when planned cost is below $10,000.00',
  );
  await expect(levels).toContainText('Only needed when planned cost is at least $100,000.00');
  await expect(levels).toContainText('Only needed when business unit is Mobility');

  await signInAs(page, PRIYA);
  const activity = await lubricantsActivity(page, 'Manager', '30000.00');
  await page.goto(activity.path);
  await page.getByRole('button', { name: 'Submit for approval' }).first().click();
  const submit = page.getByRole('alertdialog', { name: `Submit '${activity.name}' for approval?` });
  await expect(submit).toContainText('Daniel Okoro');
  await expect(submit).toContainText('Line manager · Daniel Okoro (line manager of Priya Raman)');
  await expect(submit).toContainText(
    'Marketing administrator · Not needed — only when planned cost is at least $100,000.00',
  );
  await submit.getByRole('button', { name: 'Submit for approval' }).click();
  await expectToast(page, `Activity '${activity.name}' submitted for approval`);

  const section = await openApprovalSection(page, activity);
  await expect(section).toContainText(
    'Not needed — only when planned cost is at least $100,000.00',
  );
  const history = section.getByRole('region', { name: 'Approval history' });
  await expect(history).toContainText('Only when planned cost is at least $100,000.00');
  await expect(history).not.toContainText(/only when only when/i);
});

test('017 US2 (T047 journey 2): a Mobility activity skips level 2 and needs both people of level 3', async ({
  page,
}) => {
  test.slow();
  await signInAs(page, MEI);
  const records = await createPlanningRecords(page, 'Rules', {
    withWorkflow: true,
    leaveEmpty: [],
    workflow: STANDARD_WORKFLOW,
  });
  const activity = { path: records.activityPath, name: records.activityName };
  await submitThroughApi(page, records.activityId, records.activityVersion);
  // Level 1 is Mei's line manager, Carlos Mendes; level 2 is not needed below $100,000.00.
  await decideAs(page, CARLOS, records.activityId, 'approve');

  let approve = await approveOnScreen(page, DANIEL, activity.name);
  await expect(approve).toContainText('Level 3 of 3');
  await approve.getByRole('button', { name: 'Approve', exact: true }).click();
  await expectToast(page, `Level 3 approved — '${activity.name}' now waits for`);
  await signInAs(page, MEI);
  const section = await openApprovalSection(page, activity);
  await expect(section).toContainText(
    'Not needed — only when planned cost is at least $100,000.00',
  );
  await expect(section).toContainText('1 of 2 approved');
  await expect(page.getByText('Submitted', { exact: true }).first()).toBeVisible();

  approve = await approveOnScreen(page, ALEX, activity.name);
  await expect(approve).toContainText('1 of 2 already approved.');
  await expect(approve).toContainText('Level 3 is complete after this approval.');
  await expect(approve).toContainText('This is the last level — the activity becomes Approved.');
  await approve.getByRole('button', { name: 'Approve', exact: true }).click();
  await expectToast(page, `Activity '${activity.name}' approved`);
  await signInAs(page, MEI);
  await openApprovalSection(page, activity);
  await expect(page.getByText('Approved', { exact: true }).first()).toBeVisible();
});

test('017 US3 (T047 journey 3): an approver is left out of his own activity, and the backup approves for him', async ({
  page,
}) => {
  await signInAs(page, DANIEL);
  const activity = await lubricantsActivity(page, 'Own', '20000.00', {
    owner: 'Daniel Okoro',
    workflow: 'Lubricants EU approval',
  });
  // Daniel owns no other activity in the demo (home.spec.ts), so his goes to the recycle bin at the end.
  try {
    await page.goto(activity.path);
    await page.getByRole('button', { name: 'Submit for approval' }).first().click();
    const submit = page.getByRole('alertdialog', {
      name: `Submit '${activity.name}' for approval?`,
    });
    await expect(submit).toContainText('Daniel Okoro is left out: owns this activity.');
    await expect(submit).toContainText('The backup approver Alex Morgan approves this level.');
    await submit.getByRole('button', { name: 'Submit for approval' }).click();
    await expectToast(page, `Activity '${activity.name}' submitted for approval`);

    const approve = await approveOnScreen(page, ALEX, activity.name);
    await expect(approve).toContainText('Level 1 of 1 · Alex Morgan');
    await approve.getByRole('button', { name: 'Approve', exact: true }).click();
    await expectToast(page, `Activity '${activity.name}' approved`);

    await signInAs(page, DANIEL);
    const section = await openApprovalSection(page, activity);
    const history = section.getByRole('region', { name: 'Approval history' });
    await expect(history).toContainText(/Approved · Level 1 · .+Alex Morgan/);
    await expect(history).toContainText(
      /Submitted · Level 1 · .+Daniel OkoroApprover: Alex Morgan/,
    );
  } finally {
    await signInAs(page, DANIEL);
    await callApi(page, 'DELETE', `/api/v1/activities/${activity.id}`);
  }
});

test('017 US4 (T047 journey 4): Daniel delegates to Carlos, who sees the banner and approves as his delegate', async ({
  page,
}) => {
  test.slow();
  await signInAs(page, PRIYA);
  const activity: LubricantsActivity = await lubricantsActivity(page, 'Delegation', '30000.00');
  await submitThroughApi(page, activity.id, activity.version);

  const today = new Date();
  const iso = (offset: number) =>
    new Date(today.getTime() + offset * 86_400_000).toISOString().slice(0, 10);
  await signInAs(page, DANIEL);
  await page.goto('/approvals');
  await page.getByRole('button', { name: 'Delegate my approvals' }).click();
  await page
    .getByRole('dialog', { name: 'Approval delegation' })
    .getByRole('button', { name: 'Add delegation' })
    .click();
  const form = page.getByRole('dialog', { name: 'Add delegation' });
  await form.getByRole('combobox', { name: 'Delegate' }).fill('Carlos');
  await page.getByRole('option', { name: /^Carlos Mendes/ }).click();
  await form.getByRole('textbox', { name: 'Start date' }).fill(iso(0));
  await form.getByRole('textbox', { name: 'End date' }).fill(iso(2));
  await form.getByRole('button', { name: 'Save', exact: true }).click();
  await expectToast(page, 'Delegation to Carlos Mendes saved');

  try {
    await signInAs(page, CARLOS);
    await page.goto('/approvals');
    await expect(page.getByText(/^You are approving for Daniel Okoro until /)).toBeVisible();
    await page.getByRole('searchbox', { name: 'Search activity name or ID' }).fill(activity.name);
    await expect(page.getByRole('row', { name: new RegExp(activity.name) })).toContainText(
      'as delegate of Daniel Okoro',
    );
    const approve = await approveOnScreen(page, CARLOS, activity.name);
    await expect(approve).toContainText('You approve as delegate of Daniel Okoro.');
    await approve.getByRole('button', { name: 'Approve', exact: true }).click();
    await expectToast(page, `Activity '${activity.name}' approved`);

    await signInAs(page, PRIYA);
    const section = await openApprovalSection(page, activity);
    await expect(section.getByRole('region', { name: 'Approval history' })).toContainText(
      'Carlos Mendes as delegate of Daniel Okoro',
    );
  } finally {
    await cancelCurrentDelegations(page, DANIEL);
  }
});
