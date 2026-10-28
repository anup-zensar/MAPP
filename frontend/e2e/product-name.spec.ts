import { expect, test, type Locator, type Page } from '@playwright/test';
import { expectNoAxeViolations, signInAs } from './fixtures';

const ALEX = 'alex.morgan@shell.com';
const PAGES = [
  '/',
  '/plans',
  '/budgets',
  '/approvals',
  '/all-work',
  '/financials',
  '/settings',
  '/recycle-bin',
  '/activity-log',
  '/profile',
];

function productName(scope: Page | Locator): Locator {
  return scope.getByRole('img', { name: 'SP&A One', exact: true });
}

async function colourOf(name: Locator, part: string): Promise<string> {
  const span = name.locator('span').filter({ hasText: new RegExp(`^${part}$`) });
  return span.evaluate((element) => getComputedStyle(element).color);
}

async function expectNameColours(name: Locator, amp: string, one: string) {
  await expect(name).toBeVisible();
  // The system theme reaches the page after emulateMedia, so the colours are awaited, not read once.
  await expect.poll(() => colourOf(name, '&')).toBe(amp);
  await expect.poll(() => colourOf(name, 'One')).toBe(one);
}

test('035 US1: the product is called SP&A One on sign-in and every page of the navigation (FR-001, FR-002)', async ({
  page,
}) => {
  await page.goto('/sign-in');
  await expect(page.getByRole('heading', { name: 'Sign in to SP&A One', level: 1 })).toBeVisible();
  await expect(page).toHaveTitle('Sign in · SP&A One');
  await expect(productName(page.getByRole('banner'))).toBeVisible();
  await expect(page.locator('body')).not.toContainText('MAPP');

  await signInAs(page, ALEX);
  await page.goto('/');
  await expect(page).toHaveTitle('Home · SP&A One');
  await expect(page.getByRole('contentinfo')).toHaveText('© SP&A One — demo data');
  await expect(page.getByRole('link', { name: 'SP&A One home' })).toHaveAttribute('href', '/');

  for (const path of PAGES) {
    await page.goto(path);
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
    await expect(page).toHaveTitle(/ · SP&A One$/);
    await expect(page.locator('body')).not.toContainText('MAPP');
  }
});

test('035 US2: the name is the logo in Shell blue and yellow, with the "1" tile on narrow screens (FR-003 – FR-005)', async ({
  page,
}) => {
  await page.goto('/sign-in');
  const signInName = productName(page.getByRole('banner'));
  await expectNameColours(signInName, 'rgb(0, 60, 136)', 'rgb(173, 135, 0)');
  const title = page.getByRole('heading', { name: 'Sign in to SP&A One', level: 1 });
  await expectNameColours(productName(title), 'rgb(0, 60, 136)', 'rgb(173, 135, 0)');
  await expectNoAxeViolations(page);

  await page.emulateMedia({ colorScheme: 'dark' });
  await expectNameColours(signInName, 'rgb(111, 155, 209)', 'rgb(255, 198, 0)');
  await expectNoAxeViolations(page);

  await page.emulateMedia({ colorScheme: 'light' });
  await signInAs(page, ALEX);
  await page.goto('/');
  const home = page.getByRole('link', { name: 'SP&A One home' });
  await expectNameColours(productName(home), 'rgb(0, 60, 136)', 'rgb(173, 135, 0)');
  await expectNoAxeViolations(page);

  await page.emulateMedia({ colorScheme: 'dark' });
  await expectNameColours(productName(home), 'rgb(111, 155, 209)', 'rgb(255, 198, 0)');
  await expectNoAxeViolations(page);

  await page.setViewportSize({ width: 800, height: 900 });
  await expect(productName(home)).toBeHidden();
  await expect(home.locator('svg')).toBeVisible();
  await page.getByRole('button', { name: 'Open menu' }).click();
  await expect(productName(page.getByRole('dialog'))).toBeVisible();
});
