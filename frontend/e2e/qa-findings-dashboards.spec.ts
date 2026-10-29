import type { Page } from '@playwright/test';
import { expect, test } from './support/no-csp-violations';
import { expectNoAxeViolations, signInAs } from './fixtures';

// 038 US4 (H-1): an address with a currency or market the person cannot use shows the dashboard with the
// default, and the address and the filter bar say which was used (FR-012).

const CARLOS = 'carlos.mendes@shell.com';

async function expectFallbackCurrency(page: Page, asked: string): Promise<void> {
  await page.goto(`/dashboards/executive?currency=${asked}`);
  await expect(page.getByRole('heading', { level: 1, name: 'Executive dashboard' })).toBeVisible();
  await expect(page.getByLabel('Summary')).toBeVisible();
  await expect(page.getByRole('button', { name: /Try again/ })).toHaveCount(0);
  await expect(page).toHaveURL(/currency=[A-Z]{3}/);
  const used = new URL(page.url()).searchParams.get('currency') ?? '';
  expect(used).not.toBe(asked);
  await expect(page.getByLabel('Currency', { exact: true })).toHaveValue(used);
}

test('038 FR-012 (H-1): an unknown currency falls back to the default, in the filter bar and the address', async ({
  page,
}) => {
  await signInAs(page, CARLOS);

  await expectFallbackCurrency(page, 'XXX');
  await expectNoAxeViolations(page);
  await expectFallbackCurrency(page, 'JPY');
});

test('038 FR-012 (H-1): an unknown market falls back to the default market', async ({ page }) => {
  await signInAs(page, CARLOS);

  await page.goto('/dashboards/regional?team=T-9999');
  await expect(page.getByRole('heading', { level: 1, name: 'Regional dashboard' })).toBeVisible();
  await expect(page.getByRole('button', { name: /Try again/ })).toHaveCount(0); // the default market may have no plans
  await expect(page).toHaveURL(/team=T-\d+/);
  const used = new URL(page.url()).searchParams.get('team') ?? '';
  expect(used).not.toBe('T-9999');
  await expect(page.getByLabel('Market', { exact: true })).toHaveValue(used);
  await expectNoAxeViolations(page);
});
