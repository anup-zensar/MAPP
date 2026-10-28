import { expect, type Page } from '@playwright/test';
import { callApi, chooseOption, expectNoAxeViolations, expectToast, signInAs } from './fixtures';
import { lubricantsActivity } from './lubricants';
import { decideAs } from './planning-changes';
import type { Named } from './planning';

// Steps the purchase-request journeys share; preparation goes through the API, the rest on screen.

export const PRIYA = 'priya.raman@shell.com';
export const DANIEL = 'daniel.okoro@shell.com';
export const ALEX = 'alex.morgan@shell.com';
export const LEA = 'lea.santos@shell.com';

export interface ApprovedActivity extends Named {
  path: string;
}

/** Priya's 12,000.00 USD Lubricants activity, approved by Daniel Okoro; ends signed in as Daniel. */
export async function approvedActivity(page: Page, label: string): Promise<ApprovedActivity> {
  await signInAs(page, PRIYA);
  const activity = await lubricantsActivity(page, label, '12000.00');
  await callApi(page, 'POST', `/api/v1/activities/${activity.id}/approval/submit`, {
    version: activity.version,
    comment: '',
  });
  await decideAs(page, DANIEL, activity.id, 'approve');
  return { id: activity.id, name: activity.name, path: activity.path };
}

export async function swbsLineItem(
  page: Page,
  activityId: string,
  name: string,
  plannedCost: string,
): Promise<Named> {
  return callApi<Named>(page, 'POST', `/api/v1/activities/${activityId}/line-items`, {
    name,
    ownerId: null,
    description: '',
    currency: 'USD',
    plannedCost,
    spread: { method: 'evenly', months: [] },
    attributes: {},
    contractedAmount: null,
    requestSwbs: true,
    financeBudgetYear: null,
    financeBudgetItemId: null,
  });
}

export async function openLineItem(page: Page, lineItem: Named): Promise<void> {
  await page.goto(`/line-items/${lineItem.id}`);
  await expect(page.getByRole('heading', { name: lineItem.name, level: 1 })).toBeVisible();
}

export function purchaseRequestsSection(page: Page) {
  return page.locator('#purchase-requests');
}

/** As Alex Morgan, who imports GSAP extracts: the line item's SWBS code from the simulated GSAP (030). */
export async function sendToSimulatedGsap(page: Page, lineItem: Named): Promise<void> {
  await signInAs(page, ALEX);
  await openLineItem(page, lineItem);
  await page.getByRole('button', { name: 'Send to simulated GSAP now' }).click();
  const confirm = page.getByRole('alertdialog', {
    name: `Send ${lineItem.name} to simulated GSAP now?`,
  });
  await confirm.getByRole('button', { name: 'Send to simulated GSAP now' }).click();
  await expect(
    page.getByRole('region', { name: 'SWBS request' }).getByText('SWBS code received'),
  ).toBeVisible();
}

/** Fills what the pre-filled PR form lacks to be submitted (Email group DE_LUBES) and saves; the PR's ID. */
export async function fillAndSavePurchaseRequest(
  page: Page,
  deliveryDate: string,
): Promise<string> {
  const moa = page.getByRole('combobox', { name: /^MoA approver/ });
  await moa.fill('Daniel');
  await page
    .getByRole('listbox', { name: 'MoA approver' })
    .getByRole('option', { name: /^Daniel Okoro/ })
    .click();
  await chooseOption(
    page.getByRole('combobox', { name: /^Approval workflow(?! for)/ }),
    'Purchase request approval (MoA)',
  );
  await chooseOption(
    page.getByRole('combobox', { name: /^Approval workflow for modification/ }),
    'Purchase request change approval (MoA)',
  );
  await page
    .getByRole('group', { name: /^Material or service/ })
    .getByLabel('Service', { exact: true })
    .check();
  await page
    .getByRole('group', { name: /^PR type/ })
    .getByLabel('Standard PR/PO', { exact: true })
    .check();
  await page.getByRole('textbox', { name: 'Short text', exact: true }).fill('Christmas gift boxes');
  await page
    .getByRole('textbox', { name: 'Invoice approver name', exact: true })
    .fill('Priya Raman');
  await page.getByRole('textbox', { name: 'Vendor', exact: true }).fill('Northlight Events GmbH');
  await page.getByRole('textbox', { name: 'Vendor code', exact: true }).fill('V-100101');
  await chooseOption(page.getByRole('combobox', { name: /^Email group/ }), 'DE_LUBES');
  await page.getByLabel(/^Delivery date/).fill(deliveryDate);
  await expectNoAxeViolations(page);
  await page.getByRole('button', { name: 'Save', exact: true }).click();

  await expect(page).toHaveURL(/\/purchase-requests\/PR-\d+$/);
  const id = /PR-\d+/.exec(page.url())?.[0] ?? '';
  expect(id).not.toBe('');
  return id;
}

export async function submitOnScreen(page: Page, name: string): Promise<void> {
  await page.getByRole('button', { name: 'Submit for approval', exact: true }).first().click();
  const dialog = page.getByRole('dialog', { name: `Submit '${name}' for approval?` });
  await expect(
    dialog.getByText('Approval workflow: Purchase request approval (MoA)'),
  ).toBeVisible();
  await expectNoAxeViolations(page);
  await dialog.getByRole('button', { name: 'Submit for approval', exact: true }).click();
  await expectToast(page, `Purchase request '${name}' submitted for approval`);
}

/** As Daniel Okoro, the MoA approver: approves the PR from the Approvals page's preview (032 FR-029). */
export async function approveFromInbox(page: Page, name: string): Promise<void> {
  await signInAs(page, DANIEL);
  await page.goto('/approvals?tab=waiting');
  await expect(page.getByRole('heading', { name: 'Approvals', level: 1 })).toBeVisible();
  await page.getByRole('searchbox', { name: 'Search activity name or ID' }).fill(name);
  await page.getByRole('button', { name, exact: true }).click();
  const preview = page.getByRole('dialog', { name });
  await expect(preview.getByText('Northlight Events GmbH').first()).toBeVisible();
  await expectNoAxeViolations(page);
  await preview.getByRole('button', { name: 'Approve', exact: true }).click();
  const confirm = page.getByRole('alertdialog', { name: `Approve purchase request '${name}'?` });
  await confirm.getByRole('button', { name: 'Approve', exact: true }).click();
  await expectToast(page, `Purchase request '${name}' approved`);
}

export async function openPurchaseRequest(page: Page, id: string, name: string): Promise<void> {
  await page.goto(`/purchase-requests/${id}`);
  await expect(page.getByRole('heading', { name, level: 1 })).toBeVisible();
}

/** As Lea Santos: Purchase requests › Waiting for GSAP lists the PR; opens it from there. */
export async function openFromWaitingForGsap(page: Page, id: string, name: string): Promise<void> {
  await signInAs(page, LEA);
  await page.goto('/purchase-requests');
  await expect(page.getByRole('heading', { name: 'Purchase requests', level: 1 })).toBeVisible();
  await page.getByRole('button', { name: 'Waiting for GSAP', exact: true }).click();
  await expect(page).toHaveURL(/waiting=true/);
  await page.getByRole('searchbox', { name: 'Search purchase requests' }).fill(id);
  const table = page.getByRole('region', { name: 'Purchase requests by vendor' });
  await expect(table.getByRole('link', { name, exact: true })).toBeVisible();
  await expectNoAxeViolations(page);
  await table.getByRole('link', { name, exact: true }).click();
  await expect(page.getByRole('heading', { name, level: 1 })).toBeVisible();
}
