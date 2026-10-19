import AxeBuilder from '@axe-core/playwright';
import { expect, type Page } from '@playwright/test';

/** The password every demo person shares (shown on the sign-in page, decision D-004). */
export const DEMO_PASSWORD = 'northstar';

/**
 * Signs `email` in through the API (the browser keeps the session cookie) and resets that person's
 * preferences, so every test starts from the same screen (research R19).
 */
export async function signInAs(page: Page, email: string): Promise<void> {
  const signIn = await page.request.post('/api/auth/sign-in', {
    data: { email, password: DEMO_PASSWORD },
  });
  expect(signIn.ok()).toBe(true);
  const reset = await page.request.patch('/api/me/preferences', {
    data: { theme: 'system', sidebarCollapsed: false },
  });
  expect(reset.ok()).toBe(true);
}

/** A name no other test uses, so tests never depend on each other's records. */
export function uniqueName(prefix: string): string {
  return `${prefix} ${Date.now().toString(36)}`;
}

/** Checks the page against WCAG 2.1 A and AA with axe and expects no violations (SC-006). */
export async function expectNoAxeViolations(page: Page): Promise<void> {
  const results = await new AxeBuilder({ page })
    .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
    .analyze();
  const summary = results.violations.map(
    (v) => `${v.id}: ${v.nodes.map((node) => node.target.join(' ')).join(', ')}`,
  );
  expect(summary).toEqual([]);
}
