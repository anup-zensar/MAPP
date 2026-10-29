import { readFile } from 'node:fs/promises';
import { expect, test } from './support/no-csp-violations';
import { expectNoAxeViolations, expectToast, signInAs } from './fixtures';

// 038 US3: the 5,000-row workbook written by platform/apibench TestWorkbookCheckStatements_038_FR008_H2.

const PRIYA = 'priya.raman@shell.com';
const PLAN = 'P-1001';
const FILE = 'workbook-5000.xlsx';

test('038 FR-008 (H-2): check, import and undo a 5,000-row workbook', async ({ page }) => {
  test.setTimeout(300_000);
  await signInAs(page, PRIYA);
  await page.goto(`/plans/${PLAN}`);
  await page.getByRole('main').getByRole('button', { name: 'More', exact: true }).click();
  await page.getByRole('menuitem', { name: 'Import from Excel' }).click();
  await expect(page.getByRole('heading', { name: 'Import from Excel', level: 1 })).toBeVisible();

  await page.getByLabel('Choose a file').setInputFiles({
    name: FILE,
    mimeType: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
    buffer: await readFile(new URL(`./fixtures/${FILE}`, import.meta.url)),
  });
  await page.getByRole('button', { name: 'Check file' }).click();
  await expect(page.getByText('5000 rows · 5000 ready · 0 with problems')).toBeVisible({
    timeout: 20_000,
  });
  await expectNoAxeViolations(page);

  await page.getByRole('button', { name: 'Import 5000 rows' }).click();
  await expect(
    page.getByText('Created 1 campaign, 50 initiatives, 2474 activities, 2475 line items.'),
  ).toBeVisible({ timeout: 120_000 });
  await expectNoAxeViolations(page);

  await page.getByRole('button', { name: 'Undo import' }).click();
  const confirm = page.getByRole('alertdialog', { name: `Undo import of '${FILE}'?` });
  await confirm.getByRole('button', { name: 'Undo import' }).click();
  await expect(page.getByText(/Undone by Priya Raman/)).toBeVisible({ timeout: 120_000 });
  await expectToast(page, 'Import undone — the records are in the recycle bin');
});
