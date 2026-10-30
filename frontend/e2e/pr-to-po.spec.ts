import { expect, test } from '@playwright/test';
import { action, expectNoAxeViolations, expectToast, signInAs, uniqueName } from './fixtures';
import {
  ALEX,
  approveFromInbox,
  approvedActivity,
  fillAndSavePurchaseRequest,
  LEA,
  openFromWaitingForGsap,
  openLineItem,
  PRIYA,
  purchaseRequestsSection,
  sendToSimulatedGsap,
  submitOnScreen,
} from './purchase-request-flow';

// The journey: approved activity → SWBS code → PR approved → simulated GSAP PR and PO → Committed.

const SIM_NUMBER = /SIM-\d{10}/;

test('033 US1: from an approved activity to the purchase order and Committed figures', async ({
  page,
}) => {
  test.slow();
  const activity = await approvedActivity(page, 'PR to PO');

  // Priya adds the line item on screen, asking GSAP for its SWBS code (D-322: approval is kept).
  await signInAs(page, PRIYA);
  await page.goto(activity.path);
  await expect(page.getByRole('heading', { name: activity.name, level: 1 })).toBeVisible();
  await action(page, 'Add line item').click();
  await expect(page.getByRole('heading', { name: 'New line item', level: 1 })).toBeVisible();
  const lineItemName = uniqueName('Gift boxes');
  await page.getByRole('textbox', { name: /^Line item name/ }).fill(lineItemName);
  await page.getByRole('textbox', { name: /^Planned cost/ }).fill('10000');
  await page.getByRole('checkbox', { name: 'Request SWBS code from GSAP' }).check();
  await page.getByRole('button', { name: 'Save', exact: true }).click();
  await expectToast(page, `Line item '${lineItemName}' created`);
  await page.locator('#line-items').getByRole('link', { name: lineItemName }).first().click();
  await expect(page.getByRole('heading', { name: lineItemName, level: 1 })).toBeVisible();
  const lineItem = { id: /L-\d+/.exec(page.url())?.[0] ?? '', name: lineItemName };
  expect(lineItem.id).not.toBe('');

  await sendToSimulatedGsap(page, lineItem);

  await signInAs(page, PRIYA);
  await openLineItem(page, lineItem);
  await action(purchaseRequestsSection(page), 'New purchase request').click();
  await expect(page.getByRole('heading', { name: 'New purchase request', level: 1 })).toBeVisible();
  const prName = `PR for ${lineItemName}`;
  const prId = await fillAndSavePurchaseRequest(page, `${new Date().getUTCFullYear() + 1}-04-30`);
  await submitOnScreen(page, prName);

  await approveFromInbox(page, prName);

  // Lea, the requisitioner, works from Waiting for GSAP: the PR form, then the simulated GSAP.
  await openFromWaitingForGsap(page, prId, prName);
  const download = page.waitForEvent('download');
  await action(page, 'Download PR form').click();
  expect((await download).suggestedFilename()).toContain(prId);

  await page.getByRole('button', { name: 'Create GSAP PR and PO (simulated)' }).click();
  const confirm = page.getByRole('alertdialog', { name: 'Create GSAP PR and PO (simulated)?' });
  await expect(confirm).toContainText(
    `The simulated GSAP will create a GSAP PR and a purchase order with 1 line (10,000.00 USD) for ${prId}.`,
  );
  await expectNoAxeViolations(page);
  await confirm.getByRole('button', { name: 'Create', exact: true }).click();

  // GSAP's answer on the PR, read-only and labelled Simulated GSAP (033 FR-002 – FR-004, FR-011).
  const gsap = page.getByRole('region', { name: 'GSAP data' });
  await expect(gsap.getByText(SIM_NUMBER).first()).toBeVisible();
  await expect(gsap.getByText('Simulated GSAP').first()).toBeVisible();
  const lines = page.getByRole('region', { name: 'Lines' });
  await expect(lines.getByText('Released')).toBeVisible();
  const poNumber =
    (
      await lines
        .getByText(/SIM-45\d{8}/)
        .first()
        .textContent()
    )?.match(/SIM-\d{10}/)?.[0] ?? '';
  expect(poNumber).toMatch(SIM_NUMBER);
  await expect(page.getByRole('button', { name: 'Create GSAP PR and PO (simulated)' })).toHaveCount(
    0,
  );
  await expectNoAxeViolations(page);

  await page.goto('/purchase-requests?waiting=true');
  await page.getByRole('searchbox', { name: 'Search purchase requests' }).fill(prId);
  await expect(page.getByText('Nothing matches these filters')).toBeVisible();
  await page.goto(`/purchase-requests?q=${poNumber}`);
  await expect(
    page.getByRole('region', { name: 'Purchase requests by vendor' }).getByRole('link', {
      name: prName,
      exact: true,
    }),
  ).toBeVisible();

  // The PO line is a GSAP purchase-order transaction: Committed follows (033 FR-005, SC-001).
  await signInAs(page, ALEX);
  await openLineItem(page, lineItem);
  const figures = page.getByRole('list', { name: 'Financial figures' });
  await expect(figures.getByText('10,000.00 USD').first()).toBeVisible();
  await page.goto(`/financials?year=${new Date().getUTCFullYear() + 1}`);
  await page.getByRole('searchbox', { name: 'Search transactions' }).fill(poNumber);
  await expect(page.getByRole('link', { name: new RegExp(poNumber) }).first()).toBeVisible();
  await expectNoAxeViolations(page);

  await signInAs(page, LEA);
  await page.goto(`/purchase-requests/${prId}`);
  await expect(page.getByRole('region', { name: 'Lines' }).getByText(poNumber)).toBeVisible();
});
