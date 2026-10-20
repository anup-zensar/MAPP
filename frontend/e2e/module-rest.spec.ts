import { readFile } from 'node:fs/promises';
import { expect, test, type Page } from '@playwright/test';
import {
  action,
  chooseOption,
  escapeRegExp,
  expectNoAxeViolations,
  expectToast,
  signInAs,
  uniqueName,
} from './fixtures';
import {
  approveThroughApi,
  createInitiative,
  createPlan,
  createPlanningRecords,
  decideAs,
  idIn,
  planTree,
  treeRow,
} from './planning';

// End-to-end journeys through the rest of the Plan module, against the real API with the demo data:
// copying a plan (004 US6); moving and copying campaigns, initiatives and activities (005 US6, 006 US5);
// approval restart and the notification bell (008 US5, US6); My profile's display preferences (003 US4);
// the Home dashboard (009); All work (010); and the Settings screens of 002 US3, US5 and US6. Feature 011
// (Activity log, Recycle bin) has its own journeys in activity-log.spec.ts and recycle-bin.spec.ts, and
// 012 (bulk actions and import) is deferred (D-209), so neither is exercised here.
//
// Every text and label comes from the "User-visible messages" tables of specs/002–010 plan.md (or the
// decisions that changed them, such as D-216 for date ranges and D-219 for All work's chip names).
//
// Tests create their own records (names from uniqueName) and never assert global counts. The demo data
// they read (backend/seed): FY2026 Lubricants Europe (P-1001, team Lubricants Europe, budget
// 1,200,000.00 USD, owner Priya Raman) with Helix launch (700,000.00) and its activities Fleet roadshow
// (A-1005, declined by Daniel Okoro), Trade magazine ads (A-1004, waiting for Daniel Okoro at level 1 of
// 1) and Billboard winter wave (A-1003, Complete, 5 Jan – 28 Feb 2026, 45,000.00); the Shell attributes
// Region (depends on Cluster; the cluster "Retail – India" has no region yet) and Spend category (line
// item, three options); and the rate EUR 0.92 per USD.

const ALEX = 'alex.morgan@northstar.demo';
const PRIYA = 'priya.raman@northstar.demo';
const DANIEL = 'daniel.okoro@northstar.demo';
const MEI = 'mei.lin@northstar.demo';
const CARLOS = 'carlos.mendes@northstar.demo';
const SARA = 'sara.novak@northstar.demo';

/** The year of the demo plans FY2026 Lubricants Europe and FY2026 Mobility China. */
const SEED_YEAR = 2026;

/** The current year: the default year of a new finance budget item, and the year All work marks Today in. */
const THIS_YEAR = new Date().getFullYear();

/** The notification bell in the top bar; its name says the unread count (008 plan, Bell). */
function bell(page: Page) {
  return page.getByRole('button', { name: /^Notifications(, \d+ unread)?$/ });
}

// ---------------------------------------------------------------------------------------------------

test('Copy a plan: a planner copies a plan with everything inside into the next year and opens the copy (004 US6)', async ({
  page,
}) => {
  await signInAs(page, MEI);
  const records = await createPlanningRecords(page, 'Copy', {
    withWorkflow: false,
    leaveEmpty: [],
  });
  const { year } = records;
  const copyName = `${records.planName} next year`;

  await page.goto(`/plans/${records.planId}`);
  await expect(page.getByRole('heading', { name: records.planName, level: 1 })).toBeVisible();
  // The header's More menu: Copy · View history · Delete (004 plan, Plan header).
  await page
    .getByRole('button', { name: `Actions for ${records.planName}` })
    .first()
    .click();
  await page.getByRole('menuitem', { name: 'Copy' }).click();

  const dialog = page.getByRole('dialog', { name: `Copy plan '${records.planName}'` });
  const name = dialog.getByRole('textbox', { name: /^Name of the copy/ });
  await expect(name).toHaveValue(`${records.planName} (copy)`);
  await expect(dialog.getByRole('radio', { name: `Same year (${year})` })).toBeChecked();
  await expect(
    dialog.getByRole('checkbox', {
      name: 'Include campaigns, initiatives, activities and line items',
    }),
  ).toBeChecked();
  await expect(dialog.getByText('1 campaign, 1 initiative and 1 activity')).toBeVisible();
  await dialog.getByRole('radio', { name: `Next year (${year + 1})` }).check();
  await expect(dialog.getByText('Every date moves one year later.')).toBeVisible();
  await name.fill(copyName);
  await dialog.getByRole('button', { name: 'Copy plan' }).click();

  // The copy opens on its own page: a Draft of the next year with a new ID (FR-026 – FR-028).
  await expectToast(page, `Plan '${copyName}' created as a copy of '${records.planName}'`);
  await expect(page).toHaveURL(/\/plans\/P-\d+$/);
  const copyId = idIn(page.url(), /\/plans\/(P-\d+)$/);
  expect(copyId).not.toBe(records.planId);
  await expect(page.getByRole('heading', { name: copyName, level: 1 })).toBeVisible();
  const facts = page.getByRole('list', { name: 'Plan facts' });
  await expect(facts).toContainText(copyId);
  await expect(facts).toContainText(`Year ${year + 1}`);
  await expect(facts).toContainText('Owner: Mei Lin');
  await expect(page.getByText('Draft', { exact: true }).first()).toBeVisible();

  // Everything inside came along, one year later, with its planned cost rolled up again.
  await page.getByRole('button', { name: 'Expand all' }).click();
  await expect(treeRow(page, copyName)).toContainText(`01 Jan – 31 Dec ${year + 1}`);
  await expect(treeRow(page, copyName)).toContainText('$20,000.00');
  await expect(treeRow(page, records.campaignName)).toContainText(`01 Jan – 31 Dec ${year + 1}`);
  await expect(treeRow(page, records.initiativeName)).toContainText('$20,000.00');
  const activityRow = treeRow(page, records.activityName);
  await expect(activityRow).toContainText(`01 Mar – 31 May ${year + 1}`);
  await expect(activityRow).toContainText('Created');
  await expect(activityRow).toContainText('$20,000.00');
  // The copied activity has an ID of its own.
  await expect(
    planTree(page).getByRole('link', { name: records.activityName }),
  ).not.toHaveAttribute('href', new RegExp(`/activities/${records.activityId}$`));

  // The source keeps its year, and the copy is on the Plans list.
  await page.goto('/plans');
  await page.getByRole('searchbox', { name: 'Search name or ID' }).fill(copyName);
  const card = page.getByRole('article', { name: copyName });
  await expect(card).toContainText(`${copyId} · ${year + 1}`);
  await card.getByRole('link', { name: copyName, exact: true }).click();
  await expect(page).toHaveURL(new RegExp(`/plans/${copyId}$`));
  await page.goto(`/plans/${records.planId}`);
  await expect(page.getByRole('list', { name: 'Plan facts' })).toContainText(`Year ${year}`);
});

test('Move and copy: an activity moves to another initiative and its campaign is copied into another plan; the roll-ups follow at source and destination (005 US6, 006 US5)', async ({
  page,
}) => {
  test.slow();
  await signInAs(page, MEI);
  const records = await createPlanningRecords(page, 'Move', {
    withWorkflow: false,
    leaveEmpty: [],
  });
  const target = await createInitiative(
    page,
    records.campaignId,
    'Target',
    records.year,
    '30000.00',
  );
  const otherPlan = await createPlan(page, 'Copy target');
  const name = records.activityName;

  // Move the activity (006 US5): the dialog lists the plan's initiatives, its own one unavailable.
  await page.goto(records.activityPath);
  await expect(page.getByRole('heading', { name, level: 1 })).toBeVisible();
  await page
    .getByRole('button', { name: `Actions for ${name}` })
    .first()
    .click();
  await page.getByRole('menuitem', { name: 'Move' }).click();
  const move = page.getByRole('dialog', { name: `Move activity '${name}'` });
  await expect(
    move.getByText(
      `Choose another initiative in '${records.planName}'. The activity keeps its status, line items and values.`,
    ),
  ).toBeVisible();
  await expect(
    move.getByRole('radio', { name: `${records.initiativeName} (this initiative)` }),
  ).toBeDisabled();
  await expect(move.getByText("It's already in this initiative.")).toBeVisible();
  await move.getByRole('radio', { name: target.name, exact: true }).check();
  await move.getByRole('button', { name: 'Move activity' }).click();
  await expectToast(page, `Activity '${name}' moved to '${target.name}'`);

  // The planned cost left the source initiative and arrived at the target; the campaign holds both.
  await page.goto(`/plans/${records.planId}`);
  await page.getByRole('button', { name: 'Expand all' }).click();
  const source = treeRow(page, records.initiativeName);
  await expect(source).toContainText('0 activities');
  await expect(source).not.toContainText('$20,000.00');
  const destination = treeRow(page, target.name);
  await expect(destination).toContainText('1 activity');
  await expect(destination).toContainText('$20,000.00');
  // 20,000 of the target's 30,000 budget: 67 % used.
  await expect(destination.getByRole('progressbar')).toHaveAttribute('aria-valuenow', '67');
  await expect(treeRow(page, records.campaignName)).toContainText('$20,000.00');
  await expect(treeRow(page, records.planName)).toContainText('$20,000.00');

  // Copy the campaign into another plan of the same year (005 US6).
  await page.getByRole('button', { name: `Actions for ${records.campaignName}` }).click();
  await page.getByRole('menuitem', { name: 'Copy' }).click();
  const copy = page.getByRole('dialog', { name: `Copy campaign '${records.campaignName}'` });
  await expect(copy.getByText('It copies 2 initiatives and 1 activity.')).toBeVisible();
  await expect(
    copy.getByText(
      'Copies get new IDs. Activities start as Created, without approvals, comments or history.',
    ),
  ).toBeVisible();
  await copy.getByRole('radio', { name: otherPlan.name, exact: true }).check();
  const copyName = `${records.campaignName} (copy)`;
  await expect(copy.getByRole('textbox', { name: /^Name of the copy/ })).toHaveValue(copyName);
  await copy.getByRole('button', { name: 'Copy campaign' }).click();
  await expectToast(page, `Campaign '${copyName}' created`);

  // The destination plan opens with the copy and its planned cost rolled up there.
  await expect(page).toHaveURL(new RegExp(`/plans/${otherPlan.id}$`));
  await expect(page.getByRole('heading', { name: otherPlan.name, level: 1 })).toBeVisible();
  await page.getByRole('button', { name: 'Expand all' }).click();
  await expect(treeRow(page, copyName)).toContainText('$20,000.00');
  await expect(treeRow(page, copyName)).toContainText('2 initiatives');
  await expect(treeRow(page, target.name)).toContainText('$20,000.00');
  await expect(treeRow(page, name)).toContainText('Created');
  await expect(treeRow(page, otherPlan.name)).toContainText('$20,000.00');
  // 20,000 of the plan's 100,000 budget: 20 % used.
  await expect(treeRow(page, otherPlan.name).getByRole('progressbar')).toHaveAttribute(
    'aria-valuenow',
    '20',
  );

  // The source plan is unchanged: still one campaign and 20,000.00.
  await page.goto(`/plans/${records.planId}`);
  await expect(treeRow(page, records.planName)).toContainText('$20,000.00');
  await expect(planTree(page).getByRole('row').filter({ hasText: copyName })).toHaveCount(0);
});

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
  await expect(page.getByRole('heading', { name: 'Your profile', level: 1 })).toBeVisible();
  await expect(
    page.getByText('Your details and teams, and how NorthStar shows dates and money to you.'),
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
  await expect(card).toContainText('Budget 1,104,000.00 EUR');
  await expect(card).toContainText(/Planned cost [\d,]+\.\d{2} EUR/);
  await expect(card).not.toContainText('$');

  // The plan page: amounts in EUR and dates as DD/MM/YYYY, in the header, the summary and the tree.
  await card.getByRole('link', { name: 'FY2026 Lubricants Europe', exact: true }).click();
  await expect(
    page.getByRole('heading', { name: 'FY2026 Lubricants Europe', level: 1 }),
  ).toBeVisible();
  await expect(
    page.getByText(/^Created by .+ on \d{2}\/\d{2}\/\d{4}, last changed by/),
  ).toBeVisible();
  await expect(page.getByRole('button', { name: '1.1M EUR (1,104,000.00 EUR)' })).toBeVisible();
  const planRow = treeRow(page, 'P-1001');
  await expect(planRow).toContainText(`01/01/${SEED_YEAR} – 31/12/${SEED_YEAR}`);
  await expect(planRow).toContainText('1,104,000.00 EUR');
  // Helix launch's 700,000.00 USD budget is 644,000.00 EUR.
  const helix = treeRow(page, 'C-1001');
  await expect(helix).toContainText(`01/01/${SEED_YEAR} – 31/12/${SEED_YEAR}`);
  await expect(helix).toContainText('644,000.00 EUR');
  await expect(planTree(page)).not.toContainText('$');
});

test('Home: the summary cards and sections for a planner and an approver, with working links (009)', async ({
  page,
}) => {
  test.slow();
  // A planner: Priya Raman, whose team is Lubricants Europe.
  await signInAs(page, PRIYA);
  await page.goto('/');
  await expect(page.getByRole('heading', { name: 'Welcome back, Priya.', level: 1 })).toBeVisible();
  await expect(
    page.getByText('Your plans, your activities and what needs you today.'),
  ).toBeVisible();
  const scope = page.getByRole('group', { name: 'Scope' });
  await expect(scope.getByRole('button', { name: 'My teams' })).toHaveAttribute(
    'aria-pressed',
    'true',
  );

  // The four summary cards are links named "<label>: <value> — <where to>" (FR-005 – FR-007).
  const summary = page.getByRole('region', { name: 'Summary' });
  const teamPlans = summary.getByRole('link', {
    name: /^My teams' plans: \d+ — open the Plans list$/,
  });
  await expect(teamPlans).toBeVisible();
  await expect(summary.getByText(/^\d+ published · \d+ draft$/)).toBeVisible();
  await expect(
    summary.getByRole('link', { name: /^Activities I own: \d+ — open your activities$/ }),
  ).toBeVisible();
  await expect(summary.getByText(/^\d+ declined · \d+ submitted$/)).toBeVisible();
  await expect(
    summary.getByRole('link', { name: /^Waiting for my approval: \d+ — open Approvals$/ }),
  ).toBeVisible();
  await expect(
    summary.getByRole('link', {
      name: /^Planned cost vs plan budget: \$[\d.]+K \(\$[\d,]+\.\d{2}\) — open the Plans list$/,
    }),
  ).toBeVisible();
  await expect(summary.getByText(/^of \$[\d.]+[KM] plan budget$/)).toBeVisible();

  // Plans at a glance: the team's plan with its status and team.
  const glance = page.getByRole('region', { name: 'Plans at a glance' });
  await expect(glance.getByText('Live roll-up across your teams')).toBeVisible();
  const glanceRow = glance.getByRole('row').filter({ hasText: 'FY2026 Lubricants Europe' });
  await expect(glanceRow).toContainText('Published');
  await expect(glanceRow).toContainText('Lubricants Europe');
  await expect(glanceRow).toContainText(/\d+ of \d+ approved/);

  // Needs your attention: Daniel Okoro declined Fleet roadshow (FR-011, FR-012).
  const attention = page.getByRole('region', { name: 'Needs your attention' });
  await expect(attention.getByText('Things you own that have a problem')).toBeVisible();
  await expect(attention.getByText(/^Declined by Daniel Okoro on /).first()).toBeVisible();

  // My activities: groups by status; Complete starts collapsed (FR-014).
  const mine = page.getByRole('region', { name: 'My activities' });
  await expect(mine.getByText('Activities you own, by status')).toBeVisible();
  await expect(mine.getByRole('button', { name: /^Declined \d+$/ })).toHaveAttribute(
    'aria-expanded',
    'true',
  );
  const complete = mine.getByRole('button', { name: /^Complete \d+$/ });
  await expect(complete).toHaveAttribute('aria-expanded', 'false');
  await expect(mine.getByRole('link', { name: 'Billboard winter wave' })).toHaveCount(0);
  await complete.click();
  await expect(complete).toHaveAttribute('aria-expanded', 'true');
  await expect(mine.getByRole('link', { name: 'Billboard winter wave' })).toBeVisible();
  await expect(mine).toContainText(`05 Jan – 28 Feb ${SEED_YEAR}`);

  // The scope switch: Everything counts every plan Priya may see — for her, the plans of her team
  // (FR-003, 014 FR-007); a Global planner's Everything is checked at the end.
  await scope.getByRole('button', { name: 'Everything' }).click();
  await expect(
    summary.getByRole('link', { name: /^All plans: \d+ — open the Plans list$/ }),
  ).toBeVisible();
  await expect(glance.getByText('Live roll-up across all plans')).toBeVisible();
  await scope.getByRole('button', { name: 'My teams' }).click();
  await expect(teamPlans).toBeVisible();

  // The currency picker redraws the amounts in the chosen currency (FR-001, FR-002).
  const currency = page.getByRole('combobox', { name: 'Currency' });
  await chooseOption(currency, 'EUR — Euro');
  await expect(
    summary.getByRole('link', { name: /^Planned cost vs plan budget: [\d.]+K EUR / }),
  ).toBeVisible();
  await expect(glanceRow).toContainText(/[\d,]+\.\d{2} EUR/);
  await chooseOption(currency, 'USD — US dollar');
  await expect(
    summary.getByRole('link', { name: /^Planned cost vs plan budget: \$/ }),
  ).toBeVisible();
  await expectNoAxeViolations(page);

  // Links: the decline opens the activity's Approval section; the cards open their lists.
  await attention
    .getByRole('link', { name: "Open approval for activity 'Fleet roadshow'" })
    .click();
  await expect(page).toHaveURL(/\/plans\/P-1001\/activities\/A-1005#approval$/);
  await expect(page.getByRole('heading', { name: 'Fleet roadshow', level: 1 })).toBeVisible();
  await expect(
    page
      .locator('#approval')
      .getByText(/^Declined by Daniel Okoro on /)
      .first(),
  ).toBeVisible();

  await page.goto('/');
  await glance.getByRole('link', { name: 'FY2026 Lubricants Europe', exact: true }).click();
  await expect(page).toHaveURL(/\/plans\/P-1001$/);
  await expect(
    page.getByRole('heading', { name: 'FY2026 Lubricants Europe', level: 1 }),
  ).toBeVisible();

  await page.goto('/');
  await teamPlans.click();
  await expect(page).toHaveURL(/\/plans\?scope=myTeams$/);
  await expect(
    page.getByRole('group', { name: 'Show' }).getByRole('button', { name: 'My teams' }),
  ).toHaveAttribute('aria-pressed', 'true');

  // "Activities I own" opens All work with the owner filter (D-174).
  await page.goto('/');
  await summary.getByRole('link', { name: /^Activities I own: / }).click();
  await expect(page).toHaveURL(/\/all-work/);
  await expect(
    page.getByRole('button', { name: 'Remove filter: Owner: Priya Raman' }),
  ).toBeVisible();

  // An approver: Daniel Okoro, with Trade magazine ads waiting for him at level 1 of 1.
  await signInAs(page, DANIEL);
  await page.goto('/');
  await expect(
    page.getByRole('heading', { name: 'Welcome back, Daniel.', level: 1 }),
  ).toBeVisible();
  await expect(
    summary.getByRole('link', { name: /^Waiting for my approval: [1-9]\d* — open Approvals$/ }),
  ).toBeVisible();
  await expect(summary.getByText(/^oldest waiting /)).toBeVisible();
  const waiting = page.getByRole('region', { name: 'Waiting for my approval' });
  const waitingRow = waiting.getByRole('listitem').filter({ hasText: 'Trade magazine ads' });
  await expect(waitingRow).toContainText('Level 1 of 1');
  await expect(waitingRow).toContainText(/waiting /);
  await expect(page.getByRole('region', { name: 'My activities' })).toContainText(
    "You don't own any activities yet.",
  );
  await expectNoAxeViolations(page);

  // Review opens Approvals with the activity's preview (SC-003).
  await waitingRow.getByRole('link', { name: 'Review Trade magazine ads' }).click();
  await expect(page).toHaveURL(/\/approvals\?tab=waiting&activity=A-1004$/);
  const preview = page.getByRole('dialog', { name: 'Trade magazine ads' });
  await expect(preview.getByText('A-1004 · Waiting at level 1 of 1')).toBeVisible();
  await preview.getByRole('button', { name: 'Close' }).click();

  await page.goto('/');
  await summary.getByRole('link', { name: /^Waiting for my approval: / }).click();
  await expect(page).toHaveURL(/\/approvals\?tab=waiting$/);
  await expect(page.getByRole('heading', { name: 'Approvals', level: 1 })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Trade magazine ads', exact: true })).toBeVisible();

  // A Global planner: Carlos Mendes sees every market, so Everything counts more plans than his teams'
  // (FR-003, 014 FR-007). Each card's value is the link's text.
  await signInAs(page, CARLOS);
  await page.goto('/');
  await expect(
    page.getByRole('heading', { name: 'Welcome back, Carlos.', level: 1 }),
  ).toBeVisible();
  const teamPlanCount = Number(await teamPlans.textContent());
  await scope.getByRole('button', { name: 'Everything' }).click();
  const allPlans = summary.getByRole('link', { name: /^All plans: \d+ — open the Plans list$/ });
  await expect(allPlans).toBeVisible();
  // Polled: the label changes at once, the count when the new scope's figures arrive.
  await expect
    .poll(async () => Number(await allPlans.textContent()))
    .toBeGreaterThan(teamPlanCount);
  await scope.getByRole('button', { name: 'My teams' }).click();
  await expect(teamPlans).toBeVisible();
});

test('All work: filter and search the grid, the calendar, budget and cost by month, a saved view and a CSV export (010)', async ({
  page,
}) => {
  test.slow();
  await signInAs(page, PRIYA);
  // The demo plans' year, named in the address (D-122), so the journey holds in any year.
  await page.goto(`/all-work?year=${SEED_YEAR}`);
  await expect(page.getByRole('heading', { name: 'All work', level: 1 })).toBeVisible();
  await expect(page.getByText('Filter, compare and export all planned work.')).toBeVisible();
  const filters = page.getByRole('group', { name: 'Filters' });
  await expect(filters.getByRole('button', { name: `Year · ${SEED_YEAR}` })).toBeVisible();
  const count = page.getByText(/^\d+ of \d+ activit(y|ies)$/);
  await expect(count).toHaveText(/^(\d+) of \1 activities$/);
  const grid = page.getByRole('table', { name: 'All work grid' });
  const gridRow = (name: string) => grid.getByRole('row').filter({ hasText: name });
  await expect(gridRow('Fleet roadshow')).toBeVisible();
  await expectNoAxeViolations(page);

  // A filter (FR-004, FR-005): only declined activities, with their plan, campaign and initiative.
  await filters.getByRole('button', { name: 'Activity status' }).click();
  await page
    .getByRole('dialog', { name: 'Activity status' })
    .getByRole('checkbox', { name: 'Declined' })
    .check();
  await page.keyboard.press('Escape');
  await expect(
    page.getByRole('button', { name: 'Remove filter: Activity status: Declined' }),
  ).toBeVisible();
  await expect(filters.getByRole('button', { name: 'Activity status · 1' })).toBeVisible();
  await expect(gridRow('Fleet roadshow')).toContainText('Declined');
  await expect(gridRow('Launch film')).toHaveCount(0);
  await expect(count).not.toHaveText(/^(\d+) of \1 activities$/);
  await page.getByRole('button', { name: 'Clear all' }).click();
  await expect(page.getByRole('button', { name: /^Remove filter: / })).toHaveCount(0);
  await expect(count).toHaveText(/^(\d+) of \1 activities$/);

  // Search (FR-006) and a filter together: one Complete activity.
  await filters.getByRole('button', { name: 'Activity status' }).click();
  await page
    .getByRole('dialog', { name: 'Activity status' })
    .getByRole('checkbox', { name: 'Complete' })
    .check();
  await page.keyboard.press('Escape');
  await page.getByRole('searchbox', { name: 'Search' }).fill('Billboard');
  await expect(count).toHaveText(/^1 of \d+ activities$/);
  const billboard = gridRow('Billboard winter wave').filter({ hasText: 'A-1003' });
  await expect(billboard).toContainText('Complete');
  await expect(billboard).toContainText(`05 Jan ${SEED_YEAR}`);
  await expect(billboard).toContainText('$45,000.00');
  await expect(billboard.locator('mark')).toHaveText('Billboard');
  await expect(grid.getByRole('row').filter({ hasText: 'Total · 1 activity' })).toContainText(
    '$45,000.00',
  );
  await expect(page.getByRole('region', { name: 'Summary' })).toContainText('$45K');

  // The calendar (FR-025 – FR-028): the year's timeline with today's line, then the month grid.
  await page.getByRole('tab', { name: 'Calendar' }).click();
  const calendar = page.getByRole('table', { name: `Calendar ${SEED_YEAR}` });
  await expect(calendar).toBeVisible();
  // The Today line is drawn only on the current year's timeline (FR-026).
  await expect(calendar.getByRole('columnheader').filter({ hasText: 'Today' })).toHaveCount(
    SEED_YEAR === THIS_YEAR ? 1 : 0,
  );
  await expect(
    calendar.getByRole('link', { name: 'Billboard winter wave Complete' }),
  ).toBeVisible();
  await expect(
    page.getByRole('tabpanel', { name: 'Calendar' }).getByText('Status', { exact: true }).first(),
  ).toBeVisible();
  await expectNoAxeViolations(page);
  await page.getByRole('button', { name: 'Month', exact: true }).click();
  await expect(page.getByRole('button', { name: 'Previous month' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Next month' })).toBeVisible();
  await expect(page.getByRole('columnheader', { name: 'Mon' })).toBeVisible();
  await expectNoAxeViolations(page);
  await page.getByRole('button', { name: 'Timeline' }).click();

  // Budget and cost by month, then by quarter (FR-022 – FR-024): Jan–Feb only, 45,000.00 in total.
  await page.getByRole('tab', { name: 'Budget & cost' }).click();
  const budget = page.getByRole('table', { name: 'Budget & cost' });
  await expect(budget.getByRole('columnheader', { name: 'Jan' })).toBeVisible();
  await expect(budget.getByRole('columnheader', { name: 'Dec' })).toBeVisible();
  await expect(page.getByText('To change how a cost is spread, open its line item')).toBeVisible();
  const footer = budget.getByRole('row').filter({ hasText: 'Total · 1 activity' });
  await expect(footer).toContainText('$45,000.00');
  await page.getByRole('button', { name: 'Quarterly' }).click();
  await expect(budget.getByRole('columnheader', { name: 'Q1' })).toBeVisible();
  const billboardBudget = budget.getByRole('row').filter({ hasText: 'Billboard winter wave' });
  // Q1, Q2, Q3, Q4, Total.
  await expect(billboardBudget.getByRole('cell')).toHaveText([
    '',
    '$45,000.00',
    '—',
    '—',
    '—',
    '$45,000.00',
  ]);

  // Save the view, clear everything, then reopen the view (FR-029 – FR-031).
  const viewName = uniqueName('E2E view');
  await page.getByRole('button', { name: 'Views' }).click();
  await page.getByRole('menuitem', { name: 'Save current view…' }).click();
  const save = page.getByRole('dialog', { name: 'Save current view' });
  await save.getByRole('textbox', { name: /^Name/ }).fill(viewName);
  await save.getByRole('button', { name: 'Save view' }).click();
  await expectToast(page, `View '${viewName}' saved`);
  await expect(page.getByRole('button', { name: `View: ${viewName}` })).toBeVisible();

  await page.getByRole('button', { name: 'Clear all' }).click();
  await page.getByRole('tab', { name: 'Grid' }).click();
  await expect(page.getByRole('searchbox', { name: 'Search' })).toHaveValue('');
  await expect(count).toHaveText(/^(\d+) of \1 activities$/);

  await page.getByRole('button', { name: `View: ${viewName}` }).click();
  await page.getByRole('menuitemcheckbox', { name: viewName }).click();
  await expect(page.getByRole('searchbox', { name: 'Search' })).toHaveValue('Billboard');
  await expect(
    page.getByRole('button', { name: 'Remove filter: Activity status: Complete' }),
  ).toBeVisible();
  await expect(page.getByRole('tab', { name: 'Budget & cost' })).toHaveAttribute(
    'aria-selected',
    'true',
  );
  await expect(count).toHaveText(/^1 of \d+ activities$/);

  // Export the grid to CSV (FR-032 – FR-034): the notes about the export, then the rows.
  await page.getByRole('tab', { name: 'Grid' }).click();
  await page.getByRole('button', { name: 'Export' }).click();
  const downloading = page.waitForEvent('download');
  await page.getByRole('menuitem', { name: 'CSV (.csv)' }).click();
  const download = await downloading;
  const fileName = download.suggestedFilename();
  expect(fileName).toMatch(new RegExp(`^NorthStar All work – Grid – ${SEED_YEAR} – .+\\.csv$`));
  await expectToast(page, `Exported '${fileName}'`);
  const csv = await readFile(await download.path(), 'utf8');
  expect(csv).toContain('View,Grid');
  expect(csv).toContain('Search,Billboard');
  expect(csv).toContain('Exported by,Priya Raman');
  expect(csv).toMatch(/^Level,Path,Name,/m);
  expect(csv).toMatch(/^Activity,.*,Billboard winter wave,,A-1003,Complete,/m);
  expect(csv).not.toContain('Fleet roadshow');
});

test('Settings: an activity type, a finance budget item, a dependent option in the preview and an option import (002 US3, US5, US6)', async ({
  page,
}) => {
  test.slow();
  await signInAs(page, ALEX);

  // Activity types (US5).
  await page.goto('/settings');
  await page.getByRole('link', { name: 'Activity types', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Activity types', level: 1 })).toBeVisible();
  await expect(page.getByText('The kinds of work activities can be.')).toBeVisible();
  await expectNoAxeViolations(page);
  const typeName = uniqueName('E2E type');
  await page.getByRole('button', { name: 'New activity type' }).click();
  const newType = page.getByRole('dialog', { name: 'New activity type' });
  await newType.getByRole('textbox', { name: /^Name/ }).fill(typeName);
  await newType.getByRole('textbox', { name: /^Default planned cost/ }).fill('12500');
  await newType.getByRole('checkbox', { name: 'Mobility 1.0' }).check();
  await newType.getByRole('button', { name: 'Save', exact: true }).click();
  await expectToast(page, `Activity type '${typeName}' created`);
  const typeRow = page.getByRole('row').filter({ hasText: typeName });
  await expect(typeRow).toContainText('$12,500.00');
  await expect(typeRow).toContainText('Mobility 1.0');
  // Not used by any activity yet; its usage names the models that allow it (002 plan, Usage parts).
  await expect(typeRow).toContainText('Allowed in 1 model');
  await expect(typeRow).not.toContainText('Used by');
  await expect(typeRow).toContainText('Active');

  // Finance budget items (US5): a new item in a new budget of this year.
  await page.goto('/settings');
  await page.getByRole('link', { name: 'Finance budget items', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Finance budget items', level: 1 })).toBeVisible();
  const budgetName = uniqueName('E2E budget');
  const itemName = uniqueName('Europe – Retail');
  const code = uniqueName('E2E').replace(' ', '-').toUpperCase();
  await page.getByRole('button', { name: 'New finance budget item' }).click();
  const newItem = page.getByRole('dialog', { name: 'New finance budget item' });
  await expect(newItem.getByRole('combobox', { name: 'Budget year' })).toHaveValue(
    String(THIS_YEAR),
  );
  await expect(
    newItem.getByText('Items with the same year and budget name form one finance budget.'),
  ).toBeVisible();
  await newItem.getByRole('combobox', { name: 'Budget name' }).fill(budgetName);
  await newItem.getByRole('textbox', { name: /^Item name/ }).fill(itemName);
  await newItem.getByRole('textbox', { name: /^Code/ }).fill(code);
  await newItem.getByRole('button', { name: 'Save', exact: true }).click();
  await expectToast(page, `Finance budget item '${itemName}' created`);
  const newBudget = page.getByRole('region', { name: budgetName });
  await expect(newBudget).toContainText('1 item');
  const itemRow = newBudget.getByRole('row').filter({ hasText: itemName });
  await expect(itemRow).toContainText(code);
  await expect(itemRow).toContainText('Not used yet');

  // A dependent option (US3): Region gets an option for the cluster Retail – India, and the preview
  // fills it in as that cluster's only region.
  await page.goto('/settings/shell-attributes?level=campaign');
  await page.getByRole('link', { name: 'Region', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Region', level: 1 })).toBeVisible();
  const regionLabel = uniqueName('RETAIL – INDIA').toUpperCase();
  await page.getByRole('button', { name: 'Add option' }).click();
  const addOption = page.getByRole('dialog', { name: 'Add option' });
  await addOption.getByRole('textbox', { name: /^Label/ }).fill(regionLabel);
  await expect(
    addOption.getByText('Shown when one of these cluster options is chosen.'),
  ).toBeVisible();
  await addOption.getByRole('checkbox', { name: 'Retail – India' }).check();
  await addOption.getByRole('button', { name: 'Save', exact: true }).click();
  await expectToast(page, `Option '${regionLabel}' added`);
  await expect(
    page.getByRole('region', { name: 'Options' }).getByRole('row').filter({ hasText: regionLabel }),
  ).toContainText('Retail – India');

  const preview = page.getByRole('region', { name: 'Preview — Campaign form' });
  await expect(
    preview.getByText('Behaves like the real form. Nothing you type here is saved.'),
  ).toBeVisible();
  await preview.getByRole('checkbox', { name: 'Retail – India' }).check();
  await expect(
    preview.getByText(`Region set to '${regionLabel}', the only region for this cluster.`),
  ).toBeVisible();
  await expect(preview.getByRole('checkbox', { name: regionLabel })).toBeChecked();

  // Import options from a small file (US6): two new rows, one already present, one twice in the file.
  await page.goto('/settings/shell-attributes');
  // The level filter is 002's SegmentedFilter: buttons named "<level> <count>".
  await page.getByRole('button', { name: /^Line item/ }).click();
  await page.getByRole('link', { name: 'Spend category', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Spend category', level: 1 })).toBeVisible();
  const first = uniqueName('Imported category A');
  const second = uniqueName('Imported category B');
  await page.getByRole('button', { name: 'Import from a spreadsheet' }).click();
  const importing = page.getByRole('dialog', { name: 'Import options into Spend category' });
  await expect(
    importing.getByText(
      'Download the template, fill one row per option, then upload it. Nothing is added until you confirm.',
    ),
  ).toBeVisible();
  const choosing = page.waitForEvent('filechooser');
  await importing.getByRole('button', { name: 'Choose a file' }).click();
  await (
    await choosing
  ).setFiles({
    name: 'spend categories.csv',
    mimeType: 'text/csv',
    buffer: Buffer.from(`Option\n${first}\nInternal\n${second}\n${first}\n`),
  });
  await expect(importing.getByText('2 options to add').first()).toBeVisible();
  await expect(importing.getByText('1 already present (skipped)').first()).toBeVisible();
  await expect(importing.getByText('1 row with problems').first()).toBeVisible();
  await expect(importing.getByText('Appears twice in the file (also row 2).')).toBeVisible();
  await importing.getByRole('button', { name: 'Add 2 options' }).click();
  await expectToast(page, '2 options added to Spend category');
  const options = page.getByRole('region', { name: 'Options' });
  await expect(options.getByRole('row').filter({ hasText: first })).toContainText('Active');
  await expect(options.getByRole('row').filter({ hasText: second })).toContainText('Active');
});
