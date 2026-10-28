import { expect, test } from '@playwright/test';
import { callApi, expectNoAxeViolations, signInAs } from './fixtures';

// 013 SP&A view and BCPIR marker: US1 (the SP&A view of a plan and its campaigns) and US2 (BCPIR plans).

const PRIYA = 'priya.raman@shell.com';
const PLAN = 'FY2026 Lubricants Europe';

function usd(amount: string): string {
  return Number(amount).toLocaleString('en-US', { style: 'currency', currency: 'USD' });
}

test('SP&A view of a plan: figures traced to activities, periods, dimension, campaign scope and a purchase request (013 US1)', async ({
  page,
}) => {
  test.slow();
  await signInAs(page, PRIYA);
  const card = await callApi<{ figures: { committed: { amount: string } } }>(
    page,
    'GET',
    '/api/v1/financials/records/P-1001',
  );
  const committed = usd(card.figures.committed.amount);

  await page.goto('/plans/P-1001');
  await page.getByRole('link', { name: 'SP&A view' }).click();
  await expect(page.getByRole('heading', { name: 'SP&A view', level: 1 })).toBeVisible();
  await expect(page.getByRole('group', { name: 'Plan facts' })).toContainText('BCPIR');
  const headline = page.getByRole('region', { name: 'Headline figures' });
  await expect(headline).toBeVisible();
  await expect(page.getByText(/^Finance data as of /)).toBeVisible();
  await expectNoAxeViolations(page);

  // SC-003 and FR-007: Committed is the plan page's, and its activities add up to it.
  await headline
    .getByRole('button', { name: `${committed}, show what makes up Committed` })
    .click();
  const trace = page.getByRole('dialog', { name: 'What makes up Committed' });
  await expect(trace.getByRole('row', { name: /Total/ })).toContainText(committed);
  await expectNoAxeViolations(page);
  await page.keyboard.press('Escape');

  // FR-017, FR-018: months, and another dimension, kept in the address.
  await page.getByRole('button', { name: 'Months' }).click();
  await expect(page.getByRole('rowheader', { name: 'Mar 2026' })).toBeVisible();
  await page.getByRole('combobox', { name: 'Group by' }).selectOption({ label: 'Activity type' });
  await expect(page).toHaveURL(/dimension=activityType/);
  await expect(
    page.getByRole('columnheader', { name: 'Activity type', exact: true }).first(),
  ).toBeVisible();

  // FR-005: a campaign's view; Back returns to the whole plan.
  await page.getByRole('combobox', { name: 'Scope' }).selectOption({ label: 'Helix launch' });
  await expect(page).toHaveURL(/campaign=C-1001/);
  const activities = page.getByRole('table', { name: 'Activities' });
  await expect(activities.getByRole('link', { name: 'Fleet roadshow' })).toHaveCount(0);
  await page.goBack();
  await expect(page.getByRole('combobox', { name: 'Scope' })).toHaveValue('');
  await expect(activities.getByRole('link', { name: 'Fleet roadshow' })).toBeVisible();

  // FR-019: sort and open an activity's line items.
  await activities.getByRole('button', { name: 'Actual spend' }).click();
  await expect(activities.getByRole('row').nth(1)).toContainText('Social always-on');
  await activities.getByRole('button', { name: 'Show the line items of Social always-on' }).click();
  await expect(activities.getByRole('link', { name: 'Venue hire' })).toBeVisible();
  await expectNoAxeViolations(page);

  // FR-022: Raise purchase request opens 032's form for the chosen line item.
  await page.getByRole('button', { name: 'Actions for Social always-on' }).click();
  await page.getByRole('menuitem', { name: 'Raise purchase request for Venue hire' }).click();
  await expect(page).toHaveURL(/\/purchase-requests\/new\?lineItemId=L-1003/);
  await expect(page.getByRole('heading', { name: 'New purchase request', level: 1 })).toBeVisible();
});

test('BCPIR plans: the marker and the filter on the Plans list and in All work (013 US2)', async ({
  page,
}) => {
  await signInAs(page, PRIYA);
  await page.goto('/plans?bcpir=only&layout=table');
  const table = page.getByRole('table');
  await expect(table.getByRole('row').filter({ hasText: PLAN })).toContainText('BCPIR');
  await expect(table.getByRole('row').filter({ hasText: 'FY2026 Mobility China' })).toHaveCount(0);
  await expect(page.getByText('BCPIR plans: Only')).toBeVisible();
  await expectNoAxeViolations(page);

  await page.goto('/all-work?year=2026&bcpir=only');
  const grid = page.getByRole('table', { name: 'All work grid' });
  await expect(grid.getByRole('row').filter({ hasText: PLAN }).first()).toContainText('BCPIR');
  await expect(grid.getByRole('row').filter({ hasText: 'FY2026 Mobility China' })).toHaveCount(0);
  await expect(page.getByText('BCPIR plans: Only')).toBeVisible();
  await expectNoAxeViolations(page);
});
