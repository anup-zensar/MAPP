import { expect, test } from '@playwright/test';
import { chooseOption, expectNoAxeViolations, signInAs } from './fixtures';

// The Home dashboard (009) for a planner and an approver, against the real API with the demo data.
// The demo data they read (backend/seed): FY2026 Lubricants Europe (P-1001, team Lubricants Europe,
// owner Priya Raman) with Helix launch and its activities. Tests create their own records.

const PRIYA = 'priya.raman@shell.com';
const DANIEL = 'daniel.okoro@shell.com';
const CARLOS = 'carlos.mendes@shell.com';

/** The year of the demo plans FY2026 Lubricants Europe and FY2026 Mobility China. */
const SEED_YEAR = 2026;

test('Home: the summary cards and sections for a planner and an approver, with working links (009)', async ({
  page,
}) => {
  test.slow();
  // A planner: Priya Raman, whose team is Lubricants Europe.
  await signInAs(page, PRIYA);
  await page.goto('/');
  await expect(page.getByRole('heading', { name: 'Welcome back, Priya.', level: 1 })).toBeVisible();
  await expect(
    page.getByText(
      'Your own activities, what needs your attention and your approvals always cover all plans.',
    ),
  ).toBeVisible();
  const scope = page.getByRole('group', { name: 'Scope' });
  await expect(scope.getByRole('button', { name: 'My teams' })).toHaveAttribute(
    'aria-pressed',
    'true',
  );

  // The four summary cards are links named "<label>: <value> — <where to>" (FR-005 – FR-007).
  const summary = page.getByRole('region', { name: 'Summary' });
  const teamPlans = summary.getByRole('link', {
    name: /^My teams' plans: \d+ — open the Plans list$/,
  });
  await expect(teamPlans).toBeVisible();
  await expect(summary.getByText(/^\d+ published · \d+ draft$/)).toBeVisible();
  await expect(
    summary.getByRole('link', { name: /^Activities I own: \d+ — open your activities$/ }),
  ).toBeVisible();
  await expect(summary.getByText(/^\d+ declined · \d+ submitted$/)).toBeVisible();
  await expect(
    summary.getByRole('link', { name: /^Waiting for my approval: \d+ — open Approvals$/ }),
  ).toBeVisible();
  await expect(
    summary.getByRole('link', {
      name: /^Planned cost vs plan budget: \$[\d.]+K \(\$[\d,]+\.\d{2}\) — open the Plans list$/,
    }),
  ).toBeVisible();
  await expect(summary.getByText(/^of \$[\d.]+[KM] plan budget$/)).toBeVisible();

  // Plans at a glance: the team's plan with its status and team.
  const glance = page.getByRole('region', { name: 'Plans at a glance' });
  await expect(glance.getByText('Live roll-up across your teams')).toBeVisible();
  const glanceRow = glance.getByRole('row').filter({ hasText: 'FY2026 Lubricants Europe' });
  await expect(glanceRow).toContainText('Published');
  await expect(glanceRow).toContainText('Lubricants Europe');
  await expect(glanceRow).toContainText(/\d+ of \d+ approved/);

  // Needs your attention: Daniel Okoro declined Fleet roadshow (FR-011, FR-012).
  const attention = page.getByRole('region', { name: 'Needs your attention' });
  await expect(attention.getByText('Things you own that have a problem')).toBeVisible();
  await expect(attention.getByText(/^Declined by Daniel Okoro on /).first()).toBeVisible();

  // My activities: groups by status; Complete starts collapsed (FR-014).
  const mine = page.getByRole('region', { name: 'My activities' });
  await expect(mine.getByText('Activities you own, by status')).toBeVisible();
  await expect(mine.getByRole('button', { name: /^Declined \d+$/ })).toHaveAttribute(
    'aria-expanded',
    'true',
  );
  const complete = mine.getByRole('button', { name: /^Complete \d+$/ });
  await expect(complete).toHaveAttribute('aria-expanded', 'false');
  await expect(mine.getByRole('link', { name: 'Billboard winter wave' })).toHaveCount(0);
  await complete.click();
  await expect(complete).toHaveAttribute('aria-expanded', 'true');
  await expect(mine.getByRole('link', { name: 'Billboard winter wave' })).toBeVisible();
  await expect(mine).toContainText(`05 Jan – 28 Feb ${SEED_YEAR}`);

  // The scope switch: Everything counts every plan Priya may see — for her, the plans of her team
  // (FR-003, 014 FR-007); a Global planner's Everything is checked at the end.
  await scope.getByRole('button', { name: 'Everything' }).click();
  await expect(
    summary.getByRole('link', { name: /^All plans: \d+ — open the Plans list$/ }),
  ).toBeVisible();
  await expect(glance.getByText('Live roll-up across all plans')).toBeVisible();
  await scope.getByRole('button', { name: 'My teams' }).click();
  await expect(teamPlans).toBeVisible();

  // The currency picker redraws the amounts in the chosen currency (FR-001, FR-002).
  const currency = page.getByRole('combobox', { name: 'Currency' });
  await chooseOption(currency, 'EUR — Euro');
  await expect(
    summary.getByRole('link', { name: /^Planned cost vs plan budget: [\d.]+K EUR / }),
  ).toBeVisible();
  await expect(glanceRow).toContainText(/[\d,]+\.\d{2} EUR/);
  await chooseOption(currency, 'USD — US dollar');
  await expect(
    summary.getByRole('link', { name: /^Planned cost vs plan budget: \$/ }),
  ).toBeVisible();
  await expectNoAxeViolations(page);

  // Links: the decline opens the activity's Approval section; the cards open their lists.
  await attention
    .getByRole('link', { name: "Open approval for activity 'Fleet roadshow'" })
    .click();
  await expect(page).toHaveURL(/\/plans\/P-1001\/activities\/A-1005#approval$/);
  await expect(page.getByRole('heading', { name: 'Fleet roadshow', level: 1 })).toBeVisible();
  await expect(
    page
      .locator('#approval')
      .getByText(/^Declined by Daniel Okoro on /)
      .first(),
  ).toBeVisible();

  await page.goto('/');
  await glance.getByRole('link', { name: 'FY2026 Lubricants Europe', exact: true }).click();
  await expect(page).toHaveURL(/\/plans\/P-1001$/);
  await expect(
    page.getByRole('heading', { name: 'FY2026 Lubricants Europe', level: 1 }),
  ).toBeVisible();

  await page.goto('/');
  await teamPlans.click();
  await expect(page).toHaveURL(/\/plans\?scope=myTeams$/);
  await expect(page.getByRole('button', { name: 'Remove filter: My teams’ plans' })).toBeVisible();

  // "Activities I own" opens All work with the owner filter (D-174).
  await page.goto('/');
  await summary.getByRole('link', { name: /^Activities I own: / }).click();
  await expect(page).toHaveURL(/\/all-work/);
  await expect(
    page.getByRole('button', { name: 'Remove filter: Owner: Priya Raman' }),
  ).toBeVisible();

  // An approver: Daniel Okoro, with Trade magazine ads waiting for him at level 1 of 1.
  await signInAs(page, DANIEL);
  await page.goto('/');
  await expect(
    page.getByRole('heading', { name: 'Welcome back, Daniel.', level: 1 }),
  ).toBeVisible();
  await expect(
    summary.getByRole('link', { name: /^Waiting for my approval: [1-9]\d* — open Approvals$/ }),
  ).toBeVisible();
  await expect(summary.getByText(/^oldest waiting /)).toBeVisible();
  const waiting = page.getByRole('region', { name: 'Waiting for my approval' });
  const waitingRow = waiting.getByRole('listitem').filter({ hasText: 'Trade magazine ads' });
  await expect(waitingRow).toContainText('Level 1 of 1');
  await expect(waitingRow).toContainText(/waiting /);
  await expect(page.getByRole('region', { name: 'My activities' })).toContainText(
    "You don't own any activities yet.",
  );
  await expectNoAxeViolations(page);

  // Review opens Approvals with the activity's preview (SC-003).
  await waitingRow.getByRole('link', { name: 'Review Trade magazine ads' }).click();
  await expect(page).toHaveURL(/\/approvals\?tab=waiting&activity=A-1004$/);
  const preview = page.getByRole('dialog', { name: 'Trade magazine ads' });
  await expect(preview.getByText('A-1004 · Waiting at level 1 of 1')).toBeVisible();
  await preview.getByRole('button', { name: 'Close' }).click();

  await page.goto('/');
  await summary.getByRole('link', { name: /^Waiting for my approval: / }).click();
  await expect(page).toHaveURL(/\/approvals\?tab=waiting$/);
  await expect(page.getByRole('heading', { name: 'Approvals', level: 1 })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Trade magazine ads', exact: true })).toBeVisible();

  // A Global planner: Carlos Mendes sees every market, so Everything counts more plans than his teams'
  // (FR-003, 014 FR-007). Each card's value is the link's text.
  await signInAs(page, CARLOS);
  await page.goto('/');
  await expect(
    page.getByRole('heading', { name: 'Welcome back, Carlos.', level: 1 }),
  ).toBeVisible();
  // Each card's value is in its link's name, "All plans: 10 — open the Plans list".
  const cardValue = async (link: typeof teamPlans) =>
    Number(/: (\d+) —/.exec((await link.getAttribute('aria-label')) ?? '')?.[1]);
  const teamPlanCount = await cardValue(teamPlans);
  expect(teamPlanCount).toBeGreaterThan(0);
  await scope.getByRole('button', { name: 'Everything' }).click();
  const allPlans = summary.getByRole('link', { name: /^All plans: \d+ — open the Plans list$/ });
  await expect(allPlans).toBeVisible();
  // Polled: the label changes at once, the count when the new scope's figures arrive.
  await expect.poll(async () => cardValue(allPlans)).toBeGreaterThan(teamPlanCount);
  await scope.getByRole('button', { name: 'My teams' }).click();
  await expect(teamPlans).toBeVisible();
});
