import { expect, test, type Page } from '@playwright/test';
import { DEMO_PASSWORD, expectNoAxeViolations, signInAs, uniqueName } from './fixtures';
import { chooseMenuItem, tabTo } from './keyboard';

// Feature 001 SC-006 against the real API: axe (WCAG 2.2 A/AA) on every state of the app shell in light
// and dark mode, and the keyboard-only journeys of the plan's accessibility row (T080).

const PRIYA = 'priya.raman@shell.com';
const DEMO_PEOPLE = '**/api/v1/auth/demo-people';

async function openSignIn(page: Page): Promise<void> {
  await page.goto('/sign-in');
  await expect(page.getByRole('heading', { name: 'Sign in to SP&A One' })).toBeVisible();
}

for (const colorScheme of ['light', 'dark'] as const) {
  test.describe(`${colorScheme} mode`, () => {
    test.use({ colorScheme });

    test(`T080 SC-006: every state of the sign-in page and Not found signed out pass axe (${colorScheme})`, async ({
      page,
    }) => {
      await openSignIn(page);
      await expect(page.locator('html')).toHaveAttribute('data-theme', colorScheme);
      await expect(page.getByRole('button', { name: /^Sign in as Priya Raman/ })).toBeVisible();
      await expectNoAxeViolations(page);

      // Both fields empty: the error summary lists every problem at once.
      await page.getByRole('button', { name: 'Sign in', exact: true }).click();
      await expect(page.getByText('2 problems need your attention')).toBeVisible();
      await expectNoAxeViolations(page);

      await page.getByLabel('E-mail').fill(PRIYA);
      await page.getByLabel('Password', { exact: true }).fill('wrong');
      await page.getByRole('button', { name: 'Sign in', exact: true }).click();
      await expect(page.getByText('Incorrect e-mail or password.')).toBeVisible();
      await expectNoAxeViolations(page);

      // The API's other answers to the demo-people question, given in the browser (FR-009, FR-028).
      await page.route(DEMO_PEOPLE, (route) =>
        route.fulfill({ json: { people: [], password: DEMO_PASSWORD } }),
      );
      await openSignIn(page);
      await expect(page.getByText('No demo users found — load the demo data first')).toBeVisible();
      await expectNoAxeViolations(page);

      await page.unroute(DEMO_PEOPLE);
      await page.route(DEMO_PEOPLE, (route) =>
        route.fulfill({
          status: 404,
          json: { error: { code: 'not_found', message: 'Demo sign-in is off.' } },
        }),
      );
      await openSignIn(page);
      await expect(
        page.getByText('Demo sign-in is turned off in this environment.', { exact: false }),
      ).toBeVisible();
      await expectNoAxeViolations(page);

      await page.unroute(DEMO_PEOPLE);
      await page.route(DEMO_PEOPLE, (route) => route.abort());
      await openSignIn(page);
      await expect(page.getByText(/can't reach its server right now/).first()).toBeVisible();
      await expect(page.getByRole('button', { name: 'Try again' })).toBeVisible();
      await expectNoAxeViolations(page);
      await page.unroute(DEMO_PEOPLE);

      await page.goto('/nowhere');
      await expect(page.getByRole('heading', { name: 'Page not found' })).toBeVisible();
      await expect(page.getByRole('navigation', { name: 'Main' })).toHaveCount(0);
      await expectNoAxeViolations(page);
    });

    test(`T080 SC-006: Home, Not found, the menus, the collapsed sidebar and the phone drawer pass axe (${colorScheme})`, async ({
      page,
    }) => {
      await signInAs(page, PRIYA);
      await page.goto('/');
      await expect(page.getByRole('heading', { name: 'Welcome back, Priya.' })).toBeVisible();
      await expect(page.locator('html')).toHaveAttribute('data-theme', colorScheme);
      await expectNoAxeViolations(page);

      await page.getByRole('button', { name: 'Priya — account menu' }).click();
      await expect(page.getByRole('menuitem', { name: 'Sign out' })).toBeVisible();
      await expectNoAxeViolations(page);
      await page.keyboard.press('Escape');
      await expect(page.getByRole('menu')).toHaveCount(0);

      await page.getByRole('button', { name: /^Notifications/ }).click();
      await expect(page.getByRole('dialog', { name: 'Notifications' })).toBeVisible();
      await expectNoAxeViolations(page);
      await page.keyboard.press('Escape');
      await expect(page.getByRole('dialog', { name: 'Notifications' })).toHaveCount(0);

      await page.getByRole('button', { name: 'Collapse', exact: true }).click();
      await expect(page.getByRole('button', { name: 'Expand sidebar' })).toBeVisible();
      await expectNoAxeViolations(page);
      await page.getByRole('button', { name: 'Expand sidebar' }).click();
      await expect(page.getByRole('button', { name: 'Collapse', exact: true })).toBeVisible();

      await page.goto('/nowhere');
      await expect(page.getByRole('heading', { name: 'Page not found' })).toBeVisible();
      await expect(page.getByRole('navigation', { name: 'Main' })).toBeVisible();
      await expectNoAxeViolations(page);

      await page.setViewportSize({ width: 390, height: 844 });
      await page.goto('/');
      await page.getByRole('button', { name: 'Open menu' }).click();
      const drawer = page.getByRole('dialog');
      await expect(drawer.getByRole('link', { name: 'Plans' })).toBeVisible();
      await expectNoAxeViolations(page);
    });

    test(`T080 SC-006: the sample page's panel and its errors, the confirm dialog, the Undo message and the gallery pass axe (${colorScheme})`, async ({
      page,
    }) => {
      await signInAs(page, PRIYA);
      await page.goto('/dev/sample-records');
      await expect(page.getByRole('heading', { name: 'Sample records', level: 1 })).toBeVisible();
      await expectNoAxeViolations(page);

      const name = uniqueName('Axe folder');
      await page.getByRole('button', { name: 'New sample folder' }).first().click();
      const panel = page.getByRole('dialog', { name: 'New sample folder' });
      await panel.getByRole('button', { name: 'Save', exact: true }).click();
      await expect(panel.getByText('1 problem needs your attention')).toBeVisible();
      await expectNoAxeViolations(page);
      await panel.getByLabel('Name').fill(name);
      await panel.getByRole('button', { name: 'Save', exact: true }).click();
      await expect(page.getByText(`Sample folder '${name}' created`).first()).toBeVisible();

      await page.getByRole('button', { name: `Actions for ${name}` }).click();
      await page.getByRole('menuitem', { name: 'Delete' }).click();
      const confirm = page.getByRole('alertdialog');
      await expect(confirm.getByRole('button', { name: 'Delete sample folder' })).toBeVisible();
      await expectNoAxeViolations(page);
      await confirm.getByRole('button', { name: 'Delete sample folder' }).click();
      await expect(page.getByRole('button', { name: 'Undo' })).toBeVisible();
      await expectNoAxeViolations(page);
      await page.getByRole('button', { name: 'Undo' }).click();
      await expect(page.getByText(`Sample folder '${name}' restored`).first()).toBeVisible();

      await page.goto('/dev/components');
      await expect(
        page.getByRole('heading', { name: 'Shared building blocks', level: 1 }),
      ).toBeVisible();
      await expectNoAxeViolations(page);
    });
  });
}

test('T080 SC-006: keyboard only — sign in, every sidebar item, collapse and expand, theme, bell, account menu and Sign out', async ({
  page,
}) => {
  test.slow();
  await openSignIn(page);
  await tabTo(page, page.getByLabel('E-mail'));
  await page.keyboard.type(PRIYA);
  await tabTo(page, page.getByLabel('Password', { exact: true }));
  await page.keyboard.type(DEMO_PASSWORD);
  await page.keyboard.press('Enter');
  await expect(page.getByRole('heading', { name: 'Welcome back, Priya.' })).toBeVisible();
  // signInAs resets the preferences the keyboard journey changes (theme, sidebar).
  await signInAs(page, PRIYA);
  await page.goto('/');

  const nav = page.getByRole('navigation', { name: 'Main' });
  const links = nav.getByRole('link');
  await expect(links.first()).toBeVisible();
  const count = await links.count();
  expect(count).toBeGreaterThan(5);
  for (let index = 0; index < count; index++) {
    const link = links.nth(index);
    await tabTo(page, link);
    await page.keyboard.press('Enter');
    await expect(link).toHaveAttribute('aria-current', 'page');
    await expect(page.getByRole('main').getByRole('heading', { level: 1 })).toBeVisible();
  }

  await tabTo(page, page.getByRole('button', { name: 'Collapse', exact: true }));
  await page.keyboard.press('Enter');
  const expand = page.getByRole('button', { name: 'Expand sidebar' });
  await expect(expand).toBeVisible();
  await tabTo(page, expand);
  await page.keyboard.press('Enter');
  await expect(page.getByRole('button', { name: 'Collapse', exact: true })).toBeVisible();

  const html = page.locator('html');
  const dark = page.getByRole('button', { name: 'Switch to dark mode' });
  await tabTo(page, dark);
  await page.keyboard.press('Enter');
  await expect(html).toHaveAttribute('data-theme', 'dark');
  await tabTo(page, page.getByRole('button', { name: 'Switch to light mode' }));
  await page.keyboard.press('Enter');
  await expect(html).toHaveAttribute('data-theme', 'light');

  const bell = page.getByRole('button', { name: /^Notifications/ });
  await tabTo(page, bell);
  await page.keyboard.press('Enter');
  await expect(page.getByRole('dialog', { name: 'Notifications' })).toBeVisible();
  await page.keyboard.press('Escape');
  await expect(page.getByRole('dialog', { name: 'Notifications' })).toHaveCount(0);
  await expect(bell).toBeFocused();

  await tabTo(page, page.getByRole('button', { name: 'Priya — account menu' }));
  await page.keyboard.press('Enter');
  await chooseMenuItem(page, 'Sign out');
  await expect(page).toHaveURL(/\/sign-in$/);
  await expect(page.getByText('You have been signed out.')).toBeVisible();
});

test('T080 SC-006: keyboard only on the sample page — create, rename, delete and reach Undo with F8 within 10 s', async ({
  page,
}) => {
  await signInAs(page, PRIYA);
  await page.goto('/dev/sample-records');
  await expect(page.getByRole('heading', { name: 'Sample records', level: 1 })).toBeVisible();
  const name = uniqueName('Keyboard folder');
  const renamed = `${name} edited`;

  await tabTo(page, page.getByRole('button', { name: 'New sample folder' }).first());
  await page.keyboard.press('Enter');
  const panel = page.getByRole('dialog', { name: 'New sample folder' });
  await tabTo(page, panel.getByLabel('Name'));
  await page.keyboard.type(name);
  await page.keyboard.press('Enter');
  await expect(page.getByText(`Sample folder '${name}' created`).first()).toBeVisible();

  await tabTo(page, page.getByRole('button', { name: `Actions for ${name}` }));
  await page.keyboard.press('Enter');
  await chooseMenuItem(page, 'Edit');
  const edit = page.getByRole('dialog', { name: 'Edit sample folder' });
  await tabTo(page, edit.getByLabel('Name'));
  await page.keyboard.press('ControlOrMeta+a');
  await page.keyboard.type(renamed);
  await page.keyboard.press('Enter');
  await expect(page.getByText(`Sample folder '${renamed}' saved`).first()).toBeVisible();

  await tabTo(page, page.getByRole('button', { name: `Actions for ${renamed}` }));
  await page.keyboard.press('Enter');
  await chooseMenuItem(page, 'Delete');
  await tabTo(page, page.getByRole('button', { name: 'Delete sample folder' }));
  await page.keyboard.press('Enter');
  await expect(
    page.getByText(`Sample folder '${renamed}' moved to the recycle bin`).first(),
  ).toBeVisible();
  const deletedAt = Date.now();

  // F8 takes the focus to the messages, where Undo is the next stop.
  await page.keyboard.press('F8');
  await tabTo(page, page.getByRole('button', { name: 'Undo' }));
  await page.keyboard.press('Enter');
  expect(Date.now() - deletedAt).toBeLessThan(10_000);
  await expect(page.getByText(`Sample folder '${renamed}' restored`).first()).toBeVisible();
  await expect(page.getByRole('button', { name: `Actions for ${renamed}` })).toBeVisible();
});
