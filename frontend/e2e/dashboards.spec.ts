import { readFile } from 'node:fs/promises';
import { expect, test, type Page } from '@playwright/test';
import { expectNoAxeViolations, signInAs } from './fixtures';

// Planning dashboards and exports (036): the Executive dashboard (US1) and a plan's PowerPoint and PDF (US2).

const CARLOS = 'carlos.mendes@shell.com';
const PRIYA = 'priya.raman@shell.com';
const SEED_YEAR = 2026;

async function downloaded(page: Page, item: string): Promise<{ name: string; body: Buffer }> {
  await page.getByRole('button', { name: 'More', exact: true }).click();
  const downloading = page.waitForEvent('download');
  await page.getByRole('menuitem', { name: item }).click();
  const download = await downloading;
  return { name: download.suggestedFilename(), body: await readFile(await download.path()) };
}

test('036 US1: the Executive dashboard adds up every market a person may see, and only those (FR-001 – FR-019)', async ({
  page,
}) => {
  await signInAs(page, CARLOS);
  await page.goto('/');
  await page.getByRole('link', { name: 'Dashboards' }).click();
  await expect(page.getByRole('heading', { level: 1, name: 'Executive dashboard' })).toBeVisible();
  await page.getByLabel('Year', { exact: true }).selectOption(String(SEED_YEAR));
  await expect(page).toHaveURL(/year=2026/);

  const markets = page.getByRole('table', { name: /Plans and figures by market/ });
  await expect(markets.getByRole('link', { name: 'Lubricants Europe' })).toBeVisible();
  await expect(markets.getByRole('link', { name: 'Mobility China', exact: true })).toBeVisible();
  await expect(page.getByLabel('Summary').getByText(/^\$[\d,]+\.\d{2}$/).first()).toBeVisible();
  await expect(page.getByRole('img', { name: /Planned cost by month/ })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Approvals' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Global budgets' })).toBeVisible();
  await expectNoAxeViolations(page);

  await page.getByRole('button', { name: 'Table' }).click();
  await expect(page.getByRole('rowheader', { name: 'Mar 2026' })).toBeVisible();
  await page.getByLabel('Currency', { exact: true }).selectOption('EUR');
  await expect(page.getByLabel('Summary').getByText(/^[\d,]+\.\d{2} EUR$/).first()).toBeVisible();

  await markets.getByRole('link', { name: 'Mobility China', exact: true }).click();
  await expect(page.getByRole('heading', { level: 1, name: 'Regional dashboard' })).toBeVisible();
  await expect(page.getByRole('link', { name: 'FY2026 Mobility China', exact: true })).toBeVisible();
  await expectNoAxeViolations(page);

  await signInAs(page, PRIYA);
  await page.goto(`/dashboards/executive?year=${SEED_YEAR}`);
  await expect(page.getByRole('link', { name: 'FY2026 Lubricants Europe', exact: true })).toBeVisible();
  await expect(page.getByRole('link', { name: 'Mobility China', exact: true })).toHaveCount(0);
  await expect(page.getByRole('link', { name: 'FY2026 Mobility China', exact: true })).toHaveCount(0);
});

test('036 US2: a plan downloads as PowerPoint and PDF files that open, with every page (FR-023, FR-025)', async ({
  page,
}) => {
  await signInAs(page, PRIYA);
  await page.goto('/plans/P-1001');
  await expect(
    page.getByRole('heading', { level: 1, name: 'FY2026 Lubricants Europe' }),
  ).toBeVisible();
  await page.getByRole('button', { name: 'More', exact: true }).click();
  await expect(page.getByRole('menuitem', { name: 'Export to PowerPoint' })).toBeVisible();
  await expectNoAxeViolations(page);
  await page.keyboard.press('Escape');

  const pptx = await downloaded(page, 'Export to PowerPoint');
  expect(pptx.name).toMatch(/^SP&A One – FY2026 Lubricants Europe – .+\.pptx$/);
  expect(pptx.body.subarray(0, 4).toString('latin1')).toBe('PK\u0003\u0004');
  const slides = new Set(pptx.body.toString('latin1').match(/ppt\/slides\/slide\d+\.xml/g));
  expect(slides.size).toBe(4);
  await expect(
    page.getByText(/^Exported 'SP&A One – FY2026 Lubricants Europe/).first(),
  ).toBeVisible();

  const pdf = await downloaded(page, 'Export to PDF');
  expect(pdf.name).toMatch(/\.pdf$/);
  const text = pdf.body.toString('latin1');
  expect(text.startsWith('%PDF-1.4')).toBe(true);
  expect(text.trimEnd().endsWith('%%EOF')).toBe(true);
  expect(text.match(/\/Type \/Page /g)).toHaveLength(4);
});
