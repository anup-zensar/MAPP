import { expect, test } from '@playwright/test';
import { signInAs } from './fixtures';

// 026 US3: each feature's code arrives with its first page, never before.

const FEATURES = [
  'activities',
  'activity-log',
  'all-work',
  'approval-workflows',
  'approvals',
  'dev',
  'financials',
  'global-budgets',
  'home',
  'line-items',
  'people',
  'plans',
  'recycle-bin',
  'settings',
  'sign-in',
];

/** The features whose code the page downloaded, from the names of the script files. */
function featuresIn(scripts: string[]): string[] {
  return FEATURES.filter((feature) =>
    scripts.some((url) => new RegExp(`/assets/${feature}-[\\w-]+\\.js$`).test(url)),
  );
}

test('026 FR-013: the sign-in page loads no other feature’s code (US3 scenario 2)', async ({
  page,
}) => {
  const scripts: string[] = [];
  page.on('request', (request) => {
    if (request.resourceType() === 'script') {
      scripts.push(request.url());
    }
  });

  await page.goto('/sign-in');
  await expect(page.getByRole('heading', { name: 'Sign in to MAPP' })).toBeVisible();

  expect(featuresIn(scripts)).toEqual(['sign-in']);
});

test('026 FR-012: a page whose code cannot be downloaded shows the error page with Try again (US3 scenario 3)', async ({
  page,
}) => {
  await signInAs(page, 'priya.raman@shell.com');
  await page.route(/\/assets\/plans-[\w-]+\.js$/, (route) => route.abort());

  await page.goto('/plans');

  await expect(page.getByRole('alert')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Try again' })).toBeVisible();
});
