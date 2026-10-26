import { expect, test } from '@playwright/test';
import { signInAs } from './fixtures';
import { createPlanningRecords } from './planning';

const MEI = 'mei.lin@shell.com';

// The plan page's Financials and Budget by month load on their own, above the tree. When they arrive while a
// row's menu is open, the row and its menu stay where the person sees them (constitution IX, no surprises).
test('A row menu stays usable while the plan page’s other sections finish loading (005 US6, constitution IX)', async ({
  page,
}) => {
  await signInAs(page, MEI);
  const records = await createPlanningRecords(page, 'Menu', {
    withWorkflow: false,
    leaveEmpty: [],
  });
  let release = (): void => undefined;
  const held = new Promise<void>((resolve) => {
    release = resolve;
  });
  await page.route(/\/budget-by-month|\/financials\/records\//, async (route) => {
    await held;
    await route.continue();
  });

  await page.goto(`/plans/${records.planId}`);
  await page.getByRole('button', { name: 'Expand all' }).click();
  await page.getByRole('button', { name: `Actions for ${records.campaignName}` }).click();
  const menu = page.getByRole('menu', { name: `Actions for ${records.campaignName}` });
  await expect(menu).toBeVisible();
  release();
  await expect(page.getByRole('status').filter({ hasText: 'Loading…' })).toHaveCount(0);

  await menu.getByRole('menuitem', { name: 'Copy' }).click({ timeout: 10_000 });
  await expect(
    page.getByRole('dialog', { name: `Copy campaign '${records.campaignName}'` }),
  ).toBeVisible();
});
