import { expect, test, type Locator, type Page } from '@playwright/test';
import {
  callApi,
  escapeRegExp,
  expectNoAxeViolations,
  expectToast,
  itemAt,
  signInAs,
} from './fixtures';
import { approveThroughApi } from './planning-changes';
import { createPlanningRecords } from './planning';

// In the demo data Priya deleted the initiative "Trade shows" of Rimula fleet with its activity
// "Fleet expo stand" and its line item "Stand build". The last test empties the bin, so stays last.

const PRIYA = 'priya.raman@shell.com';
const MEI = 'mei.lin@shell.com';
const ALEX = 'alex.morgan@shell.com';

interface BinRecord {
  recordType: string;
  recordId: string;
  recordName: string;
}

interface BinItem {
  id: string;
  recordName: string;
  records: BinRecord[];
}

async function binItems(page: Page): Promise<BinItem[]> {
  return (await callApi<{ items: BinItem[] }>(page, 'GET', '/api/v1/recycle-bin')).items;
}

function binRow(page: Page, name: string): Locator {
  return page
    .getByRole('listitem')
    .filter({ has: page.getByRole('heading', { level: 2, name, exact: true }) });
}

/** "Nothing matches these filters", or the empty bin when the searched rows were its last. */
function gone(page: Page): Locator {
  return page
    .getByText('Nothing matches these filters')
    .or(page.getByText(/^Nothing in the recycle bin — /));
}

async function searchBin(page: Page, text: string): Promise<void> {
  await page.goto('/recycle-bin');
  await expect(page.getByRole('heading', { level: 1, name: 'Recycle bin' })).toBeVisible();
  await page.getByRole('searchbox', { name: 'Search name or ID' }).fill(text);
}

test('US1: the seeded initiative is found by its activity’s ID and restored in three steps (FR-014 – FR-019, SC-001)', async ({
  page,
}) => {
  await signInAs(page, PRIYA);
  const trade = (await binItems(page)).find((item) => item.recordName === 'Trade shows');
  const stand = trade?.records.find((record) => record.recordName === 'Fleet expo stand');
  if (!trade || !stand) {
    throw new Error("The demo recycle bin has no 'Trade shows' with 'Fleet expo stand'");
  }

  await page.goto('/recycle-bin');
  await expect(page).toHaveTitle('Recycle bin · SP&A One');
  await expect(page.getByText(/^(\d+) of \1 items?$/)).toBeVisible();
  const row = binRow(page, 'Trade shows');
  await expect(row.getByText('Was in FY2026 Lubricants Europe › Rimula fleet')).toBeVisible();
  await expect(row.getByText('1 activity · 1 line item')).toBeVisible();
  await expect(row.getByText('Deleted by Priya Raman')).toBeVisible();
  await row.getByRole('button', { name: 'Show contents' }).click();
  await expect(row.getByText(`Activity 'Fleet expo stand' (${stand.recordId})`)).toBeVisible();
  await expectNoAxeViolations(page);

  await page.getByRole('searchbox', { name: 'Search name or ID' }).fill(stand.recordId);
  await expect(page.getByText(/^1 of \d+ items?$/)).toBeVisible();
  await expect(row.getByText(`Contains ${stand.recordId} 'Fleet expo stand'`)).toBeVisible();

  await row.getByRole('button', { name: "Restore initiative 'Trade shows'" }).click();
  const dialog = page.getByRole('alertdialog', { name: "Restore initiative 'Trade shows'?" });
  await expect(
    dialog.getByText(
      'It returns to FY2026 Lubricants Europe › Rimula fleet with 1 activity and 1 line item.',
    ),
  ).toBeVisible();
  await expectNoAxeViolations(page);

  // Step 2: confirm. Step 3: View opens it in its plan.
  await dialog.getByRole('button', { name: 'Restore initiative' }).click();
  await expectToast(page, "Initiative 'Trade shows' restored");
  await page.getByRole('button', { name: 'View', exact: true }).click();
  await expect(page).toHaveURL(
    new RegExp(`/plans/P-\\d+/initiatives/${itemAt(trade.records, 0).recordId}$`),
  );
  await expect(page.getByRole('dialog', { name: 'Trade shows' })).toBeVisible();

  // Its activity is back, Created, with the same ID.
  await page.goto(`/activities/${stand.recordId}`);
  await expect(page.getByRole('heading', { level: 1, name: 'Fleet expo stand' })).toBeVisible();
  await expect(page.getByText('Created', { exact: true }).first()).toBeVisible();
});

test('US1: an approved activity restored from the bin comes back as Created (FR-019, FR-025)', async ({
  page,
}) => {
  test.slow();
  await signInAs(page, MEI);
  const records = await createPlanningRecords(page, 'Bin approved', {
    withWorkflow: true,
    leaveEmpty: [],
  });
  await approveThroughApi(page, records);
  await signInAs(page, MEI);
  // Deleted from its page, after the Undo message closed: it is in the bin.
  await callApi(page, 'DELETE', `/api/v1/activities/${records.activityId}`);

  await searchBin(page, records.activityName);
  const name = records.activityName;
  await binRow(page, name)
    .getByRole('button', { name: `Restore activity '${name}'` })
    .click();
  const dialog = page.getByRole('alertdialog', { name: `Restore activity '${name}'?` });
  await expect(
    dialog.getByText('It comes back as Created and must be submitted for approval again.'),
  ).toBeVisible();
  await dialog.getByRole('button', { name: 'Restore activity' }).click();
  await expectToast(page, `Activity '${name}' restored`);
  await page.getByRole('button', { name: 'View', exact: true }).click();

  await expect(page).toHaveURL(new RegExp(`/activities/${records.activityId}$`));
  await expect(page.getByRole('heading', { level: 1, name })).toBeVisible();
  await expect(page.getByText('Created', { exact: true }).first()).toBeVisible();
});

test('US4: an activity whose initiative is also in the bin comes back with it — Restore both (FR-017, FR-023)', async ({
  page,
}) => {
  test.slow();
  await signInAs(page, MEI);
  const records = await createPlanningRecords(page, 'Bin both', {
    withWorkflow: false,
    leaveEmpty: [],
  });
  await callApi(page, 'DELETE', `/api/v1/activities/${records.activityId}`);
  await callApi(page, 'DELETE', `/api/v1/initiatives/${records.initiativeId}`);

  await searchBin(page, records.activityName);
  const row = binRow(page, records.activityName);
  await expect(
    row.getByText(`Its initiative '${records.initiativeName}' is also in the recycle bin`),
  ).toBeVisible();
  await row.getByRole('button', { name: `Restore activity '${records.activityName}'` }).click();
  const dialog = page.getByRole('alertdialog', {
    name: `Restore activity '${records.activityName}'?`,
  });
  await expect(
    dialog.getByText(
      `Its initiative '${records.initiativeName}' is also in the recycle bin. Restoring brings these back first:`,
    ),
  ).toBeVisible();
  await expect(
    dialog.getByText(
      new RegExp(
        `^Initiative '${escapeRegExp(records.initiativeName)}' \\(${records.initiativeId}\\)`,
      ),
    ),
  ).toBeVisible();
  await expectNoAxeViolations(page);
  await dialog.getByRole('button', { name: 'Restore both' }).click();
  await expectToast(page, `Activity '${records.activityName}' restored`);

  await expect(gone(page)).toBeVisible();
  await page.goto(records.activityPath);
  await expect(page.getByRole('heading', { level: 1, name: records.activityName })).toBeVisible();
  await expect(page.getByRole('link', { name: records.initiativeName })).toBeVisible();
});

test('US5: Delete permanently and Empty recycle bin, both in the activity log (FR-028 – FR-030)', async ({
  page,
}) => {
  test.slow();
  await signInAs(page, MEI);
  const records = await createPlanningRecords(page, 'Bin purge', {
    withWorkflow: false,
    leaveEmpty: [],
  });
  await callApi(page, 'DELETE', `/api/v1/activities/${records.activityId}`);
  const name = records.activityName;

  // Delete permanently: Cancel has the focus; the red button deletes it for good.
  await signInAs(page, ALEX);
  await searchBin(page, name);
  await binRow(page, name)
    .getByRole('button', { name: `Actions for activity '${name}'` })
    .click();
  await page.getByRole('menuitem', { name: 'Delete permanently' }).click();
  const purge = page.getByRole('alertdialog', { name: `Delete '${name}' permanently?` });
  await expect(purge.getByText("This can't be undone.")).toBeVisible();
  await expect(purge.getByRole('button', { name: 'Cancel' })).toBeFocused();
  await expectNoAxeViolations(page);
  await purge.getByRole('button', { name: 'Delete permanently' }).click();
  await expectToast(page, `Activity '${name}' deleted permanently`);
  await expect(gone(page)).toBeVisible();

  await callApi(page, 'DELETE', `/api/v1/initiatives/${records.initiativeId}`);
  await page.goto('/recycle-bin');
  // Destructive and rare, so it is in the header's More menu (design system §5.1).
  await page.getByRole('button', { name: 'More' }).click();
  await page.getByRole('menuitem', { name: 'Empty recycle bin' }).click();
  const empty = page.getByRole('alertdialog', { name: 'Empty the recycle bin?' });
  await expect(empty.getByText(/^This permanently deletes \d+ items?/)).toBeVisible();
  await expect(empty.getByText("This can't be undone.")).toBeVisible();
  await expect(empty.getByRole('button', { name: 'Cancel' })).toBeFocused();
  await expectNoAxeViolations(page);
  await empty.getByRole('button', { name: 'Empty recycle bin' }).click();
  await expectToast(page, /The recycle bin is empty — \d+ items? deleted permanently/);
  await expect(page.getByText('Nothing in the recycle bin', { exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: 'More' })).toHaveCount(0);

  await page.goto('/activity-log');
  await expect(
    page.getByText(/^Alex Morgan emptied the recycle bin \(\d+ items?\)/).first(),
  ).toBeVisible();
  await expect(
    page
      .getByText(
        new RegExp(
          `^Alex Morgan permanently deleted activity '${escapeRegExp(name)}' \\(permanently deleted\\)`,
        ),
      )
      .first(),
  ).toBeVisible();
});
