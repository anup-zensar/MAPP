import { expect, test, type Page } from '@playwright/test';
import {
  action,
  chooseOption,
  escapeRegExp,
  expectNoAxeViolations,
  expectToast,
  signInAs,
} from './fixtures';
import { approveThroughApi, decideAs } from './planning-changes';
import { createPlanningRecords, planTree, treeRow } from './planning';

// Approval restart and the notification bell (008 US5, US6) and My profile's display preferences
// (003 US4), against the real API with the demo data.
// The demo data they read (backend/seed): FY2026 Lubricants Europe (P-1001, team Lubricants Europe,
// owner Priya Raman) with Helix launch and its activities. Tests create their own records.

const MEI = 'mei.lin@shell.com';
const CARLOS = 'carlos.mendes@shell.com';
const SARA = 'sara.novak@shell.com';

/** The year of the demo plans FY2026 Lubricants Europe and FY2026 Mobility China. */
const SEED_YEAR = 2026;

/** The notification bell in the top bar; its name says the unread count (008 plan, Bell). */
function bell(page: Page) {
  return page.getByRole('button', { name: /^Notifications(, \d+ unread)?$/ });
}

test('Approval restart and the bell: changing an approved activity restarts its approval, and a decline reaches the owner (008 US5, US6)', async ({
  page,
}) => {
  test.slow();
  await signInAs(page, MEI);
  const records = await createPlanningRecords(page, 'Restart', {
    withWorkflow: true,
    leaveEmpty: [],
  });
  const name = records.activityName;
  const id = records.activityId;
  await approveThroughApi(page, records);

  // Mei changes the planned cost of the Approved activity: the form says what will happen (006 FR-027).
  await signInAs(page, MEI);
  await page.goto(records.activityPath);
  await expect(page.getByRole('heading', { name, level: 1 })).toBeVisible();
  await expect(page.getByText('Approved', { exact: true }).first()).toBeVisible();
  await expect(
    page.getByText('Approved — changing planned cost or dates will restart approval.'),
  ).toBeVisible();
  await action(page, 'Edit').click();
  await expect(page.getByRole('heading', { name: 'Edit activity', level: 1 })).toBeVisible();
  await page.getByRole('textbox', { name: /^Planned cost/ }).fill('25000');
  await page.getByRole('button', { name: 'Save', exact: true }).click();
  const restart = page.getByRole('alertdialog', { name: 'This activity will need approval again' });
  await expect(
    restart.getByText(
      `Saving returns '${name}' to Created; it must be submitted for approval again.`,
    ),
  ).toBeVisible();
  await expect(restart.getByText('Planned cost: 20,000.00 USD → 25,000.00 USD')).toBeVisible();
  await restart.getByRole('button', { name: 'Save and restart approval' }).click();
  await expectToast(page, `Activity '${name}' saved`);

  // Created again; the approval history says why (008 FR-022, FR-025).
  await expect(page).toHaveURL(new RegExp(`${escapeRegExp(records.activityPath)}$`));
  await expect(page.getByText('Created', { exact: true }).first()).toBeVisible();
  await page.getByRole('tab', { name: 'Approval' }).click();
  const approval = page.locator('#approval');
  await expect(approval.getByText(/^Approval restarted/).first()).toBeVisible();
  await expect(
    approval.getByText('Planned cost changed from 20,000.00 USD to 25,000.00 USD'),
  ).toBeVisible();

  // The people who approved it are told (FR-035, D-161): Carlos Mendes's bell.
  await signInAs(page, CARLOS);
  await page.goto('/');
  await bell(page).click();
  const carlosPanel = page.getByRole('dialog', { name: 'Notifications' });
  await expect(
    carlosPanel.getByRole('button', {
      name: new RegExp(
        `^Approval restarted Mei Lin changed '${escapeRegExp(name)}' \\(${id}\\) — it needs approval again`,
      ),
    }),
  ).toBeVisible();

  // Mei submits it again; Carlos declines it.
  await signInAs(page, MEI);
  await page.goto(records.activityPath);
  await page.getByRole('button', { name: 'Submit for approval' }).first().click();
  const submit = page.getByRole('alertdialog', { name: `Submit '${name}' for approval?` });
  await submit.getByRole('button', { name: 'Submit for approval' }).click();
  await expectToast(page, `Activity '${name}' submitted for approval`);
  const reason = 'Please keep the planned cost at 20,000.00 USD.';
  await decideAs(page, CARLOS, id, 'decline', reason);

  // The owner's bell shows the decline with the comment, unread (FR-033, FR-035).
  await signInAs(page, MEI);
  await page.goto('/');
  await expect(page.getByRole('heading', { name: 'Welcome back, Mei.' })).toBeVisible();
  await expect(bell(page)).toHaveAccessibleName(/^Notifications, \d+ unread$/);
  await bell(page).click();
  const panel = page.getByRole('dialog', { name: 'Notifications' });
  const declined = new RegExp(
    `^Activity declined Carlos Mendes declined '${escapeRegExp(name)}' \\(${id}\\): "${escapeRegExp(reason)}"`,
  );
  const notice = panel.getByRole('button', { name: declined });
  await expect(notice).toHaveAccessibleName(/ Unread$/);
  await expectNoAxeViolations(page);

  // Choosing it marks it read and opens the activity's Approval section (FR-034).
  await notice.click();
  await expect(page).toHaveURL(new RegExp(`${escapeRegExp(records.activityPath)}#approval$`));
  await expect(page.getByText('Declined', { exact: true }).first()).toBeVisible();
  // The #approval link opens the Approval tab; who declined and why are in the notice above it.
  await expect(page.getByRole('tab', { name: 'Approval' })).toHaveAttribute(
    'aria-selected',
    'true',
  );
  await expect(page.getByText(/^Declined by Carlos Mendes on /).first()).toBeVisible();
  await expect(page.getByText(reason).first()).toBeVisible();
  await bell(page).click();
  await expect(panel.getByRole('button', { name: declined })).not.toHaveAccessibleName(/Unread$/);

  // Mark all as read empties the count.
  await panel.getByRole('button', { name: 'Mark all as read' }).click();
  await expect(bell(page)).toHaveAccessibleName('Notifications');
  await expect(panel.getByRole('button', { name: 'Mark all as read' })).toHaveCount(0);
});

test('My profile: the display currency and date format follow on the Plans list and the plan page (003 US4)', async ({
  page,
}) => {
  await signInAs(page, SARA);
  await page.goto('/');
  await page.getByRole('button', { name: 'Sara — account menu' }).click();
  await page.getByRole('menuitem', { name: 'My profile' }).click();
  await expect(page).toHaveURL(/\/profile$/);
  await expect(page.getByRole('heading', { name: 'My profile', level: 1 })).toBeVisible();
  await expect(
    page.getByText('Your details, teams and access, and how MAPP shows dates and money to you.'),
  ).toBeVisible();
  const preferences = page.getByRole('region', { name: 'Display preferences' });
  await expect(preferences).toContainText('USD — US dollar');
  await expect(preferences).toContainText('12 Mar 2026 (DD MMM YYYY)');
  await expect(page.getByRole('region', { name: 'My teams' })).toContainText('Lubricants Europe');
  await expectNoAxeViolations(page);

  await preferences.getByRole('button', { name: 'Edit preferences' }).click();
  const panel = page.getByRole('dialog', { name: 'Edit display preferences' });
  await chooseOption(panel.getByRole('combobox', { name: 'Display currency' }), 'EUR — Euro');
  await chooseOption(panel.getByRole('combobox', { name: 'Date format' }), '12/03/2026');
  await panel.getByRole('button', { name: 'Save', exact: true }).click();
  await expectToast(page, 'Your preferences saved');
  await expect(preferences).toContainText('EUR — Euro');
  await expect(preferences).toContainText('12/03/2026 (DD/MM/YYYY)');

  // The Plans list: FY2026 Lubricants Europe's 1,200,000.00 USD budget is 1,104,000.00 EUR at 0.92.
  await page.getByRole('navigation', { name: 'Main' }).getByRole('link', { name: 'Plans' }).click();
  await expect(page.getByRole('heading', { name: 'Plans', level: 1 })).toBeVisible();
  const card = page.getByRole('article', { name: 'FY2026 Lubricants Europe' });
  // Amounts are in the plan's currency (USD), each with its EUR value at the current rate (016).
  await expect(card).toContainText('1,200,000.00 USD');
  await expect(card).toContainText('about 1,104,000.00 EUR');
  await expect(card).toContainText(/Planned cost[\d,]+\.\d{2} USD≈ about [\d,]+\.\d{2} EUR/);
  await expect(card).not.toContainText('$');

  // The plan page: amounts in the plan's currency with their EUR value (016), and dates as
  // DD/MM/YYYY, in the header, the summary and the tree.
  await card.getByRole('link', { name: 'FY2026 Lubricants Europe', exact: true }).click();
  await expect(
    page.getByRole('heading', { name: 'FY2026 Lubricants Europe', level: 1 }),
  ).toBeVisible();
  await expect(page.getByRole('group', { name: 'Plan facts' })).toContainText(
    `01/01/${SEED_YEAR} – 31/12/${SEED_YEAR}`,
  );
  await expect(page.getByRole('region', { name: 'Summary' })).toContainText('≈ 1,104,000.00 EUR');
  const planRow = treeRow(page, 'P-1001');
  await expect(planRow).toContainText(`01/01/${SEED_YEAR} – 31/12/${SEED_YEAR}`);
  await expect(planRow).toContainText('1,104,000.00 EUR');
  // Helix launch's 700,000.00 USD budget is 644,000.00 EUR.
  const helix = treeRow(page, 'C-1001');
  await expect(helix).toContainText(`01/01/${SEED_YEAR} – 31/12/${SEED_YEAR}`);
  await expect(helix).toContainText('644,000.00 EUR');
  await expect(planTree(page)).not.toContainText('$');
});
