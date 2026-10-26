import AxeBuilder from '@axe-core/playwright';
import { expect, type Locator, type Page } from '@playwright/test';

/** The password every demo person shares (shown on the sign-in page, decision D-004). */
export const DEMO_PASSWORD = 'mapp';

/**
 * Signs `email` in through the API (the browser keeps the session cookie) and resets that person's
 * preferences — theme, sidebar, display currency (003), date format (003), Home's scope (009) and the
 * History sections' Include items inside (011) — so every test starts from the same screen (research R19).
 */
export async function signInAs(page: Page, email: string): Promise<void> {
  const signIn = await page.request.post('/api/v1/auth/sign-in', {
    data: { email, password: DEMO_PASSWORD },
  });
  expect(signIn.ok()).toBe(true);
  const reset = await page.request.patch('/api/v1/me/preferences', {
    data: {
      theme: 'system',
      sidebarCollapsed: false,
      displayCurrency: 'USD',
      dateFormat: 'DD MMM YYYY',
      homeScope: 'myTeams',
      historyIncludeInside: false,
    },
  });
  expect(reset.ok()).toBe(true);
}

/** A name no other test uses, so tests never depend on each other's records. */
export function uniqueName(prefix: string): string {
  return `${prefix} ${Date.now().toString(36)}`;
}

/** Checks the page against WCAG 2.2 A and AA with axe and expects no violations (SC-006). */
export async function expectNoAxeViolations(page: Page): Promise<void> {
  const results = await new AxeBuilder({ page })
    .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa', 'wcag22aa'])
    .analyze();
  const summary = results.violations.map(
    (v) => `${v.id}: ${v.nodes.map((node) => node.target.join(' ')).join(', ')}`,
  );
  expect(summary).toEqual([]);
}

/** Expects a message on screen, such as a toast; a toast's text is also copied for screen readers. */
export async function expectToast(page: Page, text: string | RegExp): Promise<void> {
  await expect(page.getByText(text).first()).toBeVisible();
}

/**
 * The action called exactly `name`, whether the page draws it as a link (LinkButton) or a button.
 * When both exist, the first in the page wins (the page header's or toolbar's).
 */
export function action(scope: Page | Locator, name: string): Locator {
  return scope
    .getByRole('link', { name, exact: true })
    .or(scope.getByRole('button', { name, exact: true }))
    .first();
}

/**
 * Chooses the option of a native select whose text starts with `text`, so "Event" matches
 * "Event · default $25,000.00" and "EUR" matches "EUR – Euro".
 */
export async function chooseOption(select: Locator, text: string): Promise<void> {
  const option = select
    .locator('option')
    .filter({ hasText: new RegExp(`^\\s*${escapeRegExp(text)}`) });
  const label = (await option.first().textContent())?.trim() ?? text;
  await select.selectOption({ label });
}

/** `text` as a regular expression that matches it literally. */
export function escapeRegExp(text: string): string {
  return text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

/**
 * Calls the API as the signed-in person and answers its JSON; a refusal fails the test with the
 * API's own message. For preparing records a test needs, never for what the test proves.
 */
export async function callApi<T>(
  page: Page,
  method: 'GET' | 'POST' | 'PATCH' | 'DELETE',
  url: string,
  data?: unknown,
): Promise<T> {
  const response = await page.request.fetch(url, { method, data });
  if (!response.ok()) {
    throw new Error(`${method} ${url} answered ${response.status()}: ${await response.text()}`);
  }
  const body: T = await response.json();
  return body;
}

/** The item at `index`; fails the test when there is none. */
export function itemAt<T>(items: readonly T[], index: number): T {
  const item = items.at(index);
  if (item === undefined) {
    throw new Error(`Expected an item at ${index} of ${items.length}.`);
  }
  return item;
}
