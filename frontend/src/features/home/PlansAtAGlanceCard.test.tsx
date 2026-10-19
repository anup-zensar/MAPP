import { screen, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { mockApi } from '@/test/mockApi';
import type { MockHandler } from '@/test/mockApi';
import { renderWithProviders } from '@/test/render';
import { testUser } from '@/test/users';
import type { HomePlanRow, HomeScope } from './homeApi';
import { PlansAtAGlanceCard } from './PlansAtAGlanceCard';
import { planRowOf, usd } from './testData';

const LUBRICANTS = planRowOf();
const CHINA = planRowOf({
  id: 'P-1002',
  name: 'FY2026 Mobility China',
  status: 'draft',
  teams: [{ id: 'T-1002', name: 'Mobility China', colour: 'teal', active: true, mine: true }],
  campaignCount: 2,
  activityCount: 4,
  approvedActivityCount: 4,
  budget: usd('800000.00'),
  plannedCost: usd('920000.00'),
  budgetUsedPercent: 115,
  overBudget: true,
});
const GLOBAL_BRAND = planRowOf({
  id: 'P-1003',
  name: 'FY2027 Global Brand',
  year: 2027,
  strategicAop: true,
  status: 'draft',
  teams: [
    { id: 'T-1003', name: 'Global Marketing Ops', colour: 'violet', active: true, mine: false },
  ],
  campaignCount: 1,
  activityCount: 0,
  approvedActivityCount: 0,
  budget: null,
  plannedCost: null,
  budgetUsedPercent: null,
  overBudget: false,
});

const page = (items: HomePlanRow[], total = items.length) => ({ body: { items, total } });

function renderCard(
  scope: HomeScope = 'myTeams',
  plans: MockHandler = page([LUBRICANTS, CHINA]),
  onShowEverything = vi.fn(),
) {
  const api = mockApi({ 'GET /api/home/plans': plans });
  const rendered = renderWithProviders(
    <PlansAtAGlanceCard scope={scope} onShowEverything={onShowEverything} />,
    { user: testUser(), path: '/' },
  );
  return { ...rendered, api, onShowEverything };
}

function rowOf(name: string): HTMLElement {
  return screen.getByRole('link', { name }).closest('tr') as HTMLElement;
}

describe('PlansAtAGlanceCard', () => {
  it('lists each plan with status, teams, counts, approval, budget used and planned cost (FR-008, US2 scenario 1)', async () => {
    const { api } = renderCard();

    const name = await screen.findByRole('link', { name: 'FY2026 Lubricants Europe' });
    expect(name).toHaveAttribute('href', '/plans/P-1001');
    const cells = within(rowOf('FY2026 Lubricants Europe')).getAllByRole('cell');
    expect(within(rowOf('FY2026 Lubricants Europe')).getByRole('rowheader')).toHaveTextContent(
      'FY2026 Lubricants EuropeP-1001 · 2026',
    );
    expect(cells.map((cell) => cell.textContent)).toEqual([
      'Published',
      'Lubricants Europe',
      '3',
      '7',
      '3 of 7 approved',
      '84 %',
      '$1,003,000.00',
    ]);
    expect(
      within(cells[4]).getByRole('progressbar', { name: 'Approval progress' }),
    ).toHaveAttribute('aria-valuenow', '43');
    expect(within(cells[5]).getByRole('progressbar', { name: 'Budget used' })).toHaveAttribute(
      'aria-valuenow',
      '84',
    );
    expect(api.requests.find((r) => r.path === '/api/home/plans')?.search).toBe('?scope=myTeams');
  });

  it('shows each team as a pill in its own colour, with its name as text (FR-008, SC-007, D-217)', async () => {
    const { api } = renderCard();

    await screen.findByRole('link', { name: 'FY2026 Mobility China' });
    const pill = within(rowOf('FY2026 Mobility China')).getByText('Mobility China');
    expect(pill.querySelector('[aria-hidden="true"]')?.getAttribute('style')).toContain(
      'var(--team-teal)',
    );
    // The row carries the colour itself: no list of teams is asked for.
    expect(api.requests.some((request) => request.path === '/api/teams')).toBe(false);
  });

  it('shows a budget used above 100 % in red (US2 scenario 2)', async () => {
    renderCard();

    await screen.findByRole('link', { name: 'FY2026 Mobility China' });
    const row = within(rowOf('FY2026 Mobility China'));
    expect(row.getByText('115 %').className).toMatch(/problem/);
    expect(row.getByRole('progressbar', { name: 'Budget used' })).toHaveAttribute(
      'aria-valuetext',
      '115 %',
    );
    expect(row.getByText('4 of 4 approved')).toBeInTheDocument();
  });

  it('says "No budget" and "No activities" for a plan without them, and marks AOP plans (US2 scenario 3)', async () => {
    renderCard('everything', page([GLOBAL_BRAND, LUBRICANTS]));

    await screen.findByRole('link', { name: 'FY2027 Global Brand' });
    const row = within(rowOf('FY2027 Global Brand'));
    expect(row.getByRole('rowheader')).toHaveTextContent('P-1003 · 2027 AOP');
    expect(row.getByText('No activities')).toBeInTheDocument();
    expect(row.getByText('No budget').parentElement).toHaveTextContent('— No budget');
    expect(row.queryByRole('progressbar')).not.toBeInTheDocument();
    // Nothing planned shows "—", as on the plan page (D-217).
    expect(row.getAllByRole('cell').at(-1)).toHaveTextContent('—');
    expect(row.getByText('Global Marketing Ops')).toBeInTheDocument();
  });

  it('opens the plan page when its row is chosen (FR-010)', async () => {
    const { user, router } = renderCard();

    await screen.findByRole('link', { name: 'FY2026 Mobility China' });
    await user.click(within(rowOf('FY2026 Mobility China')).getByText('Draft'));

    expect(router.state.location.pathname).toBe('/plans/P-1002');
  });

  it('offers "View all <n>" with the Plans list of the same scope when more than 10 plans are in scope (FR-010)', async () => {
    renderCard('everything', page([LUBRICANTS, CHINA], 12));

    expect(await screen.findByRole('link', { name: 'View all 12 plans' })).toHaveAttribute(
      'href',
      '/plans?scope=all',
    );
  });

  it('has no "View all" when every plan in scope is shown', async () => {
    renderCard();

    await screen.findByRole('link', { name: 'FY2026 Lubricants Europe' });
    expect(screen.queryByRole('link', { name: /View all/ })).not.toBeInTheDocument();
  });

  it('says which plans it covers and what it shows (FR-008)', async () => {
    const { user } = renderCard('everything');

    const card = screen.getByRole('region', { name: 'Plans at a glance' });
    expect(card).toHaveTextContent('Live roll-up across all plans');
    await user.click(within(card).getByRole('button', { name: 'About Plans at a glance' }));
    expect(screen.getByRole('dialog', { name: 'About Plans at a glance' })).toHaveTextContent(
      'Every plan your teams work on, the teams on it, and live status rolled up from its activities.',
    );
  });

  it('offers Switch to Everything when none of the teams has a plan (FR-021)', async () => {
    const { user, onShowEverything } = renderCard('myTeams', page([]));

    expect(await screen.findByText('None of your teams has a plan yet.')).toBeInTheDocument();
    expect(screen.getByText('Live roll-up across your teams')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Switch to Everything' }));

    expect(onShowEverything).toHaveBeenCalledOnce();
  });

  it('sends people to the Plans page when there is no plan at all (FR-021)', async () => {
    renderCard('everything', page([]));

    expect(
      await screen.findByText('No plans yet — create the first one on the Plans page.'),
    ).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Go to Plans' })).toHaveAttribute('href', '/plans');
  });

  it('shows a placeholder while loading (FR-022)', () => {
    renderCard('myTeams', () => new Promise<never>(() => {}));

    const card = screen.getByRole('region', { name: 'Plans at a glance' });
    expect(within(card).getByRole('status')).toHaveTextContent('Loading…');
  });

  it('says the card could not load and loads it again on Try again (FR-022)', async () => {
    let calls = 0;
    const { user } = renderCard('myTeams', () =>
      ++calls === 1 ? { status: 500 } : page([LUBRICANTS]),
    );

    expect(await screen.findByRole('alert')).toHaveTextContent("This section couldn't load");
    await user.click(screen.getByRole('button', { name: 'Try again' }));

    expect(
      await screen.findByRole('link', { name: 'FY2026 Lubricants Europe' }),
    ).toBeInTheDocument();
  });
});
