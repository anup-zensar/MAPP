import { screen, waitFor, within } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { mockApi } from '@/test/mockApi';
import type { MockHandler } from '@/test/mockApi';
import { renderWithProviders } from '@/test/render';
import { testUser } from '@/test/users';
import { remember, DEFAULT_FILTERS } from './listPreferences';
import { PlansPage } from './PlansPage';
import { seedList } from './testData';

afterEach(() => window.localStorage.clear());

function renderPage(
  route = '/plans',
  list: MockHandler = { body: seedList() },
  { user = testUser(), handlers = {} as Record<string, MockHandler> } = {},
) {
  const api = mockApi({ 'GET /api/plans': list, ...handlers });
  const result = renderWithProviders(<PlansPage />, { user, route, path: '/plans' });
  return { ...result, api };
}

/** The page with `pinned` already pinned and PATCH /api/me/preferences answered (018). */
function renderWithPins(...pinned: string[]) {
  return renderPage(
    '/plans',
    { body: seedList() },
    {
      user: testUser({
        preferences: { ...testUser().preferences, pinnedPlanIds: pinned },
      }),
      handlers: { 'PATCH /api/me/preferences': { body: {} } },
    },
  );
}

/** The query strings sent to GET /api/plans, in order. */
function listQueries(api: ReturnType<typeof mockApi>) {
  return api.requests.filter((request) => request.path === '/api/plans').map((r) => r.search);
}

function cardNames() {
  return screen
    .getAllByRole('article')
    .map((card) => within(card).getByRole('heading').textContent);
}

describe('PlansPage (US2, FR-001 – FR-004)', () => {
  it('shows the plans newest year first with the summary cards (US2#1, FR-001)', async () => {
    renderPage();

    expect(screen.getByRole('heading', { level: 1, name: 'Plans' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'New plan' })).toHaveAttribute('href', '/plans/new');
    await waitFor(() =>
      expect(cardNames()).toEqual([
        'FY2026 Lubricants Europe',
        'FY2026 Mobility China',
        'FY2025 Lubricants Europe',
      ]),
    );
    const summary = screen.getByRole('region', { name: 'Summary' });
    expect(within(summary).getByText('Plans').parentElement).toHaveTextContent('3');
    expect(within(summary).getByText('2 draft · 1 published')).toBeInTheDocument();
    expect(
      within(summary).getByRole('button', { name: '$2.26M ($2,260,000.00)' }),
    ).toHaveTextContent('$2.26M');
    expect(
      within(summary).getByRole('button', { name: /^\S+ \(\$2,400,000\.00\)$/ }),
    ).toBeInTheDocument();
    expect(screen.getByText('3 of 3 plans')).toBeInTheDocument();
  });

  it('filters by status and year with chips and the count (US2#2, FR-003)', async () => {
    const { api, user, router } = renderPage('/plans', (request) => {
      const url = new URL(request.url);
      return url.searchParams.get('status') === 'draft' && url.searchParams.get('year') === '2026'
        ? { body: seedList({ items: [seedList().items[1]] }) }
        : { body: seedList() };
    });

    await screen.findAllByRole('article');
    await user.click(
      within(screen.getByRole('group', { name: 'Status' })).getByRole('button', {
        name: 'Draft (2)',
      }),
    );
    // The year is one of the less-used filters behind Filters (design system §2.4 A).
    await user.click(screen.getByRole('button', { name: 'Filters' }));
    // Until 016's API is merged the list takes one year and knows no plan types.
    await user.selectOptions(screen.getByRole('combobox', { name: 'Year' }), '2026');
    expect(screen.queryByRole('group', { name: 'Plan type' })).not.toBeInTheDocument();
    await user.keyboard('{Escape}');

    await waitFor(() => expect(cardNames()).toEqual(['FY2026 Mobility China']));
    expect(screen.getByText('1 of 3 plans')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Filters (1)' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Remove filter: Draft' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Remove filter: Year: 2026' })).toBeInTheDocument();
    expect(router.state.location.search).toBe('?status=draft&yearFrom=2026&yearTo=2026');
    // One year goes as the year parameter, the alias every API version understands (D-269).
    expect(listQueries(api)).toContain('?scope=all&status=draft&year=2026');

    await user.click(screen.getByRole('button', { name: 'Remove filter: Draft' }));
    await waitFor(() => expect(router.state.location.search).toBe('?yearFrom=2026&yearTo=2026'));
  });

  it('filters by a range of years and by plan type (016 FR-033)', async () => {
    const typed = seedList();
    const { api, user, router } = renderPage('/plans', {
      body: { ...typed, items: typed.items.map((plan) => ({ ...plan, planType: 'operational' })) },
    });

    await screen.findAllByRole('article');
    await user.click(screen.getByRole('button', { name: 'Filters' }));
    const to = screen.getByRole('combobox', { name: 'To' });
    expect(to).toBeDisabled();
    expect(
      within(screen.getByRole('combobox', { name: 'From' }))
        .getAllByRole('option')
        .map((option) => option.textContent),
    ).toEqual(['All years', '2025', '2026']);

    await user.selectOptions(screen.getByRole('combobox', { name: 'From' }), '2025');
    await user.selectOptions(screen.getByRole('combobox', { name: 'To' }), '2026');
    await user.click(screen.getByRole('checkbox', { name: 'Strategic' }));
    await user.click(screen.getByRole('checkbox', { name: 'Cross-market' }));
    await user.keyboard('{Escape}');

    expect(screen.getByRole('button', { name: 'Filters (2)' })).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'Remove filter: Years: 2025 – 2026' }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'Remove filter: Plan type: Strategic, Cross-market' }),
    ).toBeInTheDocument();
    expect(router.state.location.search).toBe(
      '?yearFrom=2025&yearTo=2026&type=strategic&type=crossMarket',
    );
    await waitFor(() =>
      expect(listQueries(api)).toContain(
        '?scope=all&yearFrom=2025&yearTo=2026&type=strategic&type=crossMarket',
      ),
    );

    await user.click(screen.getByRole('button', { name: 'Remove filter: Years: 2025 – 2026' }));
    await waitFor(() =>
      expect(router.state.location.search).toBe('?type=strategic&type=crossMarket'),
    );
  });

  it('reads an old single year in the address as a one-year range (D-269)', async () => {
    const { api } = renderPage('/plans?year=2025');

    expect(
      await screen.findByRole('button', { name: 'Remove filter: Year: 2025' }),
    ).toBeInTheDocument();
    expect(listQueries(api)[0]).toBe('?scope=all&year=2025');
  });

  it('searches name or ID once typing stops (US2#3)', async () => {
    const { api, user } = renderPage();

    await screen.findAllByRole('article');
    await user.type(screen.getByRole('searchbox', { name: 'Search plans' }), 'mobility');

    await waitFor(() => expect(listQueries(api)).toContain('?scope=all&q=mobility'));
    expect(listQueries(api)).not.toContain('?scope=all&q=m');
    expect(
      await screen.findByRole('button', { name: 'Remove filter: Search: mobility' }),
    ).toBeInTheDocument();
  });

  it('shows my teams with the hint (US2#4, FR-004)', async () => {
    const { api, user } = renderPage();

    await screen.findAllByRole('article');
    await user.click(screen.getByRole('button', { name: 'Filters' }));
    await user.click(screen.getByRole('radio', { name: 'My teams’ plans' }));

    expect(
      screen.getByText(
        "Shows your teams' plans. Everyone can still open every plan in this phase.",
      ),
    ).toBeInTheDocument();
    await waitFor(() => expect(listQueries(api)).toContain('?scope=myTeams'));
  });

  it('invites to create the first plan when there are none (US2#5)', async () => {
    renderPage('/plans', {
      body: seedList({
        items: [],
        total: 0,
        years: [],
        summary: { plans: 0, draft: 0, published: 0, plannedCost: null, budget: null },
      }),
    });

    expect(await screen.findByText('No plans yet — create your first plan')).toBeInTheDocument();
    expect(
      screen.getByText('A plan holds one year of marketing for one business.'),
    ).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Create your first plan' })).toHaveAttribute(
      'href',
      '/plans/new',
    );
  });

  it('offers Clear filters when the filters hide every plan (US2#5)', async () => {
    const { user, router } = renderPage('/plans?status=published&year=2026', {
      body: seedList({ items: [] }),
    });

    expect(await screen.findByText('Nothing matches these filters')).toBeInTheDocument();
    expect(screen.getByText('0 of 3 plans')).toBeInTheDocument();
    const clear = screen.getAllByRole('button', { name: 'Clear filters' });
    await user.click(clear[clear.length - 1]);

    await waitFor(() => expect(router.state.location.search).toBe(''));
  });

  it('shows grey shapes while loading and the error with Try again', async () => {
    let calls = 0;
    const { user } = renderPage('/plans', () => {
      calls += 1;
      return calls === 1 ? { status: 500 } : { body: seedList() };
    });

    expect(screen.getByLabelText('Loading plans')).toBeInTheDocument();
    expect(await screen.findByRole('alert')).toHaveTextContent("MAPP can't reach its server");
    await user.click(screen.getByRole('button', { name: 'Try again' }));

    expect(await screen.findAllByRole('article')).toHaveLength(3);
  });

  it('shows the table layout and keeps it in the address', async () => {
    const { user, router } = renderPage();

    await screen.findAllByRole('article');
    await user.click(screen.getByRole('button', { name: 'Table' }));

    const table = await screen.findByRole('table');
    expect(within(table).getAllByRole('row')).toHaveLength(4);
    expect(within(table).getByRole('link', { name: 'FY2026 Mobility China' })).toHaveAttribute(
      'href',
      '/plans/P-1002',
    );
    expect(router.state.location.search).toBe('?layout=table');
  });

  it('takes the filters from the address, else from what the person chose last time', async () => {
    remember('U-1002', { ...DEFAULT_FILTERS, status: 'published', layout: 'table' });
    const { api } = renderPage();

    expect(await screen.findByRole('table')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Remove filter: Published' })).toBeInTheDocument();
    expect(listQueries(api)[0]).toBe('?scope=all&status=published');
  });

  it('pins a plan to the top of the cards and saves the full list for the person (018 FR-001, FR-002, FR-004)', async () => {
    const { api, user } = renderWithPins();

    await screen.findAllByRole('article');
    await user.click(screen.getByRole('button', { name: "Pin plan 'FY2025 Lubricants Europe'" }));

    // The pinned plan sits first at once; the others keep the list's own order (FR-002).
    await waitFor(() =>
      expect(cardNames()).toEqual([
        'FY2025 Lubricants Europe',
        'FY2026 Lubricants Europe',
        'FY2026 Mobility China',
      ]),
    );
    // The full new list is saved for the person, like every display preference (FR-004).
    await waitFor(() =>
      expect(api.bodiesOf('PATCH /api/me/preferences')).toEqual([{ pinnedPlanIds: ['P-1003'] }]),
    );
    // The star reads back from the saved preferences: filled and renamed (FR-003).
    const star = screen.getByRole('button', { name: "Unpin plan 'FY2025 Lubricants Europe'" });
    expect(star.className).toMatch(/pinned/);
  });

  it('unpins a plan back into the list’s own order (018 FR-001)', async () => {
    const { api, user } = renderWithPins('P-1003');

    await screen.findAllByRole('article');
    expect(cardNames()[0]).toBe('FY2025 Lubricants Europe');
    await user.click(screen.getByRole('button', { name: "Unpin plan 'FY2025 Lubricants Europe'" }));

    await waitFor(() =>
      expect(cardNames()).toEqual([
        'FY2026 Lubricants Europe',
        'FY2026 Mobility China',
        'FY2025 Lubricants Europe',
      ]),
    );
    expect(api.bodiesOf('PATCH /api/me/preferences')).toEqual([{ pinnedPlanIds: [] }]);
    expect(
      screen.getByRole('button', { name: "Pin plan 'FY2025 Lubricants Europe'" }).className,
    ).not.toMatch(/pinned/);
  });

  it('puts pinned plans first in the table too, each group keeping the current sort (018 FR-002)', async () => {
    // Pinned in reverse order on purpose: the list's own sort, not the pin order, orders the group.
    const { user } = renderWithPins('P-1003', 'P-1002');

    await screen.findAllByRole('article');
    await user.click(screen.getByRole('button', { name: 'Table' }));

    const table = await screen.findByRole('table');
    expect(
      within(table)
        .getAllByRole('link', { name: /^FY/ })
        .map((link) => link.textContent),
    ).toEqual(['FY2026 Mobility China', 'FY2025 Lubricants Europe', 'FY2026 Lubricants Europe']);
    expect(
      within(table).getByRole('button', { name: "Unpin plan 'FY2026 Mobility China'" }),
    ).toBeInTheDocument();
    expect(
      within(table).getByRole('button', { name: "Pin plan 'FY2026 Lubricants Europe'" }),
    ).toBeInTheDocument();
  });

  it('simply doesn’t show a pinned plan the list no longer has (018 FR-007)', async () => {
    renderWithPins('P-9999');

    await screen.findAllByRole('article');

    expect(cardNames()).toEqual([
      'FY2026 Lubricants Europe',
      'FY2026 Mobility China',
      'FY2025 Lubricants Europe',
    ]);
    expect(screen.getByText('3 of 3 plans')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /^Unpin plan/ })).not.toBeInTheDocument();
  });

  it('offers View, Edit, Copy, Publish and Delete in a Draft plan’s actions menu', async () => {
    const { user, router } = renderPage();

    await screen.findAllByRole('article');
    await user.click(screen.getByRole('button', { name: 'Actions for FY2026 Mobility China' }));
    const items = (await screen.findAllByRole('menuitem')).map((item) => item.textContent);
    expect(items).toEqual(['View', 'Edit', 'Copy', 'Publish', 'Delete']);
    await user.click(screen.getByRole('menuitem', { name: 'Edit' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/plans/P-1002/edit'));
  });
});
