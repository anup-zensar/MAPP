import { expect, test } from '@playwright/test';
import {
  action,
  callApi,
  chooseOption,
  escapeRegExp,
  expectToast,
  signInAs,
  uniqueName,
} from './fixtures';
import { createPlanningRecords, planTree, treeRow } from './planning';

// The core flow of the Plan module (features 002–008), against the real API with the demo data.
// Tests create their own records (names from uniqueName) and never assert global counts.

const ALEX = 'alex.morgan@shell.com';
const MEI = 'mei.lin@shell.com';

test('C: Settings — an administrator edits a model, a Shell attribute and a currency', async ({
  page,
}) => {
  test.slow();
  await signInAs(page, ALEX);

  // Models (002 US1): edit a duplicate, never a demo model — other tests use the demo models by name.
  await page.goto('/settings');
  await expect(page.getByRole('heading', { name: 'Settings', level: 1 })).toBeVisible();
  await page.getByRole('link', { name: 'Models', exact: true }).click();
  await expect(page).toHaveURL(/\/settings\/models/);
  await expect(page.getByRole('heading', { name: 'Models', level: 1 })).toBeVisible();
  const modelName = uniqueName('Core model');
  await page.getByRole('button', { name: 'Actions for Lubricants 1.0' }).click();
  await page.getByRole('menuitem', { name: 'Duplicate' }).click();
  const newModel = page.getByRole('dialog', { name: 'New model' });
  const modelNameField = newModel.getByRole('textbox', { name: /^Name/ });
  await expect(modelNameField).toHaveValue('Lubricants 1.0 (copy)');
  await modelNameField.fill(modelName);
  await newModel.getByRole('button', { name: 'Save', exact: true }).click();
  await expectToast(page, `Model '${modelName}' created`);

  await page.getByRole('button', { name: `Actions for ${modelName}` }).click();
  await page.getByRole('menuitem', { name: 'Edit' }).click();
  const editModel = page.getByRole('dialog', { name: 'Edit model' });
  await editModel.getByRole('textbox', { name: /^INQ→MQL rate/ }).fill('3');
  await expect(
    editModel.getByText('250,000 INQ → 7,500 MQL → 375 deals → 3,750,000.00 USD revenue'),
  ).toBeVisible();
  // An unused model has no impact to confirm, so it saves straight away.
  await editModel.getByRole('button', { name: 'Save', exact: true }).click();
  await expectToast(page, `Model '${modelName}' saved`);
  await expect(page.getByRole('row').filter({ hasText: modelName })).toContainText(
    '3.00 % → 5.00 %',
  );

  await page.goto('/settings');
  await page.getByRole('link', { name: 'Shell attributes', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Shell attributes', level: 1 })).toBeVisible();
  await page.getByRole('button', { name: /^Line item/ }).click();
  await page.getByRole('link', { name: 'Contracted spend', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Contracted spend', level: 1 })).toBeVisible();
  await action(page, 'Edit').click();
  await expect(
    page.getByRole('heading', { name: "Edit Shell attribute 'Contracted spend'", level: 1 }),
  ).toBeVisible();
  const helpText = `Whether the spend is under contract (${uniqueName('core flow')}).`;
  await page.getByRole('textbox', { name: /^Help text/ }).fill(helpText);
  await page.getByRole('button', { name: 'Save', exact: true }).click();
  await expectToast(page, "Shell attribute 'Contracted spend' saved");
  await expect(page.getByRole('heading', { name: 'Contracted spend', level: 1 })).toBeVisible();
  await expect(page.getByText(helpText).first()).toBeVisible();

  // Currencies (002 US4): INR, which no demo line item uses; put back to 83.50 for the tests that follow.
  await page.goto('/settings');
  await page.getByRole('link', { name: 'Currencies', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Currencies', level: 1 })).toBeVisible();
  const inrRow = page.getByRole('row').filter({ hasText: 'INR' });
  await expect(inrRow).toContainText('1 USD = 83.50 INR');
  await inrRow.getByRole('button', { name: 'Actions for INR' }).click();
  await page.getByRole('menuitem', { name: 'Edit' }).click();
  const editCurrency = page.getByRole('dialog', { name: 'Edit currency' });
  await expect(
    editCurrency.getByText("The code can't change once the currency is saved."),
  ).toBeVisible();
  await editCurrency.getByRole('textbox', { name: /^Rate/ }).fill('84');
  await expect(editCurrency.getByText('1 USD = 84.00 INR')).toBeVisible();
  await expect(editCurrency.getByText('1 INR = 0.011905 USD')).toBeVisible();
  await editCurrency.getByRole('button', { name: 'Save', exact: true }).click();
  await expectToast(page, "Currency 'Indian rupee' saved");
  await expect(inrRow).toContainText('1 USD = 84.00 INR');

  const inr = await callApi<{ version: number }>(page, 'GET', '/api/v1/currencies/INR');
  await callApi(page, 'PATCH', '/api/v1/currencies/INR', { rate: '83.5', version: inr.version });
});

test('D: the plan form offers only members of the chosen teams as owner and refuses an owner outside them', async ({
  page,
}) => {
  // Alex Morgan, whose role lets him give a plan any team (014 FR-013); his team is Global Marketing Ops.
  await signInAs(page, ALEX);
  await page.goto('/plans/new');
  await expect(page.getByRole('heading', { name: 'New plan', level: 1 })).toBeVisible();
  const teams = page.getByRole('combobox', { name: /^Teams/ });
  const owner = page.getByRole('combobox', { name: /^Owner/ });

  await expect(page.getByText('Default: your teams')).toBeVisible();
  await expect(page.getByText('Default: you', { exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Remove Global Marketing Ops' })).toBeVisible();

  const owners = page.getByRole('listbox', { name: 'Owner' });
  await owner.click();
  await owner.press('ArrowDown');
  await expect(owners.getByRole('option').first()).toHaveAccessibleName(/^Alex Morgan \(you\)/);
  await expect(owners.getByRole('option', { name: /^Carlos Mendes/ })).toBeVisible();
  await expect(owners.getByRole('option', { name: /^Lea Santos/ })).toBeVisible();
  await expect(owners.getByRole('option', { name: /^Mei Lin/ })).toHaveCount(0);
  await expect(page.getByText('3 people').first()).toBeVisible();
  await owner.fill('mei');
  await expect(
    page.getByText(
      'No members of the selected teams match. Add people to these teams in Settings › Teams.',
    ),
  ).toBeVisible();
  await owner.press('Escape');

  // Swap the team for Mobility China: Alex is not one of its members (R1, 004 FR-021).
  await teams.fill('Mobility');
  await page.getByRole('option', { name: /^Mobility China/ }).click();
  await teams.press('Escape');
  await page.getByRole('button', { name: 'Remove Global Marketing Ops' }).click();
  const notInTeams =
    "Alex Morgan isn't in any of the chosen teams — choose another owner or clear the field";
  await expect(page.getByText(notInTeams).first()).toBeVisible();

  // Saving anyway is refused by the API with the same rule, listed in the error summary.
  const planName = uniqueName('Owner rules plan');
  await page.getByRole('textbox', { name: /^Plan name/ }).fill(planName);
  await chooseOption(page.getByLabel(/^Model/), 'Mobility 1.0');
  await page.getByRole('checkbox', { name: 'Mobility', exact: true }).check();
  await page.getByRole('button', { name: 'Save', exact: true }).click();
  await expect(
    page.getByRole('heading', { name: /problems? needs? your attention/ }),
  ).toBeVisible();
  await expect(
    page.getByRole('link', { name: new RegExp(`^${escapeRegExp(notInTeams)}`) }),
  ).toBeVisible();
  await expect(page).toHaveURL(/\/plans\/new$/);

  await page.getByRole('button', { name: 'Remove Mobility China' }).click();
  await expect(owner).toHaveAttribute('aria-disabled', 'true');
  await expect(
    page.getByText("Choose teams first — the owner must belong to one of the plan's teams").first(),
  ).toBeVisible();

  await teams.fill('Mobility');
  await page.getByRole('option', { name: /^Mobility China/ }).click();
  await teams.press('Escape');
  await owner.fill('finance');
  await expect(owners.getByRole('option', { name: /^Sara Novak/ })).toBeVisible();
  await owner.fill('Mei');
  await owners.getByRole('option', { name: /^Mei Lin/ }).click();
  await page.getByRole('button', { name: 'Save', exact: true }).click();

  await expectToast(page, `Plan '${planName}' created — add campaigns next.`);
  await expect(page).toHaveURL(/\/plans\/P-\d+$/);
  await expect(page.getByRole('heading', { name: planName, level: 1 })).toBeVisible();
  await expect(
    page.getByRole('group', { name: 'Plan facts' }).getByText('Mei Lin', { exact: true }),
  ).toBeVisible();
  await expect(page.getByText('Mobility China').first()).toBeVisible();
});

test('E: deleting an activity moves it to the recycle bin with its cost, and Undo brings it back', async ({
  page,
}) => {
  await signInAs(page, MEI);
  const records = await createPlanningRecords(page, 'Delete', {
    withWorkflow: false,
    leaveEmpty: [],
  });
  const name = records.activityName;

  await page.goto(records.activityPath);
  await expect(page.getByRole('heading', { name, level: 1 })).toBeVisible();
  await page.getByRole('button', { name: `Actions for ${name}` }).click();
  await page.getByRole('menuitem', { name: 'Delete' }).click();
  const dialog = page.getByRole('alertdialog', { name: `Delete activity '${name}'?` });
  await expect(
    dialog.getByText('The initiative, campaign and plan planned costs go down by 20,000.00 USD.'),
  ).toBeVisible();
  await expect(dialog.getByText('It moves to the recycle bin; you can restore it.')).toBeVisible();
  await expect(dialog.getByRole('button', { name: 'Cancel' })).toBeFocused();
  await dialog.getByRole('button', { name: 'Delete activity' }).click();

  await expect(page).toHaveURL(new RegExp(`/plans/${records.planId}$`));
  await expectToast(page, `Activity '${name}' moved to the recycle bin`);
  await page.getByRole('button', { name: 'Expand all' }).click();
  await expect(treeRow(page, records.initiativeName)).toBeVisible();
  await expect(treeRow(page, name)).toHaveCount(0);
  await expect(treeRow(page, records.initiativeName)).not.toContainText('$20,000.00');

  // Undo within the toast's 10 seconds brings it back with the same ID (001 FR-026).
  await page.getByRole('button', { name: 'Undo' }).click();
  await expectToast(page, `Activity '${name}' restored`);
  await page.getByRole('button', { name: 'Expand all' }).click();
  await expect(planTree(page).getByRole('link', { name, exact: true })).toHaveAttribute(
    'href',
    new RegExp(`/activities/${records.activityId}$`),
  );
  await expect(treeRow(page, records.initiativeName)).toContainText('$20,000.00');
});
