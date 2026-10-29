import type { Page } from '@playwright/test';
import { expect, test } from './support/no-csp-violations';
import { callApi, expectNoAxeViolations, expectToast, signInAs, uniqueName } from './fixtures';
import { lubricantsActivity } from './lubricants';
import { decideAs } from './planning-changes';
import {
  approveFromInbox,
  DANIEL,
  fillAndSavePurchaseRequest,
  openLineItem,
  openPurchaseRequest,
  PRIYA,
  purchaseRequestsSection,
  sendToSimulatedGsap,
  submitOnScreen,
  swbsLineItem,
} from './purchase-request-flow';

// 038 US1 on the real API: lowering an approved purchase request keeps the approved amount reserved
// until the change is approved (FR-001 – FR-003, C-1).

const nextYear = new Date().getUTCFullYear() + 1;

async function approvedActivityOf(page: Page, amount: string) {
  await signInAs(page, PRIYA);
  const activity = await lubricantsActivity(page, 'PR reservation', amount);
  await callApi(page, 'POST', `/api/v1/activities/${activity.id}/approval/submit`, {
    version: activity.version,
    comment: '',
  });
  await decideAs(page, DANIEL, activity.id, 'approve');
  return activity;
}

function amountField(page: Page) {
  return page.locator('[id="lines[0].amount"]');
}

test('038 FR-001 (C-1): a change in review keeps the approved amount reserved', async ({
  page,
}) => {
  test.slow();
  const activity = await approvedActivityOf(page, '20000.00');
  await signInAs(page, PRIYA);
  const lineItem = await swbsLineItem(page, activity.id, uniqueName('Stand build'), '15800.00');
  const prName = `PR for ${lineItem.name}`;
  await sendToSimulatedGsap(page, lineItem);

  // An approved purchase request for all 15,800.00 USD of the line item.
  await signInAs(page, PRIYA);
  await page.goto(`/purchase-requests/new?lineItemId=${lineItem.id}`);
  await expect(amountField(page)).toHaveValue('15,800.00');
  const prId = await fillAndSavePurchaseRequest(page, `${nextYear}-03-31`);
  await submitOnScreen(page, prName);
  await approveFromInbox(page, prName);

  // Priya lowers it to 12,000.00 USD and submits the change (FR-028 confirmation first).
  await signInAs(page, PRIYA);
  await page.goto(`/purchase-requests/${prId}/edit`);
  await amountField(page).fill('12000.00');
  await page.getByRole('button', { name: 'Submit changes', exact: true }).click();
  const confirm = page.getByRole('alertdialog', { name: `Submit changes to '${prName}'?` });
  await expect(confirm.getByText(/stay reserved until the change is approved/)).toBeVisible();
  await expectNoAxeViolations(page);
  await confirm.getByRole('button', { name: 'Submit changes', exact: true }).click();
  await expect(page).toHaveURL(new RegExp(`/purchase-requests/${prId}$`));

  // While the change waits, 15,800.00 USD stay reserved: nothing is left for a second request (032 rule).
  const defaults = await page.request.get(
    `/api/v1/purchase-requests/new-defaults?lineItemId=${lineItem.id}`,
  );
  expect(defaults.status()).toBe(400);
  expect(await defaults.text()).toContain('Nothing left to request');

  // Daniel declines the change: the approved 15,800.00 USD stay reserved, nothing is available.
  await signInAs(page, DANIEL);
  await openPurchaseRequest(page, prId, prName);
  const approval = page.getByRole('region', { name: 'Approval' });
  await approval.getByRole('button', { name: 'Decline', exact: true }).click();
  const decline = page.getByRole('alertdialog');
  await decline.getByRole('textbox', { name: /Why are you declining it/ }).fill('Keep the stand.');
  await decline.getByRole('button', { name: 'Decline', exact: true }).click();
  await expectToast(page, /declined/);

  await openLineItem(page, lineItem);
  const section = purchaseRequestsSection(page);
  await expect(section.getByText('Available to request')).toBeVisible();
  await expect(section.getByText('0.00 USD').first()).toBeVisible();
  await expectNoAxeViolations(page);
});
