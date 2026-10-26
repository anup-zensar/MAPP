import { expect, test } from '@playwright/test';
import {
  action,
  chooseOption,
  escapeRegExp,
  expectNoAxeViolations,
  expectToast,
  signInAs,
  uniqueName,
} from './fixtures';
import { idIn, treeRow } from './planning';

// The core flow of the Plan module (features 002–008), against the real API with the demo data.
// Tests create their own records (names from uniqueName) and never assert global counts.
// A: plan → campaign → initiative → activity → line item; axe on the main pages.

const PRIYA = 'priya.raman@shell.com';
const DANIEL = 'daniel.okoro@shell.com';
const MEI = 'mei.lin@shell.com';

test('A: a planner builds a plan down to a line item in euros and sees the cost roll up', async ({
  page,
}) => {
  test.slow();
  await signInAs(page, MEI);
  const planName = uniqueName('Core plan');
  const campaignName = uniqueName('Core campaign');
  const initiativeName = uniqueName('Core initiative');
  const activityName = uniqueName('Core activity');
  const lineItemName = uniqueName('Core line item');

  // The plan (004 US1): Mei's team and Mei as owner are the defaults; model and business unit are chosen.
  await page.goto('/plans');
  await action(page, 'New plan').click();
  await expect(page).toHaveURL(/\/plans\/new$/);
  await expect(page.getByRole('heading', { name: 'New plan', level: 1 })).toBeVisible();
  await expect(page.getByText('Default: your teams')).toBeVisible();
  await expect(page.getByText('Default: you', { exact: true })).toBeVisible();
  await page.getByRole('textbox', { name: /^Plan name/ }).fill(planName);
  // 002's ChoiceField: up to 10 options are a native select (Model) or check boxes (Business unit).
  await chooseOption(page.getByLabel(/^Model/), 'Mobility 1.0');
  await page.getByRole('checkbox', { name: 'Mobility', exact: true }).check();
  await page.getByRole('textbox', { name: /^Plan budget/ }).fill('100000');
  await page.getByRole('button', { name: 'Save', exact: true }).click();
  await expectToast(page, `Plan '${planName}' created — add campaigns next.`);
  await expect(page).toHaveURL(/\/plans\/P-\d+$/);
  const planId = idIn(page.url(), /\/plans\/(P-\d+)$/);

  // Open it again from the Plans list (004 US2).
  await page.goto('/plans');
  await page.getByRole('searchbox', { name: 'Search plans' }).fill(planName);
  await page.getByRole('link', { name: planName, exact: true }).click();
  await expect(page).toHaveURL(new RegExp(`/plans/${planId}$`));
  await expect(page.getByRole('heading', { name: planName, level: 1 })).toBeVisible();
  await expect(page.getByText('Draft', { exact: true }).first()).toBeVisible();
  await expect(page.getByText('All filled in').first()).toBeVisible();
  await expect(
    page.getByText('No campaigns yet — a campaign is a theme that runs for most of the year.'),
  ).toBeVisible();

  // A campaign (005 US1): owner and dates come from the plan; one cluster fills its only region.
  await action(page, 'Add campaign').click();
  const campaignPanel = page.getByRole('dialog', { name: 'New campaign' });
  await expect(campaignPanel.getByText(`Inside plan '${planName}'`)).toBeVisible();
  await expect(campaignPanel.getByText('From the plan')).toBeVisible();
  await expect(campaignPanel.getByText("The plan's year")).toBeVisible();
  await campaignPanel.getByRole('textbox', { name: /^Campaign name/ }).fill(campaignName);
  await campaignPanel.getByRole('textbox', { name: /^Campaign budget/ }).fill('50000');
  // Cluster (5 options) is a check-box group; Region is filled in because it has one option here,
  // and 002's auto-fill message says so with the labels in lower case.
  await campaignPanel.getByRole('checkbox', { name: 'Retail – China', exact: true }).check();
  await expect(
    campaignPanel.getByText("Region set to 'RETAIL – CHINA', the only region for this cluster."),
  ).toBeVisible();
  await campaignPanel.getByRole('button', { name: 'Save', exact: true }).click();
  await expectToast(page, `Campaign '${campaignName}' created`);
  await expect(treeRow(page, campaignName)).toBeVisible();

  // An initiative (005 US2), from the campaign row's ⋯ menu.
  await page.getByRole('button', { name: `Actions for ${campaignName}` }).click();
  await page.getByRole('menuitem', { name: 'Add initiative' }).click();
  const initiativePanel = page.getByRole('dialog', { name: 'New initiative' });
  await expect(initiativePanel.getByText(`Inside campaign '${campaignName}'`)).toBeVisible();
  await expect(initiativePanel.getByText('From the campaign')).toBeVisible();
  await initiativePanel.getByRole('textbox', { name: /^Initiative name/ }).fill(initiativeName);
  await initiativePanel.getByRole('textbox', { name: /^Initiative budget/ }).fill('40000');
  await initiativePanel.getByRole('button', { name: 'Save', exact: true }).click();
  await expectToast(page, `Initiative '${initiativeName}' created`);
  await expect(treeRow(page, initiativeName)).toBeVisible();

  // An activity (006 US1), from the initiative row's ⋯ menu (006 adds "Add activity" there).
  await page.getByRole('button', { name: `Actions for ${initiativeName}` }).click();
  await page.getByRole('menuitem', { name: 'Add activity' }).click();
  await expect(page.getByRole('heading', { name: 'New activity', level: 1 })).toBeVisible();
  // The breadcrumb names where the activity goes: plan › campaign › initiative (design system v2).
  const crumbs = page.getByRole('navigation', { name: 'Breadcrumb' });
  await expect(crumbs.getByRole('link', { name: planName, exact: true })).toBeVisible();
  await expect(crumbs.getByRole('link', { name: campaignName, exact: true })).toBeVisible();
  await expect(crumbs.getByRole('link', { name: initiativeName, exact: true })).toBeVisible();
  await expect(page.getByText('From the initiative').first()).toBeVisible();
  await page.getByRole('textbox', { name: /^Activity name/ }).fill(activityName);
  // Activity type is a native select whose options read "<type>" or "<type> · default <amount>".
  await chooseOption(page.getByLabel(/^Activity type/), 'Content production');
  await page.getByRole('textbox', { name: /^Planned cost/ }).fill('20000');
  await page.getByRole('button', { name: 'Save', exact: true }).click();
  await expectToast(page, `Activity '${activityName}' created`);
  await expect(page).toHaveURL(new RegExp(`/plans/${planId}/activities/A-\\d+$`));
  const activityId = idIn(page.url(), /\/activities\/(A-\d+)$/);
  await expect(page.getByRole('heading', { name: activityName, level: 1 })).toBeVisible();
  await expect(page.getByText('Created', { exact: true }).first()).toBeVisible();
  await expect(
    page.getByText(
      "No line items yet — split this activity's planned cost into the costs you will pay",
    ),
  ).toBeVisible();

  // A line item in euros (007 US2): the planned cost starts as the unassigned 20,000.00 USD and follows
  // the currency; 9,200.00 EUR is 10,000.00 USD at the demo rate 0.92.
  await action(page, 'Add line item').click();
  await expect(page).toHaveURL(new RegExp(`/activities/${activityId}/line-items/new$`));
  await expect(page.getByRole('heading', { name: 'New line item', level: 1 })).toBeVisible();
  // The breadcrumb names the activity the line item goes into (design system §2.3).
  await expect(
    page.getByRole('navigation', { name: 'Breadcrumb' }).getByRole('link', { name: activityName }),
  ).toBeVisible();
  await page.getByRole('textbox', { name: /^Line item name/ }).fill(lineItemName);
  // Currency (5 active currencies) is a native select with options such as "EUR – Euro".
  await chooseOption(page.getByLabel(/^Currency/), 'EUR');
  await expect(page.getByText('Changed to 18,400.00 EUR = 20,000.00 USD')).toBeVisible();
  // The line item's Planned cost is 002's MoneyField, with the currency inside.
  await page.getByRole('textbox', { name: /^Planned cost/ }).fill('9200');
  await expect(page.getByText('= 10,000.00 USD at 0.92 EUR per USD')).toBeVisible();
  await page.getByRole('button', { name: 'Save', exact: true }).click();
  await expectToast(page, `Line item '${lineItemName}' created`);
  await expect(page).toHaveURL(new RegExp(`/activities/${activityId}#line-items$`));

  // The line item's USD cost counts against the activity's planned cost (007 FR-002).
  // The Line items section's table comes before its spread grid, which repeats the name (the
  // Financials card of 015 lists the line item too, so the row is looked for in the section).
  const lineItemRow = page
    .locator('#line-items')
    .getByRole('row')
    .filter({ hasText: lineItemName })
    .first();
  await expect(lineItemRow).toContainText('9,200.00 EUR');
  // The USD column and the balance write amounts with the code, "10,000.00 USD" (007-P4).
  await expect(lineItemRow).toContainText('10,000.00 USD');
  await expect(page.getByText('10,000.00 of 20,000.00 USD assigned')).toBeVisible();
  await expect(
    page.getByText('10,000.00 USD not yet assigned to a line item').first(),
  ).toBeVisible();

  // What the activity still needs (006 FR-022 – FR-024): the approval workflow and the nine always-shown
  // needed-to-submit attributes (Mobility's own group is optional); its line item has no finance coding.
  await expect(page.getByText('Created', { exact: true }).first()).toBeVisible();
  // 10 = Approval workflow + 9 attributes; owner, dates and planned cost are filled by defaults.
  await expect(page.getByRole('button', { name: '10 fields needed' })).toBeVisible();
  // The line item lacks Spend type, GL code and Finance budget item, so the chip adds the line item.
  await expect(page.getByRole('button', { name: '1 line item needs attention' })).toBeVisible();
  await expect(
    page.getByRole('link', { name: 'Approval workflow — needed to submit' }),
  ).toBeVisible();
  await expect(lineItemRow).toContainText('3 fields needed');
  // The Ready to submit panel names the line item once before each of its problems (006 FR-023).
  for (const label of ['Spend type', 'GL code', 'Finance budget item']) {
    await expect(
      page.getByRole('link', {
        name: new RegExp(
          `^Line item '${escapeRegExp(lineItemName)}' \\(L-\\d+\\): ${label} — needed to submit$`,
        ),
      }),
    ).toBeVisible();
  }

  // The activity's planned cost rolls up to the initiative, the campaign and the plan (006 FR-005,
  // 005 FR-011, 004 FR-012); budget used = 20,000 of 40,000, 50,000 and 100,000.
  await page
    .getByRole('navigation', { name: 'Breadcrumb' })
    .getByRole('link', { name: planName })
    .click();
  await expect(page).toHaveURL(new RegExp(`/plans/${planId}$`));
  await page.getByRole('button', { name: 'Expand all' }).click();
  // Tree amounts use the full money format ("$20,000.00"), not the compact one ("$20K").
  await expect(treeRow(page, activityName)).toContainText('$20,000.00');
  await expect(treeRow(page, activityName)).toContainText('Created');
  await expect(treeRow(page, activityName)).toContainText('10 fields needed');
  await expect(treeRow(page, activityName)).toContainText('1 line item needs attention');
  await expect(treeRow(page, initiativeName)).toContainText('$20,000.00');
  await expect(treeRow(page, initiativeName).getByText('50% used', { exact: true })).toBeVisible();
  await expect(treeRow(page, campaignName)).toContainText('$20,000.00');
  await expect(treeRow(page, campaignName).getByText('40% used', { exact: true })).toBeVisible();
  // The campaign's own fields are complete (cluster and region set); something below is not. Its one
  // completeness chip names what below needs attention (design system §4.1).
  await expect(treeRow(page, campaignName)).toContainText('2 items below need attention');
  await expect(treeRow(page, campaignName)).not.toContainText('fields needed');
  await expect(treeRow(page, planName)).toContainText('$20,000.00');
  await expect(treeRow(page, planName).getByText('20% used', { exact: true })).toBeVisible();
  await expect(treeRow(page, planName)).toContainText('Draft');
  // The plan header's completeness counts the records below that need attention: the activity and
  // its line item.
  await expect(
    page.getByRole('button', { name: '2 items below need attention' }).first(),
  ).toBeVisible();
});

test('Accessibility: the Plans list, a plan page, an activity page and Approvals pass axe', async ({
  page,
}) => {
  test.slow();
  await signInAs(page, PRIYA);

  await page.goto('/plans');
  await expect(page.getByRole('heading', { name: 'Plans', level: 1 })).toBeVisible();
  await expect(
    page.getByRole('link', { name: 'FY2026 Lubricants Europe', exact: true }),
  ).toBeVisible();
  await expectNoAxeViolations(page);

  await page.goto('/plans/P-1001');
  await expect(
    page.getByRole('heading', { name: 'FY2026 Lubricants Europe', level: 1 }),
  ).toBeVisible();
  await expect(treeRow(page, 'Helix launch')).toBeVisible();
  await expectNoAxeViolations(page);

  // Launch film (A-1001) with its two seeded line items.
  await page.goto('/plans/P-1001/activities/A-1001');
  await expect(page.getByRole('heading', { name: 'Launch film', level: 1 })).toBeVisible();
  await expect(page.getByRole('row').filter({ hasText: 'Agency fee' }).first()).toBeVisible();
  await expectNoAxeViolations(page);

  // Daniel Okoro approves Lubricants Europe's activities; the rows depend on earlier test files.
  await signInAs(page, DANIEL);
  await page.goto('/approvals');
  await expect(page.getByRole('heading', { name: 'Approvals', level: 1 })).toBeVisible();
  await expect(
    page
      .getByRole('table')
      .or(page.getByText("Nothing is waiting for you. You're all caught up."))
      .first(),
  ).toBeVisible();
  await expectNoAxeViolations(page);
});
