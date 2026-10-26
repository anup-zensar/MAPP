import { expect, test, type Locator, type Page } from '@playwright/test';
import { callApi, escapeRegExp, expectNoAxeViolations, expectToast, signInAs } from './fixtures';
import { changeActivityPlannedCost, createInitiative, createPlanningRecords } from './planning';

// End-to-end journeys across feature 011's Recycle bin, Activity log and History sections, and 009's
// Recent changes card that reads the same log (011 T045, T046, T049; 009 T013, T017). They follow one
// record through its whole life on screen: created, changed, deleted from its page, found in the bin,
// restored, its History and the log narrowed to it; then a permanent delete; then Home.
//
// Every text and label comes from the "User-visible messages" tables of specs/011 and specs/009 plan.md.
// Tests create their own records (names from uniqueName) as Mei Lin and never assert global counts, so
// they run in any order with the other files; none of them empties the bin or restores the demo's
// "Trade shows" (recycle-bin.spec.ts relies on both). Deleting permanently is Alex Morgan's, whose
// Administrator role holds it (014).

const MEI = 'mei.lin@shell.com';
const ALEX = 'alex.morgan@shell.com';

/** The list item of an entry whose sentence holds `sentence` (the avatar's initials come first). */
function entry(scope: Page | Locator, sentence: string): Locator {
  return scope.getByRole('listitem').filter({ hasText: sentence }).first();
}

/** The chip of a filter in use (its button removes it). */
function chip(page: Page, label: string): Locator {
  return page.getByRole('button', { name: `Remove filter: ${label}`, exact: true });
}

/** The bin's row of the record called `name` (each row is a list item named by an h2). */
function binRow(page: Page, name: string): Locator {
  return page
    .getByRole('listitem')
    .filter({ has: page.getByRole('heading', { level: 2, name, exact: true }) });
}

/**
 * Expects the entries of `list` (log entries, newest first) to be exactly `sentences`, in that order,
 * each by its actor with its action chip and its time — the full date and time in the time's label.
 */
async function expectEntries(
  list: Locator,
  sentences: { text: string; chip: string }[],
): Promise<void> {
  const items = list.locator(':scope > li');
  await expect(items).toHaveCount(sentences.length);
  const year = new Date().getFullYear();
  for (const [index, { text, chip: word }] of sentences.entries()) {
    const item = items.nth(index);
    await expect(item).toContainText(text);
    await expect(item.getByText(word, { exact: true })).toBeVisible();
    // "When": hh:mm on screen, "20 Oct 2026, 14:05" in the button's name (and tooltip).
    await expect(
      item.getByRole('button', { name: new RegExp(`^\\d{2} \\w{3} ${year}, \\d{2}:\\d{2}$`) }),
    ).toBeVisible();
  }
}

test('An activity deleted from its page is found in the recycle bin and restored; its History and the activity log show each step with who and when (011 US1 – US3, FR-003, FR-005, FR-008, FR-011, FR-018)', async ({
  page,
}) => {
  test.slow();
  await signInAs(page, MEI);
  // Created and changed once (through the API, to set the stage).
  const records = await createPlanningRecords(page, 'Life', {
    withWorkflow: false,
    leaveEmpty: [],
  });
  await changeActivityPlannedCost(page, records.activityId, '21000.00');
  const name = records.activityName;
  const place = `${records.planName} › ${records.campaignName} › ${records.initiativeName}`;
  // Mei's recycle bin lists only entries of plans she can see (014 FR-007) — the demo's "Trade shows" is
  // in Lubricants Europe — so an initiative of her own waits there too, and the bin is not empty once the
  // activity is restored.
  const binned = await createInitiative(
    page,
    records.campaignId,
    'Life other',
    records.year,
    '1000.00',
  );
  await callApi(page, 'DELETE', `/api/v1/initiatives/${binned.id}`);

  // Delete it from its page.
  await page.goto(records.activityPath);
  await expect(page.getByRole('heading', { level: 1, name })).toBeVisible();
  await page.getByRole('button', { name: `Actions for ${name}` }).click();
  await page.getByRole('menuitem', { name: 'Delete' }).click();
  await page
    .getByRole('alertdialog', { name: `Delete activity '${name}'?` })
    .getByRole('button', { name: 'Delete activity' })
    .click();
  await expectToast(page, `Activity '${name}' moved to the recycle bin`);

  // Find it in the recycle bin from the main navigation, by its ID.
  await page
    .getByRole('navigation', { name: 'Main' })
    .getByRole('link', { name: 'Recycle bin' })
    .click();
  await expect(page.getByRole('heading', { level: 1, name: 'Recycle bin' })).toBeVisible();
  await page.getByRole('searchbox', { name: 'Search name or ID' }).fill(records.activityId);
  await expect(page.getByText(/^1 of [\d,]+ items?$/)).toBeVisible();
  const row = binRow(page, name);
  await expect(row.getByText(records.activityId, { exact: true })).toBeVisible();
  await expect(row.getByText(`Was in ${place}`)).toBeVisible();
  await expect(row.getByText('Nothing inside')).toBeVisible();
  await expect(row.getByText('Deleted by Mei Lin')).toBeVisible();
  await expect(row.getByRole('button', { name: /^(just now|\d+ minutes? ago), / })).toBeVisible();
  await expectNoAxeViolations(page);

  // Restore it: the dialog says where it returns; View opens it again.
  await row.getByRole('button', { name: `Restore activity '${name}'` }).click();
  const dialog = page.getByRole('alertdialog', { name: `Restore activity '${name}'?` });
  await expect(dialog.getByText(`It returns to ${place}.`)).toBeVisible();
  await expectNoAxeViolations(page);
  await dialog.getByRole('button', { name: 'Restore activity' }).click();
  await expectToast(page, `Activity '${name}' restored`);
  await expect(page.getByText('Nothing matches these filters')).toBeVisible();
  await page.getByRole('button', { name: 'View', exact: true }).click();
  await expect(page).toHaveURL(new RegExp(`/activities/${records.activityId}$`));
  await expect(page.getByRole('heading', { level: 1, name })).toBeVisible();

  // Its History, from the page's More menu: restored, deleted, changed and created, newest first.
  await page.getByRole('button', { name: `Actions for ${name}` }).click();
  await page.getByRole('menuitem', { name: 'View history' }).click();
  await expect(page.getByRole('heading', { level: 2, name: 'History', exact: true })).toBeFocused();
  const history = page.getByRole('region', { name: 'History', exact: true });
  const lifeOf = [
    { text: `Mei Lin restored activity '${name}'`, chip: 'Restored' },
    { text: `Mei Lin deleted activity '${name}'`, chip: 'Deleted' },
    { text: `Mei Lin changed activity '${name}'`, chip: 'Changed' },
    { text: `Mei Lin created activity '${name}'`, chip: 'Created' },
  ];
  await expectEntries(history.locator('ol[role="list"]'), lifeOf);
  await expect(history.getByRole('heading', { name: /^Today · \d+ entr(y|ies)$/ })).toBeVisible();

  // The activity log, narrowed by a search for its name: the same four entries (FR-003, FR-005).
  await page
    .getByRole('navigation', { name: 'Main' })
    .getByRole('link', { name: 'Activity log' })
    .click();
  await expect(page.getByRole('heading', { level: 1, name: 'Activity log' })).toBeVisible();
  // With no filter the count reads "1,204 of 1,204 entries" (design system §11, rule 14).
  await expect(page.getByText(/^([\d,]+) of \1 entries$/)).toBeVisible();
  await page.getByRole('searchbox', { name: 'Search name, ID or person' }).fill(name);
  await expect(page).toHaveURL(/[?&]q=/);
  await expect(chip(page, `Search: '${name}'`)).toBeVisible();
  await expect(page.getByText(/^4 of [\d,]+ entries$/)).toBeVisible();
  const log = page.getByRole('main');
  await expectEntries(log.locator('ol[role="list"]'), lifeOf);
  await expectNoAxeViolations(page);

  // An action filter narrows it further, to the restore.
  await page.getByRole('button', { name: 'Action: Any action' }).click();
  await page.getByRole('menuitemcheckbox', { name: 'Restored', exact: true }).click();
  await page.keyboard.press('Escape');
  await expect(chip(page, 'Action: Restored')).toBeVisible();
  await expect(page.getByText(/^1 of [\d,]+ entries$/)).toBeVisible();
  await expectEntries(log.locator('ol[role="list"]'), [lifeOf[0]]);

  // Its record link opens the activity, which exists again (FR-008).
  await entry(page, lifeOf[0].text).getByRole('link', { name, exact: true }).click();
  await expect(page).toHaveURL(new RegExp(`${escapeRegExp(records.activityPath)}$`));
  await expect(page.getByRole('heading', { level: 1, name })).toBeVisible();

  // Back on the log, Clear filters shows every entry again.
  await page.goBack();
  await expect(chip(page, 'Action: Restored')).toBeVisible();
  await page.getByRole('button', { name: 'Clear filters' }).click();
  await expect(page).toHaveURL(/\/activity-log$/);
  // No filter in use: the count reads "<n> of <n> entries" (design system §11, rule 14).
  await expect(page.getByText(/^([\d,]+) of \1 entries$/)).toBeVisible();
});

test('Delete permanently: the activity leaves the bin for good and the log keeps every step, without a link (011 US5, FR-008, FR-028, FR-030)', async ({
  page,
}) => {
  test.slow();
  await signInAs(page, MEI);
  const records = await createPlanningRecords(page, 'Gone', {
    withWorkflow: false,
    leaveEmpty: [],
  });
  const name = records.activityName;
  // Deleted from its page as in the journey above; here through the API.
  await callApi(page, 'DELETE', `/api/v1/activities/${records.activityId}`);

  // Deleting permanently needs "Delete permanently and empty the recycle bin": Alex's (014).
  await signInAs(page, ALEX);
  await page.goto('/recycle-bin');
  await page.getByRole('searchbox', { name: 'Search name or ID' }).fill(name);
  const row = binRow(page, name);
  await expect(row).toBeVisible();
  await row.getByRole('button', { name: `Actions for activity '${name}'` }).click();
  await page.getByRole('menuitem', { name: 'Delete permanently' }).click();
  const purge = page.getByRole('alertdialog', { name: `Delete '${name}' permanently?` });
  await expect(purge.getByText("This can't be undone.")).toBeVisible();
  await expect(purge.getByRole('button', { name: 'Cancel' })).toBeFocused();
  await expectNoAxeViolations(page);
  await purge.getByRole('button', { name: 'Delete permanently' }).click();
  await expectToast(page, `Activity '${name}' deleted permanently`);
  await expect(page.getByText('Nothing matches these filters')).toBeVisible();

  // Its page is gone too.
  await page.goto(records.activityPath);
  await expect(
    page.getByText('This activity no longer exists — someone may have deleted it.'),
  ).toBeVisible();

  // The log: permanently deleted, deleted, created — the name is plain text now, with no link.
  await page.goto(`/activity-log?q=${encodeURIComponent(name)}`);
  await expect(page.getByText(/^3 of [\d,]+ entries$/)).toBeVisible();
  await expectEntries(page.getByRole('main').locator('ol[role="list"]'), [
    {
      text: `Alex Morgan permanently deleted activity '${name}' (permanently deleted)`,
      chip: 'Permanently deleted',
    },
    { text: `Mei Lin deleted activity '${name}' (permanently deleted)`, chip: 'Deleted' },
    { text: `Mei Lin created activity '${name}' (permanently deleted)`, chip: 'Created' },
  ]);
  await expect(
    page.getByRole('main').locator('ol[role="list"]').getByRole('link', { name }),
  ).toHaveCount(0);
});

test('Home: Recent changes shows the latest changes in Mei’s plans, newest first, with links while the record exists (009 FR-015, FR-016)', async ({
  page,
}) => {
  test.slow();
  await signInAs(page, MEI);
  const records = await createPlanningRecords(page, 'Recent', {
    withWorkflow: false,
    leaveEmpty: [],
  });
  const binned = await createInitiative(
    page,
    records.campaignId,
    'Recent binned',
    records.year,
    '1000.00',
  );
  await callApi(page, 'DELETE', `/api/v1/initiatives/${binned.id}`);
  await changeActivityPlannedCost(page, records.activityId, '26000.00');

  await page.goto('/');
  await expect(page.getByRole('heading', { level: 1, name: 'Welcome back, Mei.' })).toBeVisible();
  const card = page.getByRole('region', { name: 'Recent changes' });
  await expect(card.getByText("Latest changes in your teams' plans")).toBeVisible();
  const items = card.getByRole('listitem');

  // Newest first: the planned-cost change, then the deleted initiative (in the recycle bin, no link).
  await expect(items.first()).toContainText(
    `Mei Lin changed planned cost of activity '${records.activityName}' (${records.activityId})`,
  );
  await expect(
    items.first().getByRole('button', { name: /^(just now|\d+ minutes? ago)/ }),
  ).toBeVisible();
  await expect(items.nth(1)).toContainText(
    `Mei Lin deleted initiative '${binned.name}' (${binned.id}) (in recycle bin)`,
  );
  await expect(items.nth(1).getByRole('link')).toHaveCount(0);
  await expect(items.nth(2)).toContainText(
    `Mei Lin created initiative '${binned.name}' (${binned.id}) (in recycle bin)`,
  );
  await expectNoAxeViolations(page);

  // The record link opens the activity; View activity log opens the log.
  await items.first().getByRole('link', { name: records.activityName, exact: true }).click();
  await expect(page).toHaveURL(new RegExp(`${escapeRegExp(records.activityPath)}$`));
  await expect(page.getByRole('heading', { level: 1, name: records.activityName })).toBeVisible();
  await page.goBack();
  await card.getByRole('link', { name: 'View activity log' }).click();
  await expect(page).toHaveURL(/\/activity-log$/);
  await expect(page.getByRole('heading', { level: 1, name: 'Activity log' })).toBeVisible();
});
