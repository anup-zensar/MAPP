import { expect, test, type Locator, type Page } from '@playwright/test';
import { callApi, expectNoAxeViolations, signInAs } from './fixtures';

// Feature 015 US1 and US2 against the real API, reading only the demo GSAP data of FY2026 Lubricants Europe
// (backend/seed); the spec's acceptance scenarios in order.

const PRIYA = 'priya.raman@shell.com';
// The red marker of a row whose line items include Stand build, committed above its planned cost.
const ABOVE = 'Committed above planned cost on 1 line item';
// The demo data's source, or the last import's when another journey imported a file (US4).
const SOURCE_LINE = /^GSAP data as of \d{2} \w{3} \d{4}, \d{2}:\d{2} · (Demo data|.+'.+\.csv')/;

function financials(scope: Page | Locator): Locator {
  return scope.getByRole('region', { name: 'Financials', exact: true });
}

/** One of the five figures of an activity's or line item's card, such as "Invoiced". */
function figure(card: Locator, label: string): Locator {
  return card
    .getByRole('list', { name: 'Financial figures' })
    .getByRole('listitem')
    .filter({ hasText: label });
}

async function openActivity(page: Page, id: string, name: string): Promise<Locator> {
  await page.goto(`/activities/${id}`);
  await expect(page.getByRole('heading', { name, level: 1 })).toBeVisible();
  return financials(page);
}

test('015 US1 (T021): the activity page shows its GSAP money without a click, line item by line item', async ({
  page,
}) => {
  await signInAs(page, PRIYA);

  const card = await openActivity(page, 'A-1002', 'Social always-on');
  await expect(page.getByRole('tab', { name: 'Overview' })).toHaveAttribute(
    'aria-selected',
    'true',
  );
  await expect(figure(card, 'Committed (POs)')).toContainText('25,000.00 USD');
  await expect(figure(card, 'Committed (POs)')).toContainText('31% of planned cost committed');
  await expect(figure(card, 'Invoiced')).toContainText('22,500.00 USD');
  await expect(figure(card, 'Paid')).toContainText('20,000.00 USD');
  await expect(figure(card, 'Accrued (Sep 2026)')).toContainText('2,500.00 USD');
  await expect(figure(card, 'Not yet committed')).toContainText('55,000.00 USD');
  await expect(card.getByText(SOURCE_LINE)).toBeVisible();

  const byLineItem = card.getByRole('table', { name: 'Financial figures by line item' });
  await expect(
    byLineItem.getByRole('row', {
      name: 'Venue hire 20,000.00 USD 20,000.00 USD 20,000.00 USD 0.00 USD',
    }),
  ).toBeVisible();
  await expect(
    byLineItem.getByRole('row', {
      name: 'Speaker fees 5,000.00 USD 2,500.00 USD 0.00 USD 2,500.00 USD',
    }),
  ).toBeVisible();
  const latest = card.getByRole('table', { name: 'The latest transactions, newest first' });
  await expect(latest.getByRole('row')).toHaveCount(6);
  await expect(latest.getByRole('row').nth(1)).toContainText('Accrual 9100004002');
  await expect(
    card.getByRole('link', { name: 'View all 7 transactions in Financials' }),
  ).toBeVisible();

  const lineItems = page.getByRole('table', { name: 'Line items' });
  await expect(lineItems.getByRole('columnheader', { name: 'Committed (USD)' })).toBeVisible();
  await expect(lineItems.getByRole('columnheader', { name: 'Invoiced (USD)' })).toBeVisible();
  await expect(page.getByText('From finance (coming later)')).toHaveCount(0);
  await expectNoAxeViolations(page);

  const billboard = await openActivity(page, 'A-1003', 'Billboard winter wave');
  await expect(
    billboard.getByRole('row', { name: /^Stand build 10,652\.17 USD Above planned cost/ }),
  ).toBeVisible();
  await billboard
    .getByRole('link', { name: 'Committed above planned cost on 1 line item' })
    .click();
  await expect(page).toHaveURL(/\/line-items\/L-1006$/);
  await expect(page.getByRole('heading', { name: 'Stand build', level: 1 })).toBeVisible();

  const launch = await openActivity(page, 'A-1001', 'Launch film');
  await expect(
    launch.getByText(
      "No purchase orders or invoices yet — they appear here once GSAP sends them for this activity's line items.",
    ),
  ).toBeVisible();
  await expect(figure(launch, 'Committed (POs)')).toContainText('0.00 USD');
  await expect(figure(launch, 'Not yet committed')).toContainText('120,000.00 USD');

  await page.goto('/line-items/L-1004');
  await expect(page.getByRole('heading', { name: 'Speaker fees', level: 1 })).toBeVisible();
  const speaker = financials(page);
  await expect(figure(speaker, 'Committed (POs)')).toContainText('5,000.00 USD');
  await expect(figure(speaker, 'Invoiced')).toContainText('2,500.00 USD');
  await expect(figure(speaker, 'Paid')).toContainText('0.00 USD');
  await expect(figure(speaker, 'Accrued (Sep 2026)')).toContainText('2,500.00 USD');
  const transactions = speaker.getByRole('table', {
    name: "The line item's transactions, newest first",
  });
  for (const name of [
    'PO 4500001002 / 10',
    'Invoice 5100002002',
    'Accrual 9100004001',
    'Accrual 9100004002',
  ]) {
    await expect(transactions.getByRole('row', { name: new RegExp(`^${name} `) })).toBeVisible();
  }

  // Scenario 7: values from GSAP are never typed — the form has no finance figures.
  await page.goto('/line-items/L-1004/edit');
  await expect(page.getByRole('button', { name: 'Save', exact: true })).toBeVisible();
  await expect(financials(page)).toHaveCount(0);
  await expect(page.getByRole('textbox', { name: /^Committed|^Invoiced/ })).toHaveCount(0);
});

test('015 US2 (T025): the plan, campaign and initiative add up the same money, in the reader’s currency', async ({
  page,
}) => {
  await signInAs(page, PRIYA);

  await page.goto('/plans/P-1001');
  await expect(
    page.getByRole('heading', { name: 'FY2026 Lubricants Europe', level: 1 }),
  ).toBeVisible();
  const plan = financials(page);
  await expect(plan).toContainText('$35,652.17');
  await expect(plan).toContainText('8% of planned cost committed');
  await expect(plan).toContainText('$424,347.83');
  const byCampaign = plan.getByRole('table', { name: 'By campaign' });
  await expect(
    byCampaign.getByRole('row', {
      name: `Helix launch · ${ABOVE} $35,652.17 $33,152.17 $30,652.17 $2,500.00`,
    }),
  ).toBeVisible();
  await expect(
    byCampaign.getByRole('row', { name: /^Rimula fleet \$0\.00 \$0\.00 \$0\.00 \$0\.00$/ }),
  ).toBeVisible();

  await byCampaign.getByRole('link', { name: 'Helix launch' }).click();
  const campaign = financials(page.getByRole('dialog', { name: 'Helix launch' }));
  await expect(campaign).toContainText('$35,652.17');
  await expect(campaign).toContainText('13% of planned cost committed');
  const byInitiative = campaign.getByRole('table', { name: 'By initiative' });
  await expect(
    byInitiative.getByRole('row', { name: /^Digital & social \$25,000\.00 / }),
  ).toBeVisible();
  await expect(
    byInitiative.getByRole('row', {
      name: `Print & outdoor · ${ABOVE} $10,652.17 $10,652.17 $10,652.17 $0.00`,
    }),
  ).toBeVisible();

  await byInitiative.getByRole('link', { name: 'Print & outdoor' }).click();
  const initiative = financials(page.getByRole('dialog', { name: 'Print & outdoor' }));
  const byActivity = initiative.getByRole('table', { name: 'By activity' });
  await expect(
    byActivity.getByRole('row', {
      name: `Billboard winter wave · ${ABOVE} $10,652.17 $10,652.17 $10,652.17 $0.00`,
    }),
  ).toBeVisible();
  await expect(byActivity.getByRole('row', { name: /^Trade magazine ads \$0\.00 / })).toBeVisible();
  await expectNoAxeViolations(page);

  await initiative.getByRole('link', { name: 'View all 3 transactions in Financials' }).click();
  await expect(page).toHaveURL(/\/financials\?recordId=I-1002$/);
  await expect(page.getByRole('heading', { name: 'Financials', level: 1 })).toBeVisible();
  await expect(page.getByRole('tab', { name: 'All (3)' })).toBeVisible();
  await expect(page.getByText('3 of 3 transactions')).toBeVisible();

  // Scenario 4: a reader whose display currency is EUR sees every figure in EUR (USD 1 = EUR 0.92).
  await callApi(page, 'PATCH', '/api/v1/me/preferences', { displayCurrency: 'EUR' });
  await page.goto('/plans/P-1001');
  await expect(plan).toContainText('32,800.00 EUR');
  await expect(plan).toContainText('390,400.00 EUR');
  await expect(plan).not.toContainText('$');
});
