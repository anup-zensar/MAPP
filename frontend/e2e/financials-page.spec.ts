import { expect, test } from '@playwright/test';
import { expectNoAxeViolations, expectToast, signInAs, uniqueName } from './fixtures';
import { createLineItem } from './planning-changes';
import { createPlanningRecords } from './planning';

// Feature 015 US3 and US4 against the real API: the Financials page on the demo GSAP data (backend/seed),
// and an import of a generated GSAP extract that links to records of the test's own.

const ALEX = 'alex.morgan@shell.com';
const MEI = 'mei.lin@shell.com';
// The demo data's source, or the last import's when another journey imported a file (US4).
const SOURCE_LINE = /^GSAP data as of \d{2} \w{3} \d{4}, \d{2}:\d{2} · (Demo data|.+'.+\.csv')/;

test('015 US3 (T030): from the sidebar, a person finds a PO, opens it and follows its link to the line item', async ({
  page,
}) => {
  await signInAs(page, ALEX);
  await page.goto('/');
  await page
    .getByRole('navigation', { name: 'Main' })
    .getByRole('link', { name: 'Financials' })
    .click();
  await expect(page.getByRole('heading', { name: 'Financials', level: 1 })).toBeVisible();
  await expect(page.getByText(SOURCE_LINE)).toBeVisible();

  const search = page.getByRole('searchbox', { name: 'Search transactions' });
  await search.fill('keynote');
  const table = page.getByRole('table', { name: 'Transactions' });
  await expect(page.getByRole('tab', { name: 'All (4)' })).toBeVisible();
  await expect(table.getByRole('row')).toHaveCount(5);
  for (const row of await table.getByRole('row').all()) {
    await expect(row).toContainText(/Vendor|Keynote Speakers Ltd/);
  }

  await search.fill('4500001003');
  await table.getByRole('link', { name: 'PO 4500001003 / 10' }).click();
  await expect(page).toHaveURL(/\/financials\/transactions\/FT-1003\?/);
  const panel = page.getByRole('dialog', { name: 'PO 4500001003 / 10' });
  await expect(panel.getByText('Purchase order · FT-1003')).toBeVisible();
  for (const [term, value] of [
    ['Document number', '4500001003'],
    ['Amount', '9,800.00 EUR'],
    ['Amount in USD', '$10,652.17'],
    ['Vendor', 'V-100103 · Outdoor Frames Ltd'],
    ['Matched by', 'HX key HX1006'],
  ] as const) {
    await expect(
      panel.getByRole('term').filter({ hasText: new RegExp(`^${term}$`) }),
    ).toBeVisible();
    await expect(panel.getByRole('definition').filter({ hasText: value }).first()).toBeVisible();
  }
  const path = panel.getByRole('navigation', { name: 'Linked to' });
  for (const name of [
    'FY2026 Lubricants Europe',
    'Helix launch',
    'Print & outdoor',
    'Billboard winter wave',
    'Stand build (L-1006)',
  ]) {
    await expect(path.getByRole('link', { name, exact: true })).toBeVisible();
  }
  await expectNoAxeViolations(page);
  await path.getByRole('link', { name: 'Stand build (L-1006)' }).click();
  await expect(page).toHaveURL(/\/line-items\/L-1006$/);
  await expect(page.getByRole('heading', { name: 'Stand build', level: 1 })).toBeVisible();

  // Scenario 4: the Not linked tab, with the reason in the row.
  await page.goto('/financials');
  await page.getByRole('tab', { name: /^Not linked \(\d+\)$/ }).click();
  const notLinked = page
    .getByRole('table', { name: 'Transactions' })
    .getByRole('row', { name: /^PO 4500001004 \/ 10 / });
  await expect(notLinked).toContainText('Not linked');
  await expect(notLinked).toContainText(
    'No line item for HX key HX1099 — check the tracking key in GSAP.',
  );
  await expectNoAxeViolations(page);

  await page.goto('/financials?recordId=A-1002');
  const chip = page.getByRole('button', { name: /Activity: Social always-on \(A-1002\)/ });
  await expect(chip).toBeVisible();
  await expect(page.getByText('7 of 7 transactions')).toBeVisible();
  await chip.click();
  await expect(page).not.toHaveURL(/recordId=/);
  await expect(page.getByText(/^\d+ of \d+ transactions$/)).not.toHaveText('7 of 7 transactions');

  await page.setViewportSize({ width: 390, height: 844 });
  await expect(page.getByRole('list', { name: 'Transactions' })).toBeVisible();
  await expect(page.getByRole('table', { name: 'Transactions' })).toHaveCount(0);
});

test('015 US4 (T035): a generated GSAP extract is checked, imported, and changes the activity’s figures', async ({
  page,
}) => {
  test.slow();
  await signInAs(page, MEI);
  const records = await createPlanningRecords(page, 'Import', {
    withWorkflow: false,
    leaveEmpty: [],
  });
  const lineItem = await createLineItem(page, records.activityId, 'Import', '8000.00');
  const hxKey = `HX${lineItem.id.replace('L-', '')}`;
  // Document numbers of this run only, so the rows are always new.
  const stamp = Date.now().toString(36).toUpperCase();
  const fileName = `${uniqueName('gsap-e2e').replace(' ', '-')}.csv`;
  const header =
    'type,document_number,line,po_number,po_line,posting_date,accrual_month,amount,currency,vendor_code,vendor_name,description,swbs_code,tracking_key,line_item_id';
  const rows = [
    `PO,PO${stamp},10,,,${records.year}-03-02,,6000.00,USD,V-900001,E2E Agency Ltd,Purchase order,,${hxKey},`,
    `INVOICE,IN${stamp},,PO${stamp},10,${records.year}-03-20,,2500.00,USD,V-900001,E2E Agency Ltd,Invoice,,,`,
    `INVOICE,IX${stamp},,PO${stamp},10,${records.year}-03-21,,100.00,XYZ,V-900001,E2E Agency Ltd,Invoice,,,`,
  ];

  await signInAs(page, ALEX);
  await page.goto('/financials');
  await page.getByRole('button', { name: 'Import GSAP extract' }).click();
  const dialog = page.getByRole('dialog', { name: 'Import GSAP extract' });
  await expect(dialog.getByRole('button', { name: 'Download sample file' })).toBeVisible();
  await expectNoAxeViolations(page);

  // The check saves nothing and says what the import will do, row by row.
  await dialog.getByLabel(/^GSAP extract file/).setInputFiles({
    name: fileName,
    mimeType: 'text/csv',
    buffer: Buffer.from([header, ...rows].join('\r\n') + '\r\n'),
  });
  await expect(
    dialog.getByText('3 rows: 2 new, 0 updated, 0 unchanged, 1 refused · 0 not linked'),
  ).toBeVisible();
  await expect(
    dialog.getByText(/^Row \d · currency: Currency 'XYZ' isn't set up in Settings/),
  ).toBeVisible();
  await expectNoAxeViolations(page);
  await dialog.getByRole('button', { name: 'Import 2 rows' }).click();
  await expectToast(
    page,
    `Imported '${fileName}': 2 new, 0 updated · 0 not linked. 1 row refused — fix it and import the file again.`,
  );
  await expect(page.getByText(`GSAP data as of`).first()).toContainText(`'${fileName}'`);

  await page.getByRole('button', { name: 'Import history' }).click();
  await expect(page.getByRole('dialog', { name: 'Import history' })).toContainText(fileName);
  await page.keyboard.press('Escape');

  // The PO links by HX key; the invoice by its PO number: the activity's card shows both.
  await page.goto(records.activityPath);
  await expect(page.getByRole('heading', { name: records.activityName, level: 1 })).toBeVisible();
  const card = page.getByRole('region', { name: 'Financials', exact: true });
  const figures = card.getByRole('list', { name: 'Financial figures' }).getByRole('listitem');
  await expect(figures.filter({ hasText: 'Committed (POs)' })).toContainText('6,000.00 USD');
  await expect(figures.filter({ hasText: 'Invoiced' })).toContainText('2,500.00 USD');
  await expect(figures.filter({ hasText: 'Not yet committed' })).toContainText('14,000.00 USD');
});
