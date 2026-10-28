import { expect, test, type Page } from '@playwright/test';
import { expectNoAxeViolations, expectToast, signInAs } from './fixtures';

const ALEX = 'alex.morgan@shell.com';
const DANIEL = 'daniel.okoro@shell.com';
const PRIYA = 'priya.raman@shell.com';

test.describe.configure({ mode: 'serial' });
test.use({ permissions: ['clipboard-read', 'clipboard-write'] });

function swbsCard(page: Page) {
  return page.getByRole('region', { name: 'SWBS request' });
}

async function openLineItem(page: Page, id: string, name: string) {
  await page.goto(`/line-items/${id}`);
  await expect(page.getByRole('heading', { name, level: 1 })).toBeVisible();
}

test('030 US2, US5: a marketer reads the SWBS request, its code and the HX key', async ({
  page,
}) => {
  await signInAs(page, PRIYA);

  await openLineItem(page, 'L-1003', 'Venue hire');
  const card = swbsCard(page);
  await expect(card.getByText('SWBS code received')).toBeVisible();
  await expect(card.getByText('SIM-DE03-0001')).toBeVisible();
  await expect(card.getByText('Simulated GSAP', { exact: true })).toBeVisible();
  await expect(card.getByText('Generated from the line-item number')).toBeVisible();
  await card.getByRole('button', { name: 'Copy HX key HX1003' }).click();
  await expectToast(page, 'HX key copied');
  expect(await page.evaluate(() => navigator.clipboard.readText())).toBe('HX1003');
  await expectNoAxeViolations(page);

  await openLineItem(page, 'L-1004', 'Speaker fees');
  await expect(swbsCard(page).getByText('Not requested', { exact: true })).toBeVisible();

  await openLineItem(page, 'L-1006', 'Stand build');
  await expect(swbsCard(page).getByText('Ready to send to GSAP')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Send to simulated GSAP now' })).toHaveCount(0);

  await page.goto('/financials');
  await expect(page.getByRole('heading', { name: 'Financials', level: 1 })).toBeVisible();
  await expect(page.getByRole('region', { name: 'Simulated GSAP' })).toHaveCount(0);
});

test('030 US4: a finance person sends one line item to the simulated GSAP from its page', async ({
  page,
}) => {
  await signInAs(page, ALEX);
  await openLineItem(page, 'L-1006', 'Stand build');

  await swbsCard(page).getByRole('button', { name: 'Send to simulated GSAP now' }).click();
  const confirm = page.getByRole('alertdialog', {
    name: 'Send Stand build to simulated GSAP now?',
  });
  await expect(confirm).toContainText("Codes can't be changed or removed in SP&A One.");
  await confirm.getByRole('button', { name: 'Send to simulated GSAP now' }).click();

  const card = swbsCard(page);
  await expect(card.getByText('SIM-GB01-0001')).toBeVisible();
  await expect(card.getByText('SWBS code received')).toBeVisible();
  await expectNoAxeViolations(page);
});

test('030 US1: after approval, the waiting request goes to the simulated GSAP from Financials', async ({
  page,
}) => {
  await signInAs(page, DANIEL);
  await page.goto('/approvals');
  await page
    .getByRole('searchbox', { name: 'Search activity name or ID' })
    .fill('Trade magazine ads');
  await page.getByRole('button', { name: 'Trade magazine ads', exact: true }).click();
  const preview = page.getByRole('dialog', { name: 'Trade magazine ads' });
  await preview.getByRole('button', { name: 'Approve', exact: true }).click();
  const approve = page.getByRole('alertdialog', { name: "Approve 'Trade magazine ads'?" });
  await approve.getByRole('button', { name: 'Approve', exact: true }).click();
  await expectToast(page, "Activity 'Trade magazine ads' approved");

  await signInAs(page, ALEX);
  await page.goto('/financials');
  const panel = page.getByRole('region', { name: 'Simulated GSAP' });
  await expect(panel.getByText('1 line item is ready to send')).toBeVisible();
  await panel.getByRole('button', { name: 'Send SWBS requests to simulated GSAP' }).click();
  const confirm = page.getByRole('alertdialog', { name: 'Send 1 SWBS request to simulated GSAP?' });
  await expectNoAxeViolations(page);
  await confirm.getByRole('button', { name: 'Send SWBS requests to simulated GSAP' }).click();

  const result = panel.getByRole('status');
  await expect(result).toContainText('Simulated GSAP returned 1 SWBS code');
  await expect(result).toContainText('Media booking (L-1005) · SIM-NL02-0001');
  await expect(panel.getByText('No line item is waiting for an SWBS code')).toBeVisible();
  await expect(
    panel.getByRole('button', { name: 'Send SWBS requests to simulated GSAP' }),
  ).toBeDisabled();
  await expect(panel.getByText(/^Simulated GSAP · SWR-\d+$/).first()).toBeVisible();
  await expectNoAxeViolations(page);
});

test('030 US3: a received code stays and keeps the request on', async ({ page }) => {
  await signInAs(page, PRIYA);
  await page.goto('/line-items/L-1005/edit');

  const request = page.getByLabel('Request SWBS code from GSAP');
  await expect(request).toBeChecked();
  await expect(request).toBeDisabled();
  await expect(
    page.getByText(
      'This line item already has its SWBS code from the simulated GSAP, so the request stays on.',
    ),
  ).toBeVisible();
  await page.getByLabel('Description').fill('Print and online booking');
  await page.getByRole('button', { name: 'Save', exact: true }).click();

  await expect(page.getByRole('heading', { name: 'Media booking', level: 1 })).toBeVisible();
  await expect(swbsCard(page).getByText('SIM-NL02-0001')).toBeVisible();
  await expect(swbsCard(page).getByText('SWBS code received')).toBeVisible();
});
