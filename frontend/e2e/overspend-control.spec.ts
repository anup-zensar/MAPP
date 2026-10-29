import { expect, test } from '@playwright/test';
import { callApi, expectNoAxeViolations, expectToast, signInAs, uniqueName } from './fixtures';
import type { Named } from './planning';
import {
  approvedActivity,
  fillAndSavePurchaseRequest,
  openLineItem,
  openPurchaseRequest,
  PRIYA,
  purchaseRequestsSection,
  sendToSimulatedGsap,
  swbsLineItem,
} from './purchase-request-flow';

// Feature 037 on the real API and PostgreSQL: each journey prepares its own records.

test.describe.configure({ mode: 'serial' });

const UK_PLAN = 'P-1004';
const OVERSPEND = 'When planned cost goes over this budget';
const nextYear = new Date().getUTCFullYear() + 1;

test('037 US1: the owner sets Refuse, and an over-budget activity is refused before and on saving', async ({
  page,
}) => {
  test.slow();
  await signInAs(page, PRIYA);
  const campaign = await callApi<Named>(page, 'POST', `/api/v1/plans/${UK_PLAN}/campaigns`, {
    name: uniqueName('Overspend'),
    ownerId: 'U-1002',
    budget: { amount: '1000.00', currency: 'GBP' },
  });
  const initiative = await callApi<Named>(
    page,
    'POST',
    `/api/v1/campaigns/${campaign.id}/initiatives`,
    { name: 'Refused spend', ownerId: 'U-1002' },
  );

  // FR-001 – FR-003: the owner chooses Refuse on the campaign's form.
  await page.goto(`/plans/${UK_PLAN}/campaigns/${campaign.id}/edit`);
  const form = page.getByRole('dialog', { name: 'Edit campaign' });
  const setting = form.getByRole('group', { name: OVERSPEND });
  await expect(setting.getByRole('radio', { name: /^Warn/ })).toBeChecked();
  await setting.getByRole('radio', { name: /^Refuse/ }).check();
  await expectNoAxeViolations(page);
  await form.getByRole('button', { name: 'Save', exact: true }).click();
  await expectToast(page, `Campaign '${campaign.name}' saved`);
  await expect(page.getByText('Refuses overspending').first()).toBeVisible();

  // FR-014: the activity form says what is left and refuses an over-budget cost before saving.
  await page.goto(`/plans/${UK_PLAN}/initiatives/${initiative.id}/activities/new`);
  await expect(page.getByRole('heading', { name: 'New activity', level: 1 })).toBeVisible();
  await expect(
    page.getByText(
      `1,000.00 GBP left to plan in campaign '${campaign.name}', which refuses overspending.`,
    ),
  ).toBeVisible();
  await page.getByLabel('Activity name', { exact: true }).fill('Too expensive');
  const cost = page.getByLabel('Planned cost', { exact: true });
  await cost.fill('1500');
  await cost.blur();
  const over = `This is 500.00 GBP over the budget of campaign '${campaign.name}', which refuses overspending.`;
  await expect(page.getByText(over).first()).toBeVisible();
  await expectNoAxeViolations(page);

  // FR-009, FR-010: the API refuses the same save, naming the budget, its amount and the overage.
  const types = await callApi<{ activityTypes: { id: string }[] }>(
    page,
    'GET',
    `/api/v1/initiatives/${initiative.id}/activity-form`,
  );
  const typeId = types.activityTypes[0]?.id ?? '';
  const create = (amount: string) =>
    page.request.post(`/api/v1/initiatives/${initiative.id}/activities`, {
      data: {
        name: uniqueName('Spend'),
        activityTypeId: typeId,
        plannedCost: { amount, currency: 'GBP' },
      },
    });
  const refused = await create('1500.00');
  expect(refused.status()).toBe(409);
  const answer: { error: { code: string; message: string } } = await refused.json();
  expect(answer.error.code).toBe('over_budget');
  expect(answer.error.message).toContain(
    `campaign '${campaign.name}' would be 1,500.00 GBP, over its budget of 1,000.00 GBP by 500.00 GBP`,
  );
  expect((await create('900.00')).status()).toBe(201);
});

test('037 US5: a purchase request is refused while its activity’s line items exceed the planned cost', async ({
  page,
}) => {
  test.slow();
  const activity = await approvedActivity(page, 'Overspent PR');
  await signInAs(page, PRIYA);
  const lineItem = await swbsLineItem(page, activity.id, uniqueName('Extra stands'), '20000.00');
  const prName = `PR for ${lineItem.name}`;

  await openLineItem(page, lineItem);
  await purchaseRequestsSection(page)
    .getByRole('link', { name: 'New purchase request' })
    .or(purchaseRequestsSection(page).getByRole('button', { name: 'New purchase request' }))
    .first()
    .click();
  await expect(page.getByRole('heading', { name: 'New purchase request', level: 1 })).toBeVisible();
  const prId = await fillAndSavePurchaseRequest(page, `${nextYear}-03-31`);
  await sendToSimulatedGsap(page, lineItem);

  await signInAs(page, PRIYA);
  await openPurchaseRequest(page, prId, prName);
  await page.getByRole('button', { name: 'Submit for approval', exact: true }).first().click();
  const dialog = page.getByRole('dialog', { name: `'${prName}' can't be submitted yet` });
  await expect(
    dialog.getByText(
      `Activity '${activity.name}': line items exceed its planned cost by 8,000.00 USD — lower them, or raise the activity's planned cost and get it approved again.`,
    ),
  ).toBeVisible();
  await expectNoAxeViolations(page);
});
