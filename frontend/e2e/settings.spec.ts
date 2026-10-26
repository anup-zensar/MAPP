import { expect, test } from '@playwright/test';
import { expectNoAxeViolations, expectToast, signInAs, uniqueName } from './fixtures';

// The Settings screens of 002 US3, US5 and US6, against the real API with the demo data.

const ALEX = 'alex.morgan@shell.com';

/** The current year: the default year of a new finance budget item, and the year All work marks Today in. */
const THIS_YEAR = new Date().getFullYear();

test('Settings: an activity type, a finance budget item, a dependent option in the preview and an option import (002 US3, US5, US6)', async ({
  page,
}) => {
  test.slow();
  await signInAs(page, ALEX);

  // Activity types (US5).
  await page.goto('/settings');
  await page.getByRole('link', { name: 'Activity types', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Activity types', level: 1 })).toBeVisible();
  await expect(page.getByText('The kinds of work activities can be.')).toBeVisible();
  await expectNoAxeViolations(page);
  const typeName = uniqueName('E2E type');
  await page.getByRole('button', { name: 'New activity type' }).click();
  const newType = page.getByRole('dialog', { name: 'New activity type' });
  await newType.getByRole('textbox', { name: /^Name/ }).fill(typeName);
  await newType.getByRole('textbox', { name: /^Default planned cost/ }).fill('12500');
  await newType.getByRole('checkbox', { name: 'Mobility 1.0' }).check();
  await newType.getByRole('button', { name: 'Save', exact: true }).click();
  await expectToast(page, `Activity type '${typeName}' created`);
  const typeRow = page.getByRole('row').filter({ hasText: typeName });
  await expect(typeRow).toContainText('$12,500.00');
  await expect(typeRow).toContainText('Mobility 1.0');
  // Not used by any activity yet; its usage names the models that allow it (002 plan, Usage parts).
  await expect(typeRow).toContainText('Allowed in 1 model');
  await expect(typeRow).not.toContainText('Used by');
  await expect(typeRow).toContainText('Active');

  // Finance budget items (US5): a new item in a new budget of this year.
  await page.goto('/settings');
  await page.getByRole('link', { name: 'Finance budget items', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Finance budget items', level: 1 })).toBeVisible();
  const budgetName = uniqueName('E2E budget');
  const itemName = uniqueName('Europe – Retail');
  const code = uniqueName('E2E').replace(' ', '-').toUpperCase();
  await page.getByRole('button', { name: 'New finance budget item' }).click();
  const newItem = page.getByRole('dialog', { name: 'New finance budget item' });
  await expect(newItem.getByRole('combobox', { name: 'Budget year' })).toHaveValue(
    String(THIS_YEAR),
  );
  await expect(
    newItem.getByText('Items with the same year and budget name form one finance budget.'),
  ).toBeVisible();
  await newItem.getByRole('combobox', { name: 'Budget name' }).fill(budgetName);
  await newItem.getByRole('textbox', { name: /^Item name/ }).fill(itemName);
  await newItem.getByRole('textbox', { name: /^Code/ }).fill(code);
  await newItem.getByRole('button', { name: 'Save', exact: true }).click();
  await expectToast(page, `Finance budget item '${itemName}' created`);
  const newBudget = page.getByRole('region', { name: budgetName });
  await expect(newBudget).toContainText('1 item');
  const itemRow = newBudget.getByRole('row').filter({ hasText: itemName });
  await expect(itemRow).toContainText(code);
  await expect(itemRow).toContainText('Not used yet');

  // A dependent option (US3): Region gets an option for the cluster Retail – India, and the preview
  // fills it in as that cluster's only region.
  await page.goto('/settings/shell-attributes?level=campaign');
  await page.getByRole('link', { name: 'Region', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Region', level: 1 })).toBeVisible();
  const regionLabel = uniqueName('RETAIL – INDIA').toUpperCase();
  await page.getByRole('button', { name: 'Add option' }).click();
  const addOption = page.getByRole('dialog', { name: 'Add option' });
  await addOption.getByRole('textbox', { name: /^Label/ }).fill(regionLabel);
  await expect(
    addOption.getByText('Shown when one of these cluster options is chosen.'),
  ).toBeVisible();
  await addOption.getByRole('checkbox', { name: 'Retail – India' }).check();
  await addOption.getByRole('button', { name: 'Save', exact: true }).click();
  await expectToast(page, `Option '${regionLabel}' added`);
  await expect(
    page.getByRole('region', { name: 'Options' }).getByRole('row').filter({ hasText: regionLabel }),
  ).toContainText('Retail – India');

  const preview = page.getByRole('region', { name: 'Preview — Campaign form' });
  await expect(
    preview.getByText('Behaves like the real form. Nothing you type here is saved.'),
  ).toBeVisible();
  await preview.getByRole('checkbox', { name: 'Retail – India' }).check();
  await expect(
    preview.getByText(`Region set to '${regionLabel}', the only region for this cluster.`),
  ).toBeVisible();
  await expect(preview.getByRole('checkbox', { name: regionLabel })).toBeChecked();

  // Import options from a small file (US6): two new rows, one already present, one twice in the file.
  await page.goto('/settings/shell-attributes');
  // The level filter is 002's SegmentedFilter: buttons named "<level> <count>".
  await page.getByRole('button', { name: /^Line item/ }).click();
  await page.getByRole('link', { name: 'Spend category', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Spend category', level: 1 })).toBeVisible();
  const first = uniqueName('Imported category A');
  const second = uniqueName('Imported category B');
  await page.getByRole('button', { name: 'Import from a spreadsheet' }).click();
  const importing = page.getByRole('dialog', { name: 'Import options into Spend category' });
  await expect(
    importing.getByText(
      'Download the template, fill one row per option, then upload it. Nothing is added until you confirm.',
    ),
  ).toBeVisible();
  const choosing = page.waitForEvent('filechooser');
  await importing.getByRole('button', { name: 'Choose a file' }).click();
  await (
    await choosing
  ).setFiles({
    name: 'spend categories.csv',
    mimeType: 'text/csv',
    buffer: Buffer.from(`Option\n${first}\nInternal\n${second}\n${first}\n`),
  });
  await expect(importing.getByText('2 options to add').first()).toBeVisible();
  await expect(importing.getByText('1 already present (skipped)').first()).toBeVisible();
  await expect(importing.getByText('1 row with problems').first()).toBeVisible();
  await expect(importing.getByText('Appears twice in the file (also row 2).')).toBeVisible();
  await importing.getByRole('button', { name: 'Add 2 options' }).click();
  await expectToast(page, '2 options added to Spend category');
  const options = page.getByRole('region', { name: 'Options' });
  await expect(options.getByRole('row').filter({ hasText: first })).toContainText('Active');
  await expect(options.getByRole('row').filter({ hasText: second })).toContainText('Active');
});
