import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { mockApi } from '@/test/mockApi';
import type { MockHandler } from '@/test/mockApi';
import { renderWithProviders } from '@/test/render';
import { testUser } from '@/test/users';
import type { HomeScope, HomeSummary } from './homeApi';
import { SummaryCards } from './SummaryCards';
import { summaryOf, usd } from './testData';

function renderCards(
  scope: HomeScope = 'myTeams',
  answer: MockHandler = { body: summaryOf({ scope }) },
) {
  const api = mockApi({ 'GET /api/home/summary': answer });
  const rendered = renderWithProviders(<SummaryCards scope={scope} />, {
    user: testUser(),
    path: '/',
  });
  return { ...rendered, api };
}

function cardOf(link: HTMLElement): HTMLElement {
  // The link is the value; its card holds the label, the value and the lines under it.
  return link.closest('div') as HTMLElement;
}

const plannedCost = (overrides: Partial<HomeSummary['plannedCost']>) =>
  summaryOf({ plannedCost: { ...summaryOf().plannedCost, ...overrides } });

describe('SummaryCards', () => {
  it('shows the four cards with their figures and opens each list with the same filter (FR-005, FR-007, US1 scenario 2)', async () => {
    const { api } = renderCards();

    const plans = await screen.findByRole('link', {
      name: "My teams' plans: 1 — open the Plans list",
    });
    expect(plans).toHaveAttribute('href', '/plans?scope=myTeams');
    expect(cardOf(plans)).toHaveTextContent('1 published · 0 draft');

    const owned = screen.getByRole('link', { name: 'Activities I own: 6 — open your activities' });
    // All work is still "Coming soon": the card leads to My activities on Home (D-174).
    expect(owned).toHaveAttribute('href', '#my-activities');
    expect(cardOf(owned)).toHaveTextContent('1 declined · 2 submitted');

    const waiting = screen.getByRole('link', {
      name: 'Waiting for my approval: 0 — open Approvals',
    });
    expect(waiting).toHaveAttribute('href', '/approvals?tab=waiting');
    expect(cardOf(waiting)).toHaveTextContent('Nothing waiting');
    expect(cardOf(waiting).className).not.toMatch(/review/);

    const cost = screen.getByRole('link', {
      name: 'Planned cost vs plan budget: $1M ($1,003,000.00) — open the Plans list',
    });
    expect(cost).toHaveAttribute('href', '/plans?scope=myTeams');
    expect(cardOf(cost)).toHaveTextContent('of $1.2M plan budget');
    expect(cardOf(cost)).toHaveTextContent('84 % used');
    expect(within(cardOf(cost)).getByRole('progressbar', { name: 'Budget used' })).toHaveAttribute(
      'aria-valuenow',
      '84',
    );

    expect(api.requests.map((request) => request.search)).toEqual(['?scope=myTeams']);
  });

  it('explains every card with an info tip (FR-005)', async () => {
    renderCards();

    await screen.findByRole('link', { name: /My teams' plans/ });
    for (const label of [
      "My teams' plans",
      'Activities I own',
      'Waiting for my approval',
      'Planned cost vs plan budget',
    ]) {
      expect(screen.getByRole('button', { name: `About ${label}` })).toBeInTheDocument();
    }
  });

  it('covers every plan under Everything and opens the whole Plans list (FR-003, FR-007)', async () => {
    const { api } = renderCards('everything', {
      body: summaryOf({ scope: 'everything', plans: { count: 3, published: 1, draft: 2 } }),
    });

    expect(
      await screen.findByRole('link', { name: 'All plans: 3 — open the Plans list' }),
    ).toHaveAttribute('href', '/plans?scope=all');
    expect(screen.getByText('1 published · 2 draft')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /^Planned cost vs plan budget: / })).toHaveAttribute(
      'href',
      '/plans?scope=all',
    );
    expect(api.requests.map((request) => request.search)).toEqual(['?scope=everything']);
  });

  it('shows the waiting count in amber with how long the oldest has waited (FR-005)', async () => {
    const since = new Date(Date.now() - (3 * 24 + 2) * 3_600_000).toISOString();
    renderCards('myTeams', {
      body: summaryOf({ waitingForMe: { count: 2, oldestWaitingSince: since } }),
    });

    const waiting = await screen.findByRole('link', {
      name: 'Waiting for my approval: 2 — open Approvals',
    });
    expect(waiting.parentElement?.className).toMatch(/review/);
    expect(cardOf(waiting)).toHaveTextContent('oldest waiting 3 days');
  });

  it('shows a budget used above 100 % in red (FR-005, FR-006)', async () => {
    renderCards('myTeams', {
      body: plannedCost({
        plannedCost: usd('1500000.00'),
        budgetUsedPercent: 125,
        overBudget: true,
      }),
    });

    const used = await screen.findByText('125 % used');
    expect(used.className).toMatch(/problem/);
    expect(screen.getByRole('progressbar', { name: 'Budget used' })).toHaveAttribute(
      'aria-valuetext',
      '125 %',
    );
  });

  it('says how many plans in scope have no budget (FR-006)', async () => {
    renderCards('everything', {
      body: plannedCost({ planCount: 3, plansWithoutBudget: 2 }),
    });

    expect(await screen.findByText('Includes 2 plans with no budget')).toBeInTheDocument();
  });

  it('shows "—" and "No plan budgets set" when no plan in scope has a budget (FR-006)', async () => {
    renderCards('myTeams', {
      body: plannedCost({
        planBudget: null,
        budgetUsedPercent: null,
        plansWithoutBudget: 1,
      }),
    });

    const cost = await screen.findByRole('link', { name: /^Planned cost vs plan budget: / });
    expect(cardOf(cost)).toHaveTextContent('No plan budgets set');
    expect(cardOf(cost)).toHaveTextContent('—');
    expect(cardOf(cost)).not.toHaveTextContent('Includes');
  });

  it('shows "—" and "No plans in scope" when the scope holds no plan (FR-021)', async () => {
    renderCards('myTeams', {
      body: plannedCost({
        planCount: 0,
        plannedCost: usd('0.00'),
        planBudget: null,
        budgetUsedPercent: null,
      }),
    });

    expect(
      await screen.findByRole('link', {
        name: 'Planned cost vs plan budget: — — open the Plans list',
      }),
    ).toBeInTheDocument();
    expect(screen.getByText('No plans in scope')).toBeInTheDocument();
  });

  it('shows every amount in the display currency, labelled (US1 scenario 4, FR-018)', async () => {
    renderCards('myTeams', {
      body: summaryOf({
        displayCurrency: 'EUR',
        plannedCost: {
          ...summaryOf().plannedCost,
          plannedCost: { amount: '922760.00', currency: 'EUR' },
          planBudget: { amount: '1104000.00', currency: 'EUR' },
        },
      }),
    });

    expect(
      await screen.findByRole('link', {
        name: 'Planned cost vs plan budget: 922.76K EUR (922,760.00 EUR) — open the Plans list',
      }),
    ).toBeInTheDocument();
    expect(screen.getByText('of 1.1M EUR plan budget')).toBeInTheDocument();
  });

  it('shows a placeholder in each card while loading (FR-022)', () => {
    renderCards('myTeams', () => new Promise<never>(() => {}));

    const summary = screen.getByRole('region', { name: 'Summary' });
    expect(within(summary).getAllByRole('status')).toHaveLength(4);
  });

  it('says the cards could not load and loads them again on Try again (FR-022)', async () => {
    let calls = 0;
    const { user } = renderCards('myTeams', () =>
      ++calls === 1 ? { status: 500 } : { body: summaryOf() },
    );

    expect(await screen.findByRole('alert')).toHaveTextContent("This section couldn't load");
    await user.click(screen.getByRole('button', { name: 'Try again' }));

    await waitFor(() =>
      expect(
        screen.getByRole('link', { name: "My teams' plans: 1 — open the Plans list" }),
      ).toBeInTheDocument(),
    );
  });
});
