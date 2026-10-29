import { expect, test, type Page } from '@playwright/test';
import {
  callApi,
  chooseOption,
  expectNoAxeViolations,
  expectToast,
  signInAs,
  uniqueName,
} from './fixtures';
import { lubricantsActivity } from './lubricants';
import { lineItemFields } from './planning-changes';

// Feature 016's journeys (plan, Tests › End-to-end) against the real API, on records of the tests' own:
// GSAP rates, a plan in GBP with its budget by month, a split in GBP, and All work over two years.
// Attribute filters, Breakdown and Field help are left to the feature that finishes them.

const ALEX = 'alex.morgan@shell.com';
const PRIYA = 'priya.raman@shell.com';
const THIS_YEAR = new Date().getUTCFullYear();
const NEXT_YEAR = THIS_YEAR + 1;

async function addLineItem(page: Page, activityId: string, amount: string, currency: string) {
  await callApi(
    page,
    'POST',
    `/api/v1/activities/${activityId}/line-items`,
    lineItemFields(uniqueName(`${currency} line item`), amount, currency),
  );
}

test('016 US1 (T089): an administrator imports GSAP rates after a review; a future rate waits for its date', async ({
  page,
}) => {
  await signInAs(page, ALEX);
  await page.goto('/settings/currencies/import');
  await expect(page.getByRole('heading', { name: 'Import GSAP rates', level: 1 })).toBeVisible();
  await expectNoAxeViolations(page);

  const fileName = `${uniqueName('gsap_rates').replace(' ', '_')}.csv`;
  const rows = [
    'rate_type,from_currency,to_currency,valid_from,rate',
    `M,USD,EUR,${NEXT_YEAR}-03-01,0.95`,
    `M,USD,GBP,${NEXT_YEAR}-03-01,0.81`,
    `M,USD,CHF,${NEXT_YEAR}-03-01,0.90`,
    `M,USD,EUR,${NEXT_YEAR}-03-02,0`,
  ];
  const chooser = page.waitForEvent('filechooser');
  await page.getByRole('button', { name: 'GSAP daily rate extract (.csv)' }).click();
  await (
    await chooser
  ).setFiles({
    name: fileName,
    mimeType: 'text/csv',
    buffer: Buffer.from(rows.join('\n')),
  });

  // Nothing is saved before the review is confirmed; every row has its outcome.
  const review = page.getByRole('region', { name: `Review of '${fileName}'` });
  const outcomes = review.getByRole('list', { name: 'Rows by outcome' });
  await expect(outcomes.getByRole('listitem').filter({ hasText: 'To add' })).toHaveText(
    /^To add\s*2$/,
  );
  await expect(outcomes.getByRole('listitem').filter({ hasText: 'Not used' })).toHaveText(
    /^Not used\s*1$/,
  );
  await expect(review.getByRole('row', { name: /^4 CHF / })).toContainText(
    'CHF is not set up in Settings › Currencies.',
  );
  await expect(review.getByRole('row', { name: /^5 EUR / })).toContainText(
    'Row 5: the rate must be above 0 with at most 6 decimals.',
  );
  await expectNoAxeViolations(page);
  await review.getByRole('button', { name: 'Import 2 rates' }).click();
  await expectToast(page, `2 exchange rates imported from '${fileName}'.`);

  // The rate of next March is in the history, while today's rate stays the current one.
  await page.goto('/settings/currencies');
  await page.getByRole('button', { name: 'EUR', exact: true }).click();
  const euro = page.getByRole('dialog', { name: 'Euro (EUR)' });
  await expect(euro).toContainText('1 USD = 0.92 EUR');
  const history = euro.getByRole('region', { name: 'Rate history' });
  const future = history.getByRole('row', { name: new RegExp(`^From 01 Mar ${NEXT_YEAR} `) });
  await expect(future).toContainText('0.95');
  await expect(future).toContainText(fileName);
  await expect(future).not.toContainText('Current');
  await expect(history.getByRole('row', { name: / Current / })).toContainText('0.92');
  await expectNoAxeViolations(page);
});

test('016 US2, US3 (T089): Priya plans in GBP with USD beside it and phases the budget by month', async ({
  page,
}) => {
  test.slow();
  await signInAs(page, PRIYA);
  const name = uniqueName('GBP plan');
  await page.goto('/plans/new');
  await page.getByRole('textbox', { name: 'Plan name' }).fill(name);
  await chooseOption(page.getByRole('combobox', { name: 'Model' }), 'Lubricants 1.0');
  await chooseOption(page.getByRole('combobox', { name: 'Plan currency' }), 'GBP');
  await page.getByRole('textbox', { name: 'Plan budget' }).fill('600000');
  await page.getByRole('checkbox', { name: 'Lubricants', exact: true }).check();
  await page.getByRole('button', { name: 'Save', exact: true }).click();
  await expect(page.getByRole('heading', { name, level: 1 })).toBeVisible();

  // The plan's amounts are in GBP with their USD value (GBP 0.79 per USD in the demo rates).
  await expect(page.getByRole('group', { name: 'Plan facts' })).toContainText('Planned in GBP');
  const summary = page.getByRole('region', { name: 'Summary' });
  await expect(summary).toContainText('600,000.00 GBP');
  await expect(summary).toContainText('≈ 759,493.67 USD');

  const card = page.getByRole('region', { name: 'Budget by month', exact: true });
  const editor = page.getByRole('dialog', { name: `Budget by month — ${name}` });
  await card.getByRole('button', { name: 'Edit budget by month' }).click();
  await editor.getByRole('radio', { name: /^Evenly/ }).check();
  await expect(editor.getByText('Each month gets 50,000.00 GBP.')).toBeVisible();
  await editor.getByRole('button', { name: 'Save budget by month' }).click();
  await expect(editor).toHaveCount(0);
  await expect(
    card.getByRole('row', {
      name: new RegExp(`^Budget (About Budget )?${'50,000\\.00 '.repeat(12)}600,000\\.00$`),
    }),
  ).toBeVisible();

  // Custom by quarter: 100,000.00 in Q1 spreads over its months, the cent going to March.
  await card.getByRole('button', { name: 'Edit budget by month' }).click();
  await editor.getByRole('radio', { name: /^Custom/ }).check();
  await editor
    .getByRole('group', { name: 'Type amounts' })
    .getByRole('button', { name: 'Quarterly' })
    .click();
  await editor.getByRole('textbox', { name: new RegExp(`^Q1 ${THIS_YEAR}`) }).fill('100000');
  await expect(
    editor.getByText('550,000.00 of 600,000.00 GBP phased — 50,000.00 left'),
  ).toBeVisible();
  await expectNoAxeViolations(page);
  await editor.getByRole('button', { name: 'Save budget by month' }).click();
  await expect(editor).toHaveCount(0);
  await expect(
    card.getByRole('row', {
      name: /^Budget (About Budget )?33,333\.33 33,333\.33 33,333\.34 50,000\.00 /,
    }),
  ).toBeVisible();
  await expect(
    card.getByText(
      'Budget by month: Adds up to 550,000.00 GBP, not the budget 600,000.00 GBP. Needed to publish.',
    ),
  ).toBeVisible();
  await expectNoAxeViolations(page);
});

test('016 US2, US4 (T089): a GBP activity converts its EUR line item and splits it off into a new activity', async ({
  page,
}) => {
  await signInAs(page, PRIYA);
  const activity = await lubricantsActivity(page, 'Split GBP', '40000.00', { currency: 'GBP' });
  await addLineItem(page, activity.id, '15000.00', 'GBP');
  await addLineItem(page, activity.id, '23000.00', 'EUR');

  // 23,000.00 EUR at 0.92 is 25,000.00 USD, which is 19,750.00 GBP at 0.79.
  await page.goto(activity.path);
  await expect(page.getByRole('heading', { name: activity.name, level: 1 })).toBeVisible();
  await expect(page.getByRole('region', { name: 'Summary' })).toContainText('5,250.00 GBP');
  await expect(
    page.getByRole('row', {
      name: /^EUR line item \S+ L-\d+ 23,000\.00 EUR 19,750\.00 GBP 25,000\.00 USD /,
    }),
  ).toBeVisible();

  await page
    .getByRole('button', { name: `Actions for ${activity.name}` })
    .first()
    .click();
  await page.getByRole('menuitem', { name: 'Split activity' }).click();
  const dialog = page.getByRole('dialog', { name: `Split '${activity.name}'` });
  const newName = uniqueName('Ireland roadshow');
  await dialog.getByRole('textbox', { name: /^New activity name/ }).fill(newName);
  await dialog.getByRole('checkbox', { name: /^EUR line item/ }).check();
  const after = dialog.getByRole('region', { name: 'After the split' });
  await expect(after).toContainText('20,250.00 GBP');
  await expect(after).toContainText('19,750.00 GBP');
  await expectNoAxeViolations(page);
  await dialog.getByRole('button', { name: 'Split activity' }).click();
  await expectToast(page, `'${activity.name}' split — '${newName}' created.`);
  await expect(page.getByText(new RegExp(`Split into '${newName}'`)).first()).toBeVisible();
});

test('016 US5 (T089): All work over two years shows Budget & cost by year and compares it with the phased budget', async ({
  page,
}) => {
  await signInAs(page, PRIYA);
  await page.goto('/all-work');
  await page.getByRole('button', { name: /^Years/ }).click();
  const years = page.getByRole('dialog', { name: 'Years' });
  await chooseOption(years.getByRole('combobox', { name: 'To' }), String(NEXT_YEAR));
  await page.keyboard.press('Escape');
  await expect(
    page.getByRole('button', { name: `Years · ${THIS_YEAR}–${NEXT_YEAR}` }),
  ).toBeVisible();

  await page.getByRole('tab', { name: 'Budget & cost' }).click();
  const period = page.getByRole('group', { name: 'Period' });
  await expect(period.getByRole('button', { name: 'Yearly' })).toHaveAttribute(
    'aria-pressed',
    'true',
  );
  const table = page.getByRole('table', { name: 'Budget & cost' });
  await expect(table.getByRole('columnheader', { name: String(THIS_YEAR) })).toBeVisible();
  await expect(table.getByRole('columnheader', { name: String(NEXT_YEAR) })).toBeVisible();
  await page.getByRole('switch', { name: 'Compare with phased budget' }).click();
  await expect(table.getByRole('row', { name: /Phased budget/ }).first()).toBeVisible();
  await expectNoAxeViolations(page);
});
