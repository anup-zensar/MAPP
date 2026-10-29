import { expect, test } from '@playwright/test';
import { DEMO_PASSWORD, expectNoAxeViolations, signInAs, uniqueName } from './fixtures';

test('US1: a visitor is sent to sign in and comes back after choosing a demo person (FR-001–FR-005)', async ({
  page,
}) => {
  await page.goto('/plans');

  await expect(page).toHaveURL(/\/sign-in\?next=%2Fplans$/);
  await expect(page).toHaveTitle('Sign in · SP&A One');
  await expect(page.getByRole('heading', { name: 'Sign in to SP&A One' })).toBeVisible();
  await expect(
    page.getByText('Demo environment — sign-in and data are for demonstration only'),
  ).toBeVisible();
  await expect(page.getByText('Demo sign-in — Shell single sign-on comes later')).toBeVisible();

  // The browser may put a space before the comma of this name, so it is matched loosely.
  await page.getByRole('button', { name: /^Sign in as Priya Raman\s*,\s*Marketer$/ }).click();

  await expect(page).toHaveURL(/\/plans$/);
  await expect(page).toHaveTitle('Plans · SP&A One');
  await expect(page.getByRole('heading', { name: 'Plans', level: 1 })).toBeVisible();
});

test('US1: wrong password is refused; e-mail case and spaces are ignored (FR-003, FR-004)', async ({
  page,
}) => {
  await page.goto('/sign-in');

  await page.getByRole('button', { name: 'Sign in', exact: true }).click();
  await expect(page.getByText('Enter your e-mail').first()).toBeVisible();
  await expect(page.getByText('Enter the password').first()).toBeVisible();

  await page.getByLabel('E-mail').fill('  PRIYA.RAMAN@shell.com  ');
  await page.getByLabel('Password', { exact: true }).fill('wrong');
  await page.getByRole('button', { name: 'Sign in', exact: true }).click();
  await expect(page.getByText('Incorrect e-mail or password.')).toBeVisible();
  // The e-mail stays in the field (the browser itself trims the spaces around an e-mail value).
  await expect(page.getByLabel('E-mail')).toHaveValue('PRIYA.RAMAN@shell.com');

  await page.getByLabel('Password', { exact: true }).fill(DEMO_PASSWORD);
  await page.getByRole('button', { name: 'Sign in', exact: true }).click();

  await expect(page).toHaveURL(/\/$/);
  await expect(page.getByRole('heading', { name: 'Welcome back, Priya.' })).toBeVisible();
  // Priya's teams have plans, so Home shows them at a glance instead of Getting started.
  await expect(page.getByRole('region', { name: 'Plans at a glance' })).toBeVisible();
});

test('US2: every navigation item opens its page with its tab title (FR-010, FR-015, FR-017, SC-003)', async ({
  page,
}) => {
  // Global budgets and Financials are shown only to people whose role includes them, such as Alex.
  await signInAs(page, 'alex.morgan@shell.com');
  await page.goto('/');
  const nav = page.getByRole('navigation', { name: 'Main' });
  const main = page.getByRole('main');

  const pages: { item: string; heading: string }[] = [
    { item: 'Plans', heading: 'Plans' },
    { item: 'Global budgets', heading: 'Global budgets' },
    { item: 'Approvals', heading: 'Approvals' },
    { item: 'All work', heading: 'All work' },
    { item: 'Settings', heading: 'Settings' },
    { item: 'Financials', heading: 'Financials' },
    { item: 'Recycle bin', heading: 'Recycle bin' },
    { item: 'Activity log', heading: 'Activity log' },
    { item: 'Home', heading: 'Welcome back, Alex.' },
  ];
  for (const { item, heading } of pages) {
    await nav.getByRole('link', { name: item }).click();
    await expect(page).toHaveTitle(`${item} · SP&A One`);
    await expect(nav.getByRole('link', { name: item })).toHaveAttribute('aria-current', 'page');
    await expect(main.getByRole('heading', { name: heading, level: 1 })).toBeVisible();
    await expect(main.getByText("This part of SP&A One isn't built yet.")).toHaveCount(0);
  }

  await page.goto('/nowhere');
  await expect(page.getByRole('heading', { name: 'Page not found' })).toBeVisible();
});

test('US4: sign out returns to the sign-in page and Back does not show the previous page (FR-006, FR-012)', async ({
  page,
}) => {
  await signInAs(page, 'mei.lin@shell.com');
  await page.goto('/');
  await expect(page.getByRole('heading', { name: 'Welcome back, Mei.' })).toBeVisible();
  // Open another page inside the app first, so that Back has an app page to go back to.
  await page.getByRole('navigation', { name: 'Main' }).getByRole('link', { name: 'Plans' }).click();
  await expect(page).toHaveTitle('Plans · SP&A One');

  await page.getByRole('button', { name: 'Mei — account menu' }).click();
  await expect(page.getByText('mei.lin@shell.com')).toBeVisible();
  await page.getByRole('menuitem', { name: 'Sign out' }).click();

  await expect(page).toHaveURL(/\/sign-in$/);
  await expect(page.getByText('You have been signed out.')).toBeVisible();

  await page.goBack();
  await expect(page).toHaveURL(/\/sign-in/);
  await expect(page.getByRole('heading', { name: 'Welcome back, Mei.' })).toHaveCount(0);
});

test('US5: create, rename, delete and Undo a sample folder; the activity log records each change (FR-022–FR-026)', async ({
  page,
}) => {
  await signInAs(page, 'priya.raman@shell.com');
  await page.goto('/dev/sample-records');
  await expect(page.getByRole('heading', { name: 'Sample records', level: 1 })).toBeVisible();

  const name = uniqueName('Smoke');
  const renamed = `${name} renamed`;

  await page.getByRole('button', { name: 'New sample folder' }).first().click();
  await page.getByLabel('Name').fill(name);
  await page.getByRole('button', { name: 'Save' }).click();
  await expect(page.getByText(`Sample folder '${name}' created`).first()).toBeVisible();

  await page.getByRole('button', { name: `Actions for ${name}` }).click();
  await page.getByRole('menuitem', { name: 'Edit' }).click();
  await page.getByLabel('Name').fill(renamed);
  await page.getByRole('button', { name: 'Save' }).click();
  await expect(page.getByText(`Sample folder '${renamed}' saved`).first()).toBeVisible();

  await page.getByRole('button', { name: `Actions for ${renamed}` }).click();
  await page.getByRole('menuitem', { name: 'Delete' }).click();
  await page.getByRole('button', { name: 'Delete sample folder' }).click();
  await expect(
    page.getByText(`Sample folder '${renamed}' moved to the recycle bin`).first(),
  ).toBeVisible();
  await expect(page.getByRole('button', { name: `Actions for ${renamed}` })).toHaveCount(0);

  await page.getByRole('button', { name: 'Undo' }).click();
  await expect(page.getByText(`Sample folder '${renamed}' restored`).first()).toBeVisible();
  await expect(page.getByRole('button', { name: `Actions for ${renamed}` })).toBeVisible();

  const log = page.getByRole('region', { name: 'Activity log' });
  await expect(log.getByText(renamed).first()).toBeVisible();
  await expect(log.getByText(/restored/).first()).toBeVisible();
  await expect(log.getByText(/deleted/).first()).toBeVisible();
});

test('SC-006: the sign-in page and Home pass the automated WCAG 2.1 AA check', async ({ page }) => {
  await page.goto('/sign-in');
  await expect(page.getByRole('heading', { name: 'Sign in to SP&A One' })).toBeVisible();
  await expect(page.getByRole('button', { name: /Sign in as/ }).first()).toBeVisible();
  await expectNoAxeViolations(page);

  await signInAs(page, 'sara.novak@shell.com');
  await page.goto('/');
  await expect(page.getByRole('heading', { name: 'Welcome back, Sara.' })).toBeVisible();
  await expectNoAxeViolations(page);
});
