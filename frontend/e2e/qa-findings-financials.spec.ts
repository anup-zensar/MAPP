import type { Page } from '@playwright/test';
import { expect, test } from './support/no-csp-violations';
import { callApi, expectNoAxeViolations, signInAs, uniqueName } from './fixtures';
import { idOf } from './planning';

// 038 US2 (H-4, H-5) against the real API and the demo GSAP data: not-linked transactions only for people
// who import, Link and Unlink only where the server allows, and a hidden record answers like a missing one
// (FR-004, FR-005, FR-007).

const ALEX = 'alex.morgan@shell.com';
const MEI = 'mei.lin@shell.com';
const GONE = "This GSAP transaction doesn't exist.";

interface Answer {
  status: number;
  body: unknown;
}

async function post(page: Page, url: string, data: unknown): Promise<Answer> {
  const response = await page.request.post(url, { data });
  return { status: response.status(), body: await response.json() };
}

const link = (id: string) => `/api/v1/financials/transactions/${id}/link`;
const unlink = (id: string) => `/api/v1/financials/transactions/${id}/unlink`;

test('038 FR-004, FR-007: Mei, a Planner, sees no Not linked tab and FT-1004 answers like a missing transaction', async ({
  page,
}) => {
  await signInAs(page, MEI);

  await page.goto('/financials');
  const tabs = page.getByRole('tablist', { name: 'Transaction type' });
  await expect(tabs.getByRole('tab', { name: /^All \(\d+\)$/ })).toBeVisible();
  await expect(tabs.getByRole('tab', { name: /Not linked/ })).toHaveCount(0);
  await expectNoAxeViolations(page);

  await page.goto('/financials/transactions/FT-1004');
  await expect(page.getByRole('dialog').getByText(GONE)).toBeVisible();

  const hidden = await post(page, link('FT-1004'), { lineItemId: 'L-1007', version: 1 });
  const missing = await post(page, link('FT-9999'), { lineItemId: 'L-1007', version: 1 });
  expect(hidden.status).toBe(404);
  expect(hidden).toEqual(missing);

  // So does unlinking a transaction of a plan outside her teams (FT-1003, Lubricants Europe).
  const outside = await post(page, unlink('FT-1003'), { version: 1 });
  const absent = await post(page, unlink('FT-9999'), { version: 1 });
  expect(outside.status).toBe(404);
  expect(outside).toEqual(absent);
});

test('038 FR-007: an importer of Lubricants Europe gets the same answer for an out-of-scope and a missing line item', async ({
  page,
}) => {
  await signInAs(page, ALEX);
  const email = `importer.${Date.now().toString(36)}@shell.com`;
  const teamId = await idOf(page, '/api/v1/teams', 'Lubricants Europe');
  const person = await callApi<{ id: string }>(page, 'POST', '/api/v1/people', {
    name: uniqueName('Importer'),
    email,
    jobTitle: 'Finance',
    teamIds: [teamId],
  });
  const role = await callApi<{ id: string }>(page, 'POST', '/api/v1/roles', {
    name: uniqueName('Importer'),
    permissions: ['plans.view', 'finance.view', 'finance.import', 'finance.link'],
    defaultScope: 'myTeams',
  });
  const roles = await callApi<{ version: number }>(
    page,
    'GET',
    `/api/v1/people/${person.id}/roles`,
  );
  const assigned = await page.request.put(`/api/v1/people/${person.id}/roles`, {
    data: { version: roles.version, roles: [{ roleId: role.id, scope: { type: 'myTeams' } }] },
  });
  expect(assigned.ok()).toBe(true);

  await signInAs(page, email);
  const transaction = await callApi<{ version: number }>(
    page,
    'GET',
    '/api/v1/financials/transactions/FT-1004',
  );
  // L-1007 is in FY2026 Mobility China, outside the importer's team.
  const outOfScope = await post(page, link('FT-1004'), {
    lineItemId: 'L-1007',
    version: transaction.version,
  });
  const missing = await post(page, link('FT-1004'), {
    lineItemId: 'L-9999',
    version: transaction.version,
  });
  // The same answer, but for the ID typed.
  expect(outOfScope.status).toBe(400);
  expect(missing.status).toBe(400);
  expect(JSON.stringify(outOfScope.body).replaceAll('L-1007', 'ID')).toBe(
    JSON.stringify(missing.body).replaceAll('L-9999', 'ID'),
  );
});

test('038 FR-005: Alex links FT-1004 to L-1007 and unlinks it again', async ({ page }) => {
  await signInAs(page, ALEX);

  await page.goto('/financials/transactions/FT-1004');
  const panel = page.getByRole('dialog', { name: 'PO 4500001004 / 10' });
  await panel.getByRole('button', { name: 'Link to a line item' }).click();
  const dialog = page.getByRole('dialog', { name: "Link 'PO 4500001004 / 10' to a line item" });
  await dialog.getByRole('textbox', { name: /Line item ID/ }).fill('L-1007');
  await dialog.getByRole('button', { name: 'Link', exact: true }).click();

  const unlinkButton = panel.getByRole('button', { name: 'Unlink' });
  await expect(unlinkButton).toBeVisible();
  await expect(panel.getByRole('navigation', { name: 'Linked to' })).toContainText(
    'Station signage (L-1007)',
  );
  await expectNoAxeViolations(page);

  await unlinkButton.click();
  const confirm = page.getByRole('alertdialog', {
    name: "Unlink 'PO 4500001004 / 10' from 'Station signage (L-1007)'?",
  });
  await confirm.getByRole('button', { name: 'Unlink' }).click();
  await expect(panel.getByRole('button', { name: 'Link to a line item' })).toBeVisible();
});
