import { expect, test } from './support/no-csp-violations';
import { action, expectNoAxeViolations, expectToast, signInAs, uniqueName } from './fixtures';
import {
  approvedActivity,
  fillAndSavePurchaseRequest,
  openLineItem,
  openPurchaseRequest,
  PRIYA,
  purchaseRequestsSection,
  swbsLineItem,
} from './purchase-request-flow';

// 038 US5 on the real API: deleting a purchase request from its page.

const nextYear = new Date().getUTCFullYear() + 1;

test('038 FR-013 (H-3): delete a purchase request from its page, then Undo', async ({ page }) => {
  test.slow();
  const activity = await approvedActivity(page, 'PR delete');
  await signInAs(page, PRIYA);
  const lineItem = await swbsLineItem(page, activity.id, uniqueName('Stand build'), '5000.00');
  const prName = `PR for ${lineItem.name}`;
  await openLineItem(page, lineItem);
  await action(purchaseRequestsSection(page), 'New purchase request').click();
  const prId = await fillAndSavePurchaseRequest(page, `${nextYear}-03-31`);
  await openPurchaseRequest(page, prId, prName);

  const notFound: string[] = [];
  page.on('response', (response) => {
    if (response.status() === 404 && response.url().includes(`/api/v1/purchase-requests/${prId}`)) {
      notFound.push(response.url());
    }
  });

  await page.getByRole('button', { name: 'More', exact: true }).click();
  await page.getByRole('menuitem', { name: 'Delete purchase request' }).click();
  const confirm = page.getByRole('alertdialog', { name: `Delete purchase request '${prName}'?` });
  await confirm.getByRole('button', { name: 'Delete purchase request' }).click();

  // Priya may not see the Purchase requests page, so she lands on the line item it was raised from.
  await expect(page).toHaveURL(new RegExp(`/line-items/${lineItem.id}$`));
  await expect(page.getByRole('heading', { name: lineItem.name, level: 1 })).toBeVisible();
  await expectToast(page, `Purchase request '${prName}' deleted`);
  await expect(page.getByText("doesn't exist or you don't have access")).toHaveCount(0);
  await expectNoAxeViolations(page);

  await page.getByRole('button', { name: 'Undo', exact: true }).click();
  await expectToast(page, `Purchase request '${prName}' restored`);
  await expect(purchaseRequestsSection(page).getByText(prName)).toBeVisible();
  expect(notFound).toEqual([]);
});
