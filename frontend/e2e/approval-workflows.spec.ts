import { expect, test } from '@playwright/test';
import { action, expectNoAxeViolations, expectToast, signInAs, uniqueName } from './fixtures';
import { CHINA_WORKFLOW } from './planning';

// Feature 008 US4 (T055 journey 4) against the real API: Alex Morgan, the administrator, manages approval
// workflows in Settings. The seeded workflows are only read; the one set inactive is the test's own.

const ALEX = 'alex.morgan@shell.com';

test('008 US4 (T055 journey 4): an administrator creates a workflow, sees the open-rounds note, and sets one inactive and active again', async ({
  page,
}) => {
  test.slow();
  await signInAs(page, ALEX);
  await page.goto('/settings/approval-workflows');
  await expect(page.getByRole('heading', { name: 'Approval workflows', level: 1 })).toBeVisible();
  const china = page.getByRole('row', { name: new RegExp(`^${CHINA_WORKFLOW} `) });
  await expect(china).toContainText('Level 1 · Carlos Mendes → Level 2 · Alex Morgan');
  await expect(china).toContainText(/Used by \d+ activit(y|ies) \(\d+ waiting for approval\)/);

  // Every problem at once: no name, no approver, and no backup while escalation is on.
  await page.getByRole('button', { name: 'New approval workflow' }).click();
  const panel = page.getByRole('dialog', { name: 'New approval workflow' });
  await panel.getByRole('button', { name: 'Save', exact: true }).click();
  const problems = panel.getByRole('alert');
  await expect(
    problems.getByRole('heading', { name: '3 problems need your attention' }),
  ).toBeVisible();
  await expect(problems.getByRole('link', { name: 'Enter a name.' })).toBeVisible();
  await expect(
    problems.getByRole('link', { name: 'Level 1: choose at least one approver.' }),
  ).toBeVisible();
  await expectNoAxeViolations(page);

  const name = uniqueName('Workflow');
  await panel.getByRole('textbox', { name: 'Name', exact: true }).fill(name);
  await panel.getByRole('combobox', { name: 'Approvers' }).fill('Carlos');
  await page.getByRole('option', { name: /^Carlos Mendes/ }).click();
  await panel.getByRole('checkbox', { name: 'Escalate to the backup approver' }).uncheck();
  await panel.getByRole('button', { name: 'Save', exact: true }).click();
  await expectToast(page, `Approval workflow '${name}' created`);
  const created = page.getByRole('row', { name: new RegExp(`^${name} `) });
  await expect(created).toContainText('Level 1 · Carlos Mendes');
  await expect(created).toContainText('Not used yet');

  // Editing a workflow with rounds in progress says they keep the levels they were submitted with.
  await page.getByRole('button', { name: `Actions for ${CHINA_WORKFLOW}` }).click();
  await page.getByRole('menuitem', { name: 'Edit' }).click();
  const edit = page.getByRole('dialog', { name: /^Edit/ });
  await expect(
    edit.getByText(/waiting for approval keeps? the levels (it was|they were) submitted with\.$/),
  ).toBeVisible();
  await edit.getByRole('button', { name: 'Cancel' }).click();
  await expect(edit).toHaveCount(0);

  await page.getByRole('button', { name: `Actions for ${name}` }).click();
  await page.getByRole('menuitem', { name: 'Set inactive' }).click();
  await expectToast(page, `Approval workflow '${name}' set inactive`);
  await page
    .getByRole('group', { name: 'Status' })
    .getByRole('button', { name: /^Inactive/ })
    .click();
  await expect(created).toContainText('Inactive');

  await page.getByRole('button', { name: `Actions for ${name}` }).click();
  await page.getByRole('menuitem', { name: 'Reactivate' }).click();
  await expectToast(page, `Approval workflow '${name}' reactivated`);
  await page
    .getByRole('group', { name: 'Status' })
    .getByRole('button', { name: /^Active/ })
    .click();
  await expect(created).toContainText('Active');
  await expect(action(page, 'New approval workflow')).toBeVisible();
});
