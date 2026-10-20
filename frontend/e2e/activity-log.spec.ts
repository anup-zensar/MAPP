import { expect, test, type Locator, type Page } from '@playwright/test';
import { callApi, escapeRegExp, expectNoAxeViolations, signInAs } from './fixtures';
import {
  changeActivityPlannedCost,
  changeLineItemPlannedCost,
  createInitiative,
  createLineItem,
  createPlanningRecords,
} from './planning';

// End-to-end journeys of feature 011's Activity log (US2) and the History sections of the record pages
// (US3), against the real API with the demo data (011 plan "Tests", T046).
//
// Every text and label comes from the "User-visible messages" table of specs/011 plan.md; the filter chips
// and their buttons are 004's FilterChips ("Remove filter: <chip>", D-197).
//
// Tests create their own records (names from uniqueName) as Mei Lin and never assert global counts; the
// demo history (seed/history.go) gives the log more than one page of entries. The whole log is read as
// Alex Morgan, whose Administrator role sees every entry; Mei's own log holds only her market's (014).

const MEI = 'mei.lin@northstar.demo';
const ALEX = 'alex.morgan@northstar.demo';

/** One entry of a list of log entries, found by the words of its sentence. */
function entry(scope: Page | Locator, sentence: string): Locator {
  return scope
    .getByRole('listitem')
    .filter({ hasText: new RegExp(escapeRegExp(sentence)) })
    .first();
}

/** The chip of a filter in use (its button removes it). */
function chip(page: Page, label: string): Locator {
  return page.getByRole('button', { name: `Remove filter: ${label}`, exact: true });
}

test('US2: the cards, filters, chips, details and record links of the activity log (FR-001 – FR-009, SC-008)', async ({
  page,
}) => {
  test.slow();
  await signInAs(page, MEI);
  // Today's changes by Mei: the records of a new plan, a planned cost changed once and an initiative
  // that went to the recycle bin.
  const records = await createPlanningRecords(page, 'Log', { withWorkflow: false, leaveEmpty: [] });
  await changeActivityPlannedCost(page, records.activityId, '25000.00');
  const binned = await createInitiative(
    page,
    records.campaignId,
    'Log binned',
    records.year,
    '1000.00',
  );
  await callApi(page, 'DELETE', `/api/initiatives/${binned.id}`);

  // The whole log, with more than one page of entries: Alex sees every entry (014).
  await signInAs(page, ALEX);
  await page.goto('/activity-log');
  await expect(page.getByRole('heading', { level: 1, name: 'Activity log' })).toBeVisible();
  await expect(page).toHaveTitle('Activity log · NorthStar');
  await expect(
    page.getByText("See who changed what, and when. Entries can't be edited or removed."),
  ).toBeVisible();
  // Entries have no edit or delete control (FR-010).
  await expect(page.getByRole('button', { name: /^(Edit|Delete)\b/ })).toHaveCount(0);
  await expectNoAxeViolations(page);

  // A summary card's link applies its filter (FR-002).
  await page.getByRole('link', { name: /^Changes today: [\d,]+ — show these entries$/ }).click();
  await expect(page).toHaveURL(/[?&]range=today\b/);
  await expect(chip(page, 'Date: Today')).toBeVisible();

  // Person and action (FR-003): chips and "N of M entries".
  await page.getByLabel('Person', { exact: true }).selectOption({ label: 'Mei Lin' });
  await expect(chip(page, 'Person: Mei Lin')).toBeVisible();
  await page.getByRole('button', { name: 'Action: Any action' }).click();
  await page.getByRole('menuitemcheckbox', { name: 'Changed', exact: true }).click();
  await page.keyboard.press('Escape');
  await expect(chip(page, 'Action: Changed')).toBeVisible();
  await expect(page).toHaveURL(/[?&]person=U-\d+/);
  await expect(page.getByText(/^[\d,]+ of [\d,]+ entries$/)).toBeVisible();

  // The activity's change expands into its field change (FR-007), with the keyboard (SC-008).
  const changed = entry(page, `Mei Lin changed activity '${records.activityName}'`);
  await expect(changed).toBeVisible();
  await expect(changed.getByText('Changed', { exact: true })).toBeVisible();
  const toggle = changed.getByRole('button', { name: 'Show details' });
  await toggle.focus();
  await page.keyboard.press('Enter');
  await expect(changed.getByRole('button', { name: 'Hide details' })).toHaveAttribute(
    'aria-expanded',
    'true',
  );
  await expect(changed.getByText(/Planned cost: .*20,000\.00.* → .*25,000\.00/)).toBeVisible();

  // A record that exists links to its page (FR-008).
  await changed.getByRole('link', { name: records.activityName, exact: true }).click();
  await expect(page).toHaveURL(new RegExp(`${escapeRegExp(records.activityPath)}$`));
  await expect(page.getByRole('heading', { level: 1, name: records.activityName })).toBeVisible();
  await page.goBack();

  // A record in the recycle bin links to its row there, which is highlighted (FR-008, FR-017).
  await chip(page, 'Action: Changed').click();
  await page.getByRole('button', { name: 'Action: Any action' }).click();
  await page.getByRole('menuitemcheckbox', { name: 'Deleted', exact: true }).click();
  await page.keyboard.press('Escape');
  const deleted = entry(page, `Mei Lin deleted initiative '${binned.name}' (in the recycle bin)`);
  await expect(deleted).toBeVisible();
  await deleted.getByRole('link', { name: binned.name, exact: true }).click();
  await expect(page).toHaveURL(new RegExp(`/recycle-bin\\?record=${binned.id}$`));
  await expect(
    page.getByRole('heading', { level: 2, name: binned.name, exact: true }),
  ).toBeFocused();

  // Back on the filtered log, Clear filters; then Load more on the whole log, with the keyboard (FR-009,
  // SC-008): the demo history holds more than 50 entries.
  await page.goBack();
  await expect(chip(page, 'Action: Deleted')).toBeVisible();
  await page.getByRole('button', { name: 'Clear filters' }).click();
  await expect(page).toHaveURL(/\/activity-log$/);
  // With no filter the count reads "1,204 of 1,204 entries" (design system §11, rule 14).
  await expect(page.getByText(/^([\d,]+) of \1 entries$/)).toBeVisible();
  await expect(page.getByText(/^Showing the last 50 of [\d,]+$/)).toBeVisible();
  await page.getByRole('button', { name: 'Load more' }).focus();
  await page.keyboard.press('Enter');
  await expect(page.getByText(/^Showing the last 50 of [\d,]+$/)).toHaveCount(0);
  await expect(
    page.getByRole('heading', { level: 2, name: /^Today · [\d,]+ entr(y|ies)$/ }),
  ).toBeVisible();
});

test('US3: an activity’s History, with and without its line items, and Open in activity log (FR-011 – FR-013, SC-005)', async ({
  page,
}) => {
  test.slow();
  await signInAs(page, MEI);
  // The activity's planned cost changes twice and one of its line items once.
  const records = await createPlanningRecords(page, 'History', {
    withWorkflow: false,
    leaveEmpty: [],
  });
  const lineItem = await createLineItem(page, records.activityId, 'History', '5000.00');
  await changeActivityPlannedCost(page, records.activityId, '22000.00');
  await changeActivityPlannedCost(page, records.activityId, '24000.00');
  await changeLineItemPlannedCost(page, lineItem, '6000.00');

  // Step 1–2: the activity's ⋯ More › View history brings its History section up.
  await page.goto(records.activityPath);
  await expect(page.getByRole('heading', { level: 1, name: records.activityName })).toBeVisible();
  await page.getByRole('button', { name: `Actions for ${records.activityName}` }).click();
  await page.getByRole('menuitem', { name: 'View history' }).click();
  const heading = page.getByRole('heading', { level: 2, name: 'History', exact: true });
  await expect(heading).toBeFocused();
  const history = page.getByRole('region', { name: 'History', exact: true });
  await expect(history.getByText('Every change to this activity, newest first')).toBeVisible();

  // The activity's own entries, newest first; no line item yet (US3 scenario 1).
  const activityChanges = history
    .getByRole('listitem')
    .filter({ hasText: `Mei Lin changed activity '${records.activityName}'` });
  await expect(activityChanges).toHaveCount(2);
  await expect(entry(history, `Mei Lin created activity '${records.activityName}'`)).toBeVisible();
  await expect(history.getByText(`line item '${lineItem.name}'`)).toHaveCount(0);
  await activityChanges.first().getByRole('button', { name: 'Show details' }).click();
  await expect(
    activityChanges.first().getByText(/Planned cost: .*22,000\.00.* → .*24,000\.00/),
  ).toBeVisible();
  await expectNoAxeViolations(page);

  // Include line items (US3 scenario 2): the line item's entries appear, naming it.
  const include = history.getByRole('checkbox', { name: 'Include line items' });
  await expect(include).not.toBeChecked();
  await include.check();
  await expect(entry(history, `Mei Lin changed line item '${lineItem.name}'`)).toBeVisible();
  await expect(entry(history, `Mei Lin created line item '${lineItem.name}'`)).toBeVisible();

  // The choice is remembered for Mei (FR-012).
  await page.reload();
  await expect(
    page
      .getByRole('region', { name: 'History', exact: true })
      .getByRole('checkbox', { name: 'Include line items' }),
  ).toBeChecked();

  // Step 3: Open in activity log, with the Record filter and the items inside as a chip (FR-013).
  await page
    .getByRole('region', { name: 'History', exact: true })
    .getByRole('link', { name: 'Open in activity log' })
    .click();
  await expect(page).toHaveURL(
    new RegExp(`/activity-log\\?record=${records.activityId}&inside=1$`),
  );
  await expect(
    chip(
      page,
      `Record: Activity '${records.activityName}' (${records.activityId}) and items inside`,
    ),
  ).toBeVisible();
  await expect(entry(page, `Mei Lin changed line item '${lineItem.name}'`)).toBeVisible();
});
