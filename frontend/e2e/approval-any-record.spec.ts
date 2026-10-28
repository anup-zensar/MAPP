import { expect, test } from '@playwright/test';
import {
  action,
  callApi,
  chooseOption,
  expectNoAxeViolations,
  expectToast,
  signInAs,
  uniqueName,
} from './fixtures';
import { CHINA_WORKFLOW, createPlanningRecords } from './planning';

// Feature 031 against the real API with the demo data: re-submitting a Submitted activity, and
// approval workflows that approve purchase requests. Tests create their own records.

const MEI = 'mei.lin@shell.com';
const ALEX = 'alex.morgan@shell.com';

test('031 US1: the owner re-submits a Submitted activity; round 1 ends as replaced and round 2 starts', async ({
  page,
}) => {
  test.slow();
  await signInAs(page, MEI);
  const records = await createPlanningRecords(page, 'Resubmit', {
    withWorkflow: true,
    leaveEmpty: [],
  });
  const name = records.activityName;
  await callApi(page, 'POST', `/api/v1/activities/${records.activityId}/approval/submit`, {
    version: records.activityVersion,
    comment: '',
  });

  await page.goto(records.activityPath);
  await expect(page.getByRole('heading', { name, level: 1 })).toBeVisible();
  await page.getByRole('tab', { name: 'Approval' }).click();
  const approval = page.locator('#approval');
  await expect(approval.getByText(`Round 1 · ${CHINA_WORKFLOW}`)).toBeVisible();
  await expect(approval.getByRole('button', { name: 'Recall submission' })).toBeVisible();
  await approval.getByRole('button', { name: 'Re-submit for approval' }).click();

  const dialog = page.getByRole('alertdialog', { name: `Re-submit '${name}' for approval?` });
  await expect(
    dialog.getByText(
      'Round 1, waiting at Level 1, ends and round 2 starts at the first level that needs a decision.',
    ),
  ).toBeVisible();
  await expectNoAxeViolations(page);
  await dialog.getByRole('button', { name: 'Re-submit for approval' }).click();

  await expectToast(page, `Activity '${name}' re-submitted for approval`);
  await expect(approval.getByText(`Round 2 · ${CHINA_WORKFLOW}`)).toBeVisible();
  await expect(page.getByText('Submitted', { exact: true }).first()).toBeVisible();
  await expect(
    approval.getByRole('heading', { name: /^Round 1 · Replaced by a new submission/ }),
  ).toBeVisible();
});

test('031 US2: an administrator creates a purchase request workflow; activities are not offered it', async ({
  page,
}) => {
  test.slow();
  const workflowName = uniqueName('PR approval');
  await signInAs(page, ALEX);
  await page.goto('/settings/approval-workflows');
  await expect(page.getByRole('heading', { name: 'Approval workflows', level: 1 })).toBeVisible();
  await page.getByRole('button', { name: 'New approval workflow' }).click();

  const panel = page.getByRole('dialog', { name: 'New approval workflow' });
  await panel.getByRole('textbox', { name: /^Name/ }).fill(workflowName);
  await chooseOption(panel.getByRole('combobox', { name: /^Approves/ }), 'Purchase requests');
  await panel.getByRole('radio', { name: 'Person named on the record' }).check();
  await chooseOption(
    panel.getByRole('combobox', { name: /^Field that names the approver/ }),
    'MoA approver',
  );
  await panel.getByRole('checkbox', { name: 'Escalate to the backup approver' }).uncheck();
  await expectNoAxeViolations(page);
  await panel.getByRole('button', { name: 'Save', exact: true }).click();
  await expectToast(page, `Approval workflow '${workflowName}' created`);

  await page
    .getByRole('group', { name: 'Approves' })
    .getByRole('button', { name: /^Purchase requests/ })
    .click();
  await expect(page).toHaveURL(/approves=purchaseRequest/);
  const row = page.getByRole('row', { name: new RegExp(workflowName) });
  await expect(row).toContainText('Purchase requests');
  await expect(row).toContainText('Level 1 · MoA approver');
  await expect(page.getByRole('row', { name: new RegExp(CHINA_WORKFLOW) })).toHaveCount(0);

  await signInAs(page, MEI);
  const records = await createPlanningRecords(page, 'Workflow kind', {
    withWorkflow: false,
    leaveEmpty: [],
  });
  await page.goto(records.activityPath);
  await action(page, 'Edit').click();
  await expect(page.getByRole('heading', { name: 'Edit activity', level: 1 })).toBeVisible();
  const offered = page.getByLabel(/^Approval workflow/).locator('option');
  await expect(offered.filter({ hasText: CHINA_WORKFLOW })).toHaveCount(1);
  await expect(offered.filter({ hasText: workflowName })).toHaveCount(0);
});
