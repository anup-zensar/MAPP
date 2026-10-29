import { readFile } from 'node:fs/promises';
import { expect, test } from '@playwright/test';
import { expectNoAxeViolations, expectToast, signInAs, uniqueName } from './fixtures';

// All work (010): the grid, the calendar, budget and cost by month, saved views and the export.

const PRIYA = 'priya.raman@shell.com';

/** The year of the demo plans FY2026 Lubricants Europe and FY2026 Mobility China. */
const SEED_YEAR = 2026;

/** The current year: the default year of a new finance budget item, and the year All work marks Today in. */
const THIS_YEAR = new Date().getFullYear();

test('All work: filter and search the grid, the calendar, budget and cost by month, a saved view and a CSV export (010)', async ({
  page,
}) => {
  test.slow();
  await signInAs(page, PRIYA);
  // The demo plans' year, named in the address (D-122), so the journey holds in any year.
  await page.goto(`/all-work?year=${SEED_YEAR}`);
  await expect(page.getByRole('heading', { name: 'All work', level: 1 })).toBeVisible();
  await expect(page.getByText('Filter, compare and export all planned work.')).toBeVisible();
  const filters = page.getByRole('group', { name: 'Filters' });
  // 016 FR-034: the year filter is a range, "Years · 2026" for one year.
  await expect(filters.getByRole('button', { name: `Years · ${SEED_YEAR}` })).toBeVisible();
  const count = page.getByText(/^\d+ of \d+ activit(y|ies)$/);
  await expect(count).toHaveText(/^(\d+) of \1 activities$/);
  const grid = page.getByRole('table', { name: 'All work grid' });
  const gridRow = (name: string) => grid.getByRole('row').filter({ hasText: name });
  await expect(gridRow('Fleet roadshow')).toBeVisible();
  await expectNoAxeViolations(page);

  // A filter (FR-004, FR-005): only declined activities, with their plan, campaign and initiative.
  await filters.getByRole('button', { name: 'Activity status' }).click();
  await page
    .getByRole('dialog', { name: 'Activity status' })
    .getByRole('checkbox', { name: 'Declined' })
    .check();
  await page.keyboard.press('Escape');
  await expect(
    page.getByRole('button', { name: 'Remove filter: Activity status: Declined' }),
  ).toBeVisible();
  await expect(filters.getByRole('button', { name: 'Activity status · 1' })).toBeVisible();
  await expect(gridRow('Fleet roadshow')).toContainText('Declined');
  await expect(gridRow('Launch film')).toHaveCount(0);
  await expect(count).not.toHaveText(/^(\d+) of \1 activities$/);
  await page.getByRole('button', { name: 'Clear all' }).click();
  await expect(page.getByRole('button', { name: /^Remove filter: / })).toHaveCount(0);
  await expect(count).toHaveText(/^(\d+) of \1 activities$/);

  // Search (FR-006) and a filter together: one Complete activity.
  await filters.getByRole('button', { name: 'Activity status' }).click();
  await page
    .getByRole('dialog', { name: 'Activity status' })
    .getByRole('checkbox', { name: 'Complete' })
    .check();
  await page.keyboard.press('Escape');
  await page.getByRole('searchbox', { name: 'Search' }).fill('Billboard');
  await expect(count).toHaveText(/^1 of \d+ activities$/);
  const billboard = gridRow('Billboard winter wave').filter({ hasText: 'A-1003' });
  await expect(billboard).toContainText('Complete');
  await expect(billboard).toContainText(`05 Jan ${SEED_YEAR}`);
  await expect(billboard).toContainText('$45,000.00');
  await expect(billboard.locator('mark')).toHaveText('Billboard');
  await expect(grid.getByRole('row').filter({ hasText: 'Total · 1 activity' })).toContainText(
    '$45,000.00',
  );
  await expect(page.getByRole('region', { name: 'Summary' })).toContainText('$45K');

  // The calendar (FR-025 – FR-028): the year's timeline with today's line, then the month grid.
  await page.getByRole('tab', { name: 'Calendar' }).click();
  const calendar = page.getByRole('table', { name: `Calendar ${SEED_YEAR}` });
  await expect(calendar).toBeVisible();
  // The Today line is drawn only on the current year's timeline (FR-026).
  await expect(calendar.getByRole('columnheader').filter({ hasText: 'Today' })).toHaveCount(
    SEED_YEAR === THIS_YEAR ? 1 : 0,
  );
  await expect(
    calendar.getByRole('link', { name: 'Billboard winter wave Complete' }),
  ).toBeVisible();
  await expect(
    page.getByRole('tabpanel', { name: 'Calendar' }).getByText('Status', { exact: true }).first(),
  ).toBeVisible();
  await expectNoAxeViolations(page);
  await page.getByRole('button', { name: 'Month', exact: true }).click();
  await expect(page.getByRole('button', { name: 'Previous month' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Next month' })).toBeVisible();
  await expect(page.getByRole('columnheader', { name: 'Mon' })).toBeVisible();
  await expectNoAxeViolations(page);
  await page.getByRole('button', { name: 'Timeline' }).click();

  // Budget and cost by month, then by quarter (FR-022 – FR-024): Jan–Feb only, 45,000.00 in total.
  await page.getByRole('tab', { name: 'Budget & cost' }).click();
  const budget = page.getByRole('table', { name: 'Budget & cost' });
  await expect(budget.getByRole('columnheader', { name: 'Jan' })).toBeVisible();
  await expect(budget.getByRole('columnheader', { name: 'Dec' })).toBeVisible();
  await expect(page.getByText('To change how a cost is spread, open its line item')).toBeVisible();
  const footer = budget.getByRole('row').filter({ hasText: 'Total · 1 activity' });
  await expect(footer).toContainText('$45,000.00');
  await page.getByRole('button', { name: 'Quarterly' }).click();
  await expect(budget.getByRole('columnheader', { name: 'Q1' })).toBeVisible();
  const billboardBudget = budget.getByRole('row').filter({ hasText: 'Billboard winter wave' });
  // Q1, Q2, Q3, Q4, Total.
  await expect(billboardBudget.getByRole('cell')).toHaveText([
    '',
    '$45,000.00',
    '—',
    '—',
    '—',
    '$45,000.00',
  ]);

  // Save the view, clear everything, then reopen the view (FR-029 – FR-031).
  const viewName = uniqueName('E2E view');
  await page.getByRole('button', { name: 'Views' }).click();
  await page.getByRole('menuitem', { name: 'Save current view…' }).click();
  const save = page.getByRole('dialog', { name: 'Save current view' });
  await save.getByRole('textbox', { name: /^Name/ }).fill(viewName);
  await save.getByRole('button', { name: 'Save view' }).click();
  await expectToast(page, `View '${viewName}' saved`);
  await expect(page.getByRole('button', { name: `View: ${viewName}` })).toBeVisible();

  await page.getByRole('button', { name: 'Clear all' }).click();
  await page.getByRole('tab', { name: 'Grid' }).click();
  await expect(page.getByRole('searchbox', { name: 'Search' })).toHaveValue('');
  await expect(count).toHaveText(/^(\d+) of \1 activities$/);

  await page.getByRole('button', { name: `View: ${viewName}` }).click();
  await page.getByRole('menuitemcheckbox', { name: viewName }).click();
  await expect(page.getByRole('searchbox', { name: 'Search' })).toHaveValue('Billboard');
  await expect(
    page.getByRole('button', { name: 'Remove filter: Activity status: Complete' }),
  ).toBeVisible();
  await expect(page.getByRole('tab', { name: 'Budget & cost' })).toHaveAttribute(
    'aria-selected',
    'true',
  );
  await expect(count).toHaveText(/^1 of \d+ activities$/);

  // Export the grid to CSV (FR-032 – FR-034): the notes about the export, then the rows.
  await page.getByRole('tab', { name: 'Grid' }).click();
  await page.getByRole('button', { name: 'Export' }).click();
  const downloading = page.waitForEvent('download');
  await page.getByRole('menuitem', { name: 'CSV (.csv)' }).click();
  const download = await downloading;
  const fileName = download.suggestedFilename();
  expect(fileName).toMatch(new RegExp(`^SP&A One All work – Grid – ${SEED_YEAR} – .+\\.csv$`));
  await expectToast(page, `Exported '${fileName}'`);
  const csv = await readFile(await download.path(), 'utf8');
  expect(csv).toContain('View,Grid');
  expect(csv).toContain('Search,Billboard');
  expect(csv).toContain('Exported by,Priya Raman');
  expect(csv).toMatch(/^Level,Path,Name,/m);
  expect(csv).toMatch(/^Activity,.*,Billboard winter wave,,A-1003,Complete,/m);
  expect(csv).not.toContain('Fleet roadshow');
});
