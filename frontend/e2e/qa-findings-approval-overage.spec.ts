import type { Page } from '@playwright/test';
import { expect, test } from './support/no-csp-violations';
import { action, expectNoAxeViolations, expectToast, signInAs, uniqueName } from './fixtures';
import {
  ALEX,
  approvedActivity,
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

// 038 US7 on the real API: approving while an activity's line items exceed its planned cost — Warn
// (default) shows the overage, Refuse disables Approve with the reason (D-360).

test.describe.configure({ mode: 'serial' });

const APPROVAL =
  "When an activity's line items exceed its planned cost, approving it or its purchase requests";
const nextYear = new Date().getUTCFullYear() + 1;

async function setApprovalMode(page: Page, choice: 'Warn the approver' | 'Refuse approval') {
  await signInAs(page, ALEX);
  await page.goto('/settings/overspend-control');
  const group = page.getByRole('group', { name: APPROVAL });
  await group.getByRole('radio', { name: new RegExp(`^${choice}`) }).check();
  await expectNoAxeViolations(page);
  await page.getByRole('button', { name: 'Save', exact: true }).click();
  await expectToast(page, 'Overspend control saved.');
}

test('038 FR-019, FR-020 (M-18): warn, then refuse approving a purchase request', async ({
  page,
}) => {
  test.slow();
  const activity = await approvedActivity(page, 'Approval overage');
  await signInAs(page, PRIYA);
  const lineItem = await swbsLineItem(page, activity.id, uniqueName('Stand build'), '10000.00');
  const prName = `PR for ${lineItem.name}`;
  await sendToSimulatedGsap(page, lineItem);
  await signInAs(page, PRIYA);
  await openLineItem(page, lineItem);
  await action(purchaseRequestsSection(page), 'New purchase request').click();
  const prId = await fillAndSavePurchaseRequest(page, `${nextYear}-03-31`);
  await submitOnScreen(page, prName);
  // Line items now 15,000.00 USD against 12,000.00 USD planned: 3,000.00 USD above.
  await swbsLineItem(page, activity.id, uniqueName('Extra stand'), '5000.00');
  const warning = `Line items of activity '${activity.name}' are 3,000.00 USD above its planned cost`;

  try {
    // Warn (the default): the section and the Approve dialog show the overage.
    await signInAs(page, DANIEL);
    await openPurchaseRequest(page, prId, prName);
    const approval = page.getByRole('region', { name: 'Approval' });
    await expect(approval.getByText(warning)).toBeVisible();
    await expectNoAxeViolations(page);
    await approval.getByRole('button', { name: 'Approve', exact: true }).click();
    const approve = page.getByRole('alertdialog', {
      name: `Approve purchase request '${prName}'?`,
    });
    await expect(approve.getByText(warning)).toBeVisible();
    await approve.getByRole('button', { name: 'Cancel', exact: true }).click();

    // Refuse: Approve is shown disabled with the reason; Decline still works.
    await setApprovalMode(page, 'Refuse approval');
    await signInAs(page, DANIEL);
    await openPurchaseRequest(page, prId, prName);
    const refusal = `${warning} — ask the owner to lower them, or decline.`;
    const disabled = approval.getByRole('button', { name: 'Approve', exact: true });
    await expect(disabled).toBeDisabled();
    await expect(disabled).toHaveAccessibleDescription(refusal);
    await expectNoAxeViolations(page);
    await approval.getByRole('button', { name: 'Decline', exact: true }).click();
    const decline = page.getByRole('alertdialog', {
      name: `Decline purchase request '${prName}'?`,
    });
    await decline.getByRole('textbox', { name: /Why are you declining it/ }).fill('Over cost.');
    await decline.getByRole('button', { name: 'Decline', exact: true }).click();
    await expectToast(page, /declined/);
  } finally {
    await setApprovalMode(page, 'Warn the approver');
  }
});
