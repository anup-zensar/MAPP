import { DEMO_PASSWORD, expectNoAxeViolations, signInAs } from './fixtures';
import { expect, test } from './support/no-csp-violations';

// Feature 038 User Story 6: sign-in brute force is refused and every answer carries the security
// headers (FR-014, FR-015, SC-005). Every test also fails on a content security policy violation.

const ALEX = 'alex.morgan@shell.com';
// Locked for 15 minutes by the last test; only specs that run before this file sign Lea in.
const LOCKED = 'lea.santos@shell.com';
const TOO_MANY = 'Too many sign-in attempts. Try again in 15 minutes.';

test('038 FR-015: a web page forbids framing, foreign referrers and sniffing, and admits only its own code', async ({
  page,
}) => {
  const answer = await page.goto('/sign-in');
  const headers = answer?.headers() ?? {};
  expect(headers['x-frame-options']).toBe('DENY');
  expect(headers['referrer-policy']).toBe('same-origin');
  expect(headers['x-content-type-options']).toBe('nosniff');
  expect(headers['content-security-policy']).toMatch(/script-src 'self' 'sha256-[^']+';/);
  expect(headers['content-security-policy']).toContain("frame-ancestors 'none'");
  // The theme script ran: it is admitted by its hash.
  await expect(page.locator('html')).toHaveAttribute('data-theme', /^(light|dark)$/);
  await expectNoAxeViolations(page);
});

test('038 FR-015: an API answer carries the protections too', async ({ page }) => {
  const answer = await page.request.get('/api/v1/auth/session');
  const headers = answer.headers();
  expect(headers['x-frame-options']).toBe('DENY');
  expect(headers['x-content-type-options']).toBe('nosniff');
  expect(headers['content-security-policy']).toContain("frame-ancestors 'none'");
  expect(headers['referrer-policy']).toBeTruthy();
});

test('038 FR-015: another site cannot show the app in a frame', async ({ page, baseURL }) => {
  await page.setContent(`<iframe title="framed" src="${baseURL ?? ''}/sign-in"></iframe>`);
  const framed = page.frameLocator('iframe[title="framed"]');
  await expect(framed.getByRole('heading', { name: 'Sign in to SP&A One' })).toHaveCount(0);
});

test('038 FR-015: a side panel opens without breaking the content security policy', async ({
  page,
}) => {
  await signInAs(page, ALEX);
  await page.goto('/settings/access');
  await page.getByRole('button', { name: 'Edit roles of Sara Novak' }).click();
  await expect(page.getByRole('dialog', { name: 'Roles of Sara Novak' })).toBeVisible();
  await expectNoAxeViolations(page);
  await page.keyboard.press('Escape');
  await expect(page.getByRole('dialog')).toHaveCount(0);
});

test('038 FR-014: after 5 wrong passwords sign-in is refused, also with the right password', async ({
  page,
}) => {
  await page.goto('/sign-in');
  const email = page.getByLabel('E-mail');
  const password = page.getByLabel('Password', { exact: true });
  const signIn = page.getByRole('button', { name: 'Sign in', exact: true });

  await email.fill(LOCKED);
  for (let attempt = 1; attempt <= 6; attempt += 1) {
    await password.fill(`wrong-${attempt}`);
    await signIn.click();
    await expect(page.getByRole('alert').filter({ hasText: /password|attempts/ })).toBeVisible();
  }
  await expect(page.getByText(TOO_MANY)).toBeVisible();

  await password.fill(DEMO_PASSWORD);
  await signIn.click();
  await expect(page.getByText(TOO_MANY)).toBeVisible();
  await expect(page).toHaveURL(/\/sign-in/);
  await expectNoAxeViolations(page);
});
