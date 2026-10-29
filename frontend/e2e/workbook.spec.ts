import { readFile } from 'node:fs/promises';
import { expect, test, type Page } from '@playwright/test';
import { expectNoAxeViolations, expectToast, signInAs } from './fixtures';
import { createPlan, createPlanningRecords } from './planning';
import { xlsx } from './xlsx';

// Feature 012 US2 and US3 against the real API: build a plan's skeleton from a workbook, undo the import,
// and send an exported plan back unchanged. The plans are the test's own (next year, as Mei Lin).

const MEI = 'mei.lin@shell.com';

async function openImport(page: Page, planId: string) {
  await page.goto(`/plans/${planId}`);
  await page.getByRole('main').getByRole('button', { name: 'More', exact: true }).click();
  await page.getByRole('menuitem', { name: 'Import from Excel' }).click();
  await expect(page.getByRole('heading', { name: 'Import from Excel', level: 1 })).toBeVisible();
}

test('012 US2: import a skeleton from a workbook, see every problem, then undo the import', async ({
  page,
}) => {
  await signInAs(page, MEI);
  const plan = await createPlan(page, 'Workbook');
  await openImport(page, plan.id);
  await expectNoAxeViolations(page);

  const template = page.waitForEvent('download');
  await page.getByRole('button', { name: 'Download template' }).click();
  expect((await template).suggestedFilename()).toBe(`SP&A One import template – ${plan.name}.xlsx`);

  await page.getByLabel('Choose a file').setInputFiles({
    name: 'skeleton.xlsx',
    mimeType: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
    buffer: xlsx({
      Campaigns: [
        ['ID', 'Campaign name'],
        ['', 'Spring push'],
      ],
      Initiatives: [
        ['Campaign', 'Initiative name'],
        ['Spring push', 'Forecourt'],
      ],
      Activities: [
        ['Campaign', 'Initiative', 'Activity name', 'Activity type'],
        ['Spring push', 'Forecourt', 'Pump toppers', 'Event'],
        ['Spring push', 'Forecourt', 'Moon base', 'Teleportation'],
      ],
      'Line items': [
        ['Campaign', 'Initiative', 'Activity', 'Line item name', 'Currency', 'Planned cost'],
        ['Spring push', 'Forecourt', 'Pump toppers', 'Printing', 'USD', '2000'],
      ],
    }),
  });
  await page.getByRole('button', { name: 'Check file' }).click();

  await expect(page.getByText('5 rows · 4 ready · 1 with problems')).toBeVisible();
  await expect(
    page
      .getByRole('table', { name: 'Problems' })
      .getByText(/'Teleportation' is not an activity type the model/),
  ).toBeVisible();
  await expectNoAxeViolations(page);
  await page.getByRole('button', { name: 'Import 4 rows' }).click();

  await expect(
    page.getByText('Created 1 campaign, 1 initiative, 1 activity, 1 line item.'),
  ).toBeVisible();
  await expectNoAxeViolations(page);
  await page.getByRole('button', { name: 'Undo import' }).click();
  const confirm = page.getByRole('alertdialog', { name: "Undo import of 'skeleton.xlsx'?" });
  await confirm.getByRole('button', { name: 'Undo import' }).click();
  await expectToast(page, 'Import undone — the records are in the recycle bin');
  await expect(page.getByText(/Undone by Mei Lin/)).toBeVisible();
});

test('012 US3: an exported plan sent back unchanged changes nothing', async ({ page }) => {
  await signInAs(page, MEI);
  const records = await createPlanningRecords(page, 'Round trip', {
    withWorkflow: false,
    leaveEmpty: [],
  });
  await openImport(page, records.planId);

  const exported = page.waitForEvent('download');
  await page.getByRole('button', { name: 'Export to Excel' }).click();
  const download = await exported;
  const path = await download.path();
  await page.getByLabel('Choose a file').setInputFiles({
    name: download.suggestedFilename(),
    mimeType: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
    buffer: await readFile(path),
  });
  await page.getByRole('button', { name: 'Check file' }).click();

  await expect(page.getByText('Nothing to change')).toBeVisible();
  await expect(
    page.getByText('The file holds what the plan holds — nothing to import.'),
  ).toBeVisible();
});
