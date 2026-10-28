import { expect, test } from '@playwright/test';
import { action, expectNoAxeViolations, expectToast, signInAs, uniqueName } from './fixtures';
import {
  approveFromInbox,
  approvedActivity,
  fillAndSavePurchaseRequest,
  LEA,
  openFromWaitingForGsap,
  openLineItem,
  openPurchaseRequest,
  PRIYA,
  purchaseRequestsSection,
  sendToSimulatedGsap,
  submitOnScreen,
  swbsLineItem,
} from './purchase-request-flow';

// Feature 032 on the real API; the journey prepares its own activity and line item.

test.describe.configure({ mode: 'serial' });

const nextYear = new Date().getUTCFullYear() + 1;

test('032 US1–US3: raise, submit, approve and route a purchase request to the requisitioner', async ({
  page,
}) => {
  test.slow();
  const activity = await approvedActivity(page, 'PR journey');
  await signInAs(page, PRIYA);
  const lineItem = await swbsLineItem(page, activity.id, uniqueName('Gift boxes'), '10000.00');
  const prName = `PR for ${lineItem.name}`;
  const hxKey = `HX${lineItem.id.replace('L-', '')}`;

  // US1: the line item's Purchase requests section, then New purchase request, pre-filled (FR-001, FR-003).
  await openLineItem(page, lineItem);
  const section = purchaseRequestsSection(page);
  await expect(section.getByText('No purchase requests on this line item yet.')).toBeVisible();
  await expectNoAxeViolations(page);
  await action(section, 'New purchase request').click();
  await expect(page.getByRole('heading', { name: 'New purchase request', level: 1 })).toBeVisible();
  await expect(page.getByRole('textbox', { name: 'Name', exact: true })).toHaveValue(prName);
  await expect(page.getByText(hxKey).first()).toBeVisible();
  await expect(page.getByText('Generated from the line-item ID').first()).toBeVisible();

  await page.getByRole('button', { name: 'Save', exact: true }).click();
  await expect(
    page.getByRole('heading', { name: /problems? needs? your attention/ }),
  ).toBeVisible();
  await expect(page).toHaveURL(/\/purchase-requests\/new/);

  const prId = await fillAndSavePurchaseRequest(page, `${nextYear}-03-31`);
  await expectToast(page, `Purchase request '${prName}' created`);
  await expect(page.getByText('Created', { exact: true }).first()).toBeVisible();
  await expect(page.getByRole('region', { name: 'Lines' }).getByText(hxKey).first()).toBeVisible();
  await expect(
    page
      .getByRole('region', { name: 'Routing history' })
      .getByText('The PR form is created when the purchase request is approved.'),
  ).toBeVisible();
  await expectNoAxeViolations(page);

  // The money is reserved on the line item at once (FR-017, FR-018).
  await openLineItem(page, lineItem);
  await expect(section.getByText(prName)).toBeVisible();
  await expect(section.getByText('10,000.00 USD').first()).toBeVisible();
  await expect(section.getByText('0.00 USD').first()).toBeVisible();
  await expect(action(section, 'New purchase request')).toHaveCount(0);

  // US2: submitting is refused while the line item has no SWBS code (FR-021).
  await openPurchaseRequest(page, prId, prName);
  await page.getByRole('button', { name: 'Submit for approval', exact: true }).first().click();
  const refused = page.getByRole('dialog', { name: `'${prName}' can't be submitted yet` });
  await expect(refused.getByText(/has no SWBS code yet \(Ready to send to GSAP\)/)).toBeVisible();
  await expectNoAxeViolations(page);
  await refused.getByRole('button', { name: 'Close', exact: true }).click();

  await sendToSimulatedGsap(page, lineItem);
  await signInAs(page, PRIYA);
  await openPurchaseRequest(page, prId, prName);
  await submitOnScreen(page, prName);
  await expect(page.getByText('Submitted', { exact: true }).first()).toBeVisible();

  await approveFromInbox(page, prName);

  // US3: approval routes the PR to the requisitioner queue (FR-030, FR-032); Lea downloads the PR form.
  await openFromWaitingForGsap(page, prId, prName);
  await expect(page.getByText('Approved', { exact: true }).first()).toBeVisible();
  const routings = page.getByRole('region', { name: 'Routing history' });
  await expect(routings.getByText('Original · routed to DE_LUBES')).toBeVisible();
  const download = page.waitForEvent('download');
  await action(page, 'Download PR form').click();
  expect((await download).suggestedFilename()).toContain(prId);
  await expect(action(page, 'Edit')).toHaveCount(0);
  await expect(
    page.getByRole('button', { name: 'Submit for approval', disabled: false }),
  ).toHaveCount(0);
  await expectNoAxeViolations(page);

  // Nothing a PR draws on can be deleted (FR-019).
  await signInAs(page, PRIYA);
  await openLineItem(page, lineItem);
  await page.getByRole('button', { name: `Actions for ${lineItem.name}` }).click();
  await page.getByRole('menuitem', { name: 'Delete' }).click();
  await page.getByRole('alertdialog').getByRole('button', { name: 'Delete line item' }).click();
  await expect(
    page
      .getByText(new RegExp(`Can't delete — purchase request ${prId} draws on line item`))
      .first(),
  ).toBeVisible();
});

test('032 US5: a person without the Purchase requests page permission has no page', async ({
  page,
}) => {
  await signInAs(page, PRIYA);
  await page.goto('/purchase-requests');
  await expect(page.getByRole('heading', { name: 'No access', level: 1 })).toBeVisible();
  await expect(
    page.getByRole('navigation', { name: 'Main' }).getByRole('link', { name: 'Purchase requests' }),
  ).toHaveCount(0);

  await signInAs(page, LEA);
  await page.goto('/');
  await expect(
    page.getByRole('navigation', { name: 'Main' }).getByRole('link', { name: 'Purchase requests' }),
  ).toBeVisible();
});
