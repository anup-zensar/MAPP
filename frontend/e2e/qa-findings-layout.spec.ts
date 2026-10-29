import type { Page } from '@playwright/test';
import { expect, test } from './support/no-csp-violations';
import { callApi, expectNoAxeViolations, signInAs } from './fixtures';
import { createPlan, createPlanningRecords } from './planning';
import { xlsx } from './xlsx';

// 038 US13 (FR-036 – FR-038; M-14 – M-17): long names wrap, pages fit 390, 768 and 1440 px without
// sideways scroll, sideways-scrolling tables are reachable by keyboard and text on grey reaches 4.5:1.

const MEI = 'mei.lin@shell.com';
const PRIYA = 'priya.raman@shell.com';
const WIDTHS = [390, 768, 1440] as const;

async function fitsTheWindow(page: Page): Promise<void> {
  const { scrollWidth, innerWidth } = await page.evaluate(() => ({
    scrollWidth: document.documentElement.scrollWidth,
    innerWidth: window.innerWidth,
  }));
  expect(scrollWidth).toBeLessThanOrEqual(innerWidth);
}

/** A 120-character name without a space, the longest a plan may have. */
function longName(): string {
  return `Unbroken${Date.now()}`.padEnd(120, 'x');
}

test('038 FR-036 (M-14, M-15): a 120-character name wraps; Plans and the SP&A view fit the window', async ({
  page,
}) => {
  await signInAs(page, MEI);
  const plan = await createPlan(page, 'Layout');
  const current = await callApi<{ version: number }>(page, 'GET', `/api/v1/plans/${plan.id}`);
  const name = longName();
  await callApi(page, 'PATCH', `/api/v1/plans/${plan.id}`, { name, version: current.version });

  for (const width of WIDTHS) {
    await page.setViewportSize({ width, height: 900 });
    await page.goto(`/plans?year=${plan.year}`);
    await expect(page.getByRole('link', { name }).first()).toBeVisible();
    await fitsTheWindow(page);
  }

  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto(`/plans/${plan.id}/spa`);
  await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
  await fitsTheWindow(page);
});

for (const colorScheme of ['light', 'dark'] as const) {
  test.describe(`${colorScheme} mode`, () => {
    test.use({ colorScheme, viewport: { width: 390, height: 844 } });

    test(`038 FR-037, FR-038 (M-16, M-17): import problems, split links, an empty Regional dashboard and the purchase-request form pass axe (${colorScheme})`, async ({
      page,
    }) => {
      await signInAs(page, MEI);
      const records = await createPlanningRecords(page, 'Axe 038', {
        withWorkflow: false,
        leaveEmpty: [],
      });

      await page.goto(`/plans/${records.planId}/import`);
      await page.getByLabel('Choose a file').setInputFiles({
        name: 'problems.xlsx',
        mimeType: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
        buffer: xlsx({
          Campaigns: [
            ['ID', 'Campaign name'],
            ['C-9999', ''],
          ],
        }),
      });
      await page.getByRole('button', { name: 'Check file' }).click();
      const problems = page.getByRole('region', { name: 'Problems, scrolls sideways' });
      await expect(problems).toBeVisible();
      await problems.focus();
      await expect(problems).toBeFocused();
      await expectNoAxeViolations(page);

      await page.goto(records.activityPath);
      const name = records.activityName;
      await page
        .getByRole('button', { name: `Actions for ${name}` })
        .first()
        .click();
      await page.getByRole('menuitem', { name: 'Split activity' }).click();
      const dialog = page.getByRole('dialog', { name: `Split '${name}'` });
      await dialog.getByRole('textbox', { name: /^New activity name/ }).fill(`${name} part`);
      await dialog.getByRole('textbox', { name: /^Planned cost to move/ }).fill('1000.00');
      await dialog.getByRole('button', { name: 'Split activity' }).click();
      await expect(page.getByRole('list', { name: 'Split' })).toBeVisible();
      await expectNoAxeViolations(page);

      await signInAs(page, PRIYA);
      await page.goto('/dashboards/regional?year=2031');
      await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
      await expectNoAxeViolations(page);

      await page.goto('/purchase-requests/new?lineItemId=L-1003');
      await expect(
        page.getByRole('heading', { name: 'New purchase request', level: 1 }),
      ).toBeVisible();
      await expectNoAxeViolations(page);
    });
  });
}
