import AxeBuilder from '@axe-core/playwright';
import { expect, type Locator, type Page } from '@playwright/test';

export const DEMO_PASSWORD = 'mapp';

/** Signs `email` in and resets their preferences, so every test starts from the same screen. */
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

export function uniqueName(prefix: string): string {
  return `${prefix} ${Date.now().toString(36)}`;
}

export async function expectNoAxeViolations(page: Page): Promise<void> {
  // The product name is a logotype, exempt from text contrast (WCAG 1.4.3, constitution X); axe cannot tell.
  const results = await new AxeBuilder({ page })
    .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa', 'wcag22aa'])
    .exclude('[role="img"][aria-label="SP&A One"] > span')
    .analyze();
  const summary = results.violations.map(
    (v) => `${v.id}: ${v.nodes.map((node) => node.target.join(' ')).join(', ')}`,
  );
  expect(summary).toEqual([]);
}

export async function expectToast(page: Page, text: string | RegExp): Promise<void> {
  await expect(page.getByText(text).first()).toBeVisible();
}

/** The action called exactly `name`, link or button; when both exist, the first one wins. */
export function action(scope: Page | Locator, name: string): Locator {
  return scope
    .getByRole('link', { name, exact: true })
    .or(scope.getByRole('button', { name, exact: true }))
    .first();
}

export async function chooseOption(select: Locator, text: string): Promise<void> {
  const option = select
    .locator('option')
    .filter({ hasText: new RegExp(`^\\s*${escapeRegExp(text)}`) });
  const label = (await option.first().textContent())?.trim() ?? text;
  await select.selectOption({ label });
}

export function escapeRegExp(text: string): string {
  return text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

/** Calls the API as the signed-in person: to prepare a test's records, never for what it proves. */
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

export function itemAt<T>(items: readonly T[], index: number): T {
  const item = items.at(index);
  if (item === undefined) {
    throw new Error(`Expected an item at ${index} of ${items.length}.`);
  }
  return item;
}
