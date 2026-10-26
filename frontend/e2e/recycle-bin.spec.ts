import { expect, test, type Locator, type Page } from '@playwright/test';
import {
  callApi,
  escapeRegExp,
  expectNoAxeViolations,
  expectToast,
  itemAt,
  signInAs,
} from './fixtures';
import { approveThroughApi, createPlanningRecords } from './planning';

// End-to-end journeys of feature 011's Recycle bin (US1 find and restore, US4 restore with its parent,
// US5 delete permanently and empty), against the real API with the demo data (011 plan "Tests", T045).
//
// Every text and label comes from the "User-visible messages" table of specs/011 plan.md.
//
// The demo data read (seed/history.go): Priya Raman deleted the initiative "Trade shows" of Rimula fleet
// (FY2026 Lubricants Europe) two days before the seed time, with its activity "Fleet expo stand" and that
// activity's line item "Stand build". Other tests set the stage through the API as Mei Lin (records
// deleted there land in the bin as they would from their pages). The last test empties the bin, so it
// stays last; it deletes permanently as Alex Morgan, whose Administrator role holds that (014).

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

/** The recycle bin's entries as the API lists them. */
async function binItems(page: Page): Promise<BinItem[]> {
  return (await callApi<{ items: BinItem[] }>(page, 'GET', '/api/v1/recycle-bin')).items;
}

/** The bin's row of the record called `name` (each row is a list item named by an h2). */
function binRow(page: Page, name: string): Locator {
  return page
    .getByRole('listitem')
    .filter({ has: page.getByRole('heading', { level: 2, name, exact: true }) });
}

/**
 * What the bin shows once the searched rows left it: "Nothing matches these filters", or the empty bin
 * when they were its last entries.
 */
function gone(page: Page): Locator {
  return page
    .getByText('Nothing matches these filters')
    .or(page.getByText(/^Nothing in the recycle bin — /));
}

/** Opens the recycle bin and searches it for `text`. */
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
  await expect(page).toHaveTitle('Recycle bin · MAPP');
  // With no filter the count reads "3 of 3 items" (design system §11, rule 14).
  await expect(page.getByText(/^(\d+) of \1 items?$/)).toBeVisible();
  const row = binRow(page, 'Trade shows');
  await expect(row.getByText('Was in FY2026 Lubricants Europe › Rimula fleet')).toBeVisible();
  await expect(row.getByText('1 activity · 1 line item')).toBeVisible();
  await expect(row.getByText('Deleted by Priya Raman')).toBeVisible();
  await row.getByRole('button', { name: 'Show contents' }).click();
  await expect(row.getByText(`Activity 'Fleet expo stand' (${stand.recordId})`)).toBeVisible();
  await expectNoAxeViolations(page);

  // The search also looks inside each entry (FR-016).
  await page.getByRole('searchbox', { name: 'Search name or ID' }).fill(stand.recordId);
  await expect(page.getByText(/^1 of \d+ items?$/)).toBeVisible();
  await expect(row.getByText(`Contains ${stand.recordId} 'Fleet expo stand'`)).toBeVisible();

  // Step 1: Restore — the dialog says where it returns and with what (FR-018).
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

  // Its activity is back, Created (FR-019, FR-021: same ID).
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

  // Both left the bin, and the activity's page is back under its initiative.
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

  // Delete permanently: Cancel has the focus; the red button deletes it for good (FR-028). Alex's
  // Administrator role holds "Delete permanently and empty the recycle bin" (014).
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

  // Empty recycle bin: everything listed goes, in one step (FR-029).
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

  // Both are in the activity log, with Alex as the actor (FR-030, SC-007).
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
