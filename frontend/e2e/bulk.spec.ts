import { expect, test } from '@playwright/test';
import { callApi, expectNoAxeViolations, expectToast, signInAs } from './fixtures';
import { createPlanningRecords } from './planning';

// Feature 012 US1 and US5 against the real API: tick rows in All work, change their owner, delete them
// and undo the delete. The records are the test's own (a plan of next year as Mei Lin).

const MEI = 'mei.lin@shell.com';

test('012 US1, US5: change the owner of ticked activities, then delete them and undo', async ({
  page,
}) => {
  await signInAs(page, MEI);
  const records = await createPlanningRecords(page, 'Bulk', {
    withWorkflow: false,
    leaveEmpty: [],
  });
  const copy = await callApi<{ id: string; name: string }>(
    page,
    'POST',
    `/api/v1/activities/${records.activityId}/copy`,
    { initiativeId: records.initiativeId },
  );

  await page.goto(`/all-work?year=${records.year}&plan=${records.planId}`);
  const grid = page.getByRole('table', { name: 'All work grid' });
  const bar = page.getByRole('region', { name: 'Selection' });
  const tick = async () => {
    await grid.getByRole('checkbox', { name: `Select activity '${records.activityName}'` }).check();
    await grid.getByRole('checkbox', { name: `Select activity '${copy.name}'` }).check();
    await expect(bar.getByText('2 selected · 2 activities')).toBeVisible();
  };
  await tick();
  await expectNoAxeViolations(page);

  await bar.getByRole('button', { name: 'Change owner' }).click();
  const dialog = page.getByRole('dialog', { name: 'Change owner' });
  const owner = dialog.getByRole('combobox', { name: /^New owner/ });
  await owner.click();
  await owner.press('ArrowDown');
  await page
    .getByRole('listbox', { name: 'New owner' })
    .getByRole('option', { name: /^Carlos Mendes/ })
    .click();
  await expect(dialog.getByText('Set the new owner on 2 records.')).toBeVisible();
  await expectNoAxeViolations(page);
  await dialog.getByRole('button', { name: 'Change owner' }).click();
  await expectToast(page, '2 changed');

  const activity = await callApi<{ owner: { name: string } }>(
    page,
    'GET',
    `/api/v1/activities/${records.activityId}`,
  );
  expect(activity.owner.name).toBe('Carlos Mendes');
  const log = await callApi<{ items: { details: string }[] }>(
    page,
    'GET',
    `/api/v1/activity-log?recordId=${records.activityId}`,
  );
  expect(log.items[0]?.details).toMatch(/Bulk change of 2 activities \(BLK-\d+\)/);

  await tick();
  await bar.getByRole('button', { name: 'Delete' }).click();
  const confirm = page.getByRole('alertdialog', { name: 'Delete 2 activities?' });
  await expect(
    confirm.getByText('They move to the recycle bin; you can restore them.', { exact: false }),
  ).toBeVisible();
  await expect(confirm.getByRole('button', { name: 'Cancel' })).toBeFocused();
  await expectNoAxeViolations(page);
  await confirm.getByRole('button', { name: 'Delete 2 records' }).click();
  await expectToast(page, '2 records moved to the recycle bin');
  await page.getByRole('button', { name: 'Undo' }).click();
  await expectToast(page, '2 records restored');
  await expect(
    grid.getByRole('checkbox', { name: `Select activity '${copy.name}'` }),
  ).toBeVisible();
});
