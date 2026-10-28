import { expect, test, type Page } from '@playwright/test';
import {
  action,
  escapeRegExp,
  expectNoAxeViolations,
  expectToast,
  signInAs,
  uniqueName,
} from './fixtures';
import { approveThroughApi } from './planning-changes';
import { createPlanningRecords, type PlanningRecords } from './planning';

// Feature 034 on the real API: line-item changes keep the activity's approval (D-322), and line items are
// coded with Shell's own lists (D-325). Tests make their own records.

const MEI = 'mei.lin@shell.com';
const APPROVED_NOTE = 'Approved — changing planned cost or dates will restart approval.';

async function startLineItem(page: Page, records: PlanningRecords, name: string, cost: string) {
  await page.goto(records.activityPath);
  await expect(page.getByRole('heading', { name: records.activityName, level: 1 })).toBeVisible();
  await action(page, 'Add line item').click();
  await expect(page.getByRole('heading', { name: 'New line item', level: 1 })).toBeVisible();
  await page.getByRole('textbox', { name: /^Line item name/ }).fill(name);
  await page.getByRole('textbox', { name: /^Planned cost/ }).fill(cost);
}

async function saveLineItem(page: Page, records: PlanningRecords, name: string) {
  await page.getByRole('button', { name: 'Save', exact: true }).click();
  await expectToast(page, `Line item '${name}' created`);
  await expect(page).toHaveURL(new RegExp(`${escapeRegExp(records.activityPath)}#line-items$`));
}

test('034 US1: line items of an approved activity are added and deleted without restarting its approval (FR-001, FR-005)', async ({
  page,
}) => {
  test.slow();
  await signInAs(page, MEI);
  const records = await createPlanningRecords(page, 'Keep approval', {
    withWorkflow: true,
    leaveEmpty: [],
  });
  await approveThroughApi(page, records);
  await signInAs(page, MEI);
  const name = uniqueName('Agency fee');

  await startLineItem(page, records, name, '5000');
  await saveLineItem(page, records, name);
  await expect(page.getByRole('alertdialog')).toHaveCount(0);
  await expect(page.getByText(APPROVED_NOTE)).toBeVisible();
  await expectNoAxeViolations(page);

  await page
    .locator('#line-items')
    .getByRole('row')
    .filter({ hasText: name })
    .first()
    .getByRole('link', { name })
    .click();
  await expect(page.getByRole('heading', { name, level: 1 })).toBeVisible();
  await page.getByRole('button', { name: `Actions for ${name}` }).click();
  await page.getByRole('menuitem', { name: 'Delete' }).click();
  const dialog = page.getByRole('alertdialog', { name: `Delete line item '${name}'?` });
  await expect(dialog).toContainText('It moves to the recycle bin; you can restore it.');
  await expect(dialog).not.toContainText(/approval/i);
  await expectNoAxeViolations(page);
  await dialog.getByRole('button', { name: 'Delete line item' }).click();
  await expectToast(page, `Line item '${name}' moved to the recycle bin`);
  await expect(page).toHaveURL(new RegExp(`${escapeRegExp(records.activityPath)}#line-items$`));
  await expect(page.getByText(APPROVED_NOTE)).toBeVisible();
});

test("034 US2: a line item is coded with Shell's spend type and GL code (FR-006, FR-007, FR-011)", async ({
  page,
}) => {
  await signInAs(page, MEI);
  const records = await createPlanningRecords(page, 'Shell lists', {
    withWorkflow: false,
    leaveEmpty: [],
  });
  const name = uniqueName('Trade show stand');
  await startLineItem(page, records, name, '2500');

  // Spend type and GL code have more than 10 values, so they are searchable lists.
  const spendType = page.getByRole('combobox', { name: /^Spend type/ });
  await spendType.click();
  await spendType.fill('BTL - Ev');
  await page.getByRole('option', { name: 'BTL - Event', exact: true }).click();
  const glCode = page.getByRole('combobox', { name: /^GL code/ });
  await glCode.click();
  const glCodes = page.getByRole('listbox', { name: 'GL code' }).getByRole('option');
  await expect(glCodes).toHaveCount(13);
  await expect(glCodes.first()).toHaveText('6140900 - PMTDR Credit');
  await page.getByRole('option', { name: '7250150 - Sponsorship programmes' }).click();
  await expectNoAxeViolations(page);
  await saveLineItem(page, records, name);

  const row = page.locator('#line-items').getByRole('row').filter({ hasText: name }).first();
  await row.getByRole('link', { name }).click();
  await expect(page.getByRole('heading', { name, level: 1 })).toBeVisible();
  await expect(page.getByText('BTL - Event', { exact: true }).first()).toBeVisible();
  await expect(
    page.getByText('7250150 - Sponsorship programmes', { exact: true }).first(),
  ).toBeVisible();
});
