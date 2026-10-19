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

function renderPage(route = '/plans', list: MockHandler = { body: seedList() }) {
  const api = mockApi({ 'GET /api/plans': list });
  const result = renderWithProviders(<PlansPage />, { user: testUser(), route, path: '/plans' });
  return { ...result, api };
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
    expect(within(summary).getByText('Draft').parentElement).toHaveTextContent('2');
    expect(within(summary).getByText('Published').parentElement).toHaveTextContent('1');
    expect(within(summary).getByRole('button', { name: '$2,260,000.00' })).toHaveTextContent(
      '$2.26M',
    );
    expect(within(summary).getByRole('button', { name: '$2,400,000.00' })).toBeInTheDocument();
    expect(screen.getByText('3 of 3')).toBeInTheDocument();
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
      within(screen.getByRole('group', { name: 'Status' })).getByRole('button', { name: 'Draft' }),
    );
    await user.selectOptions(screen.getByRole('combobox', { name: 'Year' }), '2026');

    await waitFor(() => expect(cardNames()).toEqual(['FY2026 Mobility China']));
    expect(screen.getByText('1 of 3')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Remove filter: Draft' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Remove filter: 2026' })).toBeInTheDocument();
    expect(router.state.location.search).toBe('?status=draft&year=2026');
    expect(listQueries(api)).toContain('?scope=all&status=draft&year=2026');

    await user.click(screen.getByRole('button', { name: 'Remove filter: Draft' }));
    await waitFor(() => expect(router.state.location.search).toBe('?year=2026'));
  });

  it('searches name or ID once typing stops (US2#3)', async () => {
    const { api, user } = renderPage();

    await screen.findAllByRole('article');
    await user.type(screen.getByRole('searchbox', { name: 'Search name or ID' }), 'mobility');

    await waitFor(() => expect(listQueries(api)).toContain('?scope=all&q=mobility'));
    expect(listQueries(api)).not.toContain('?scope=all&q=m');
    expect(
      await screen.findByRole('button', { name: 'Remove filter: Search: mobility' }),
    ).toBeInTheDocument();
  });

  it('shows my teams with the hint (US2#4, FR-004)', async () => {
    const { api, user } = renderPage();

    await screen.findAllByRole('article');
    await user.click(screen.getByRole('button', { name: 'My teams' }));

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
    expect(screen.getByText('0 of 3')).toBeInTheDocument();
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
    expect(await screen.findByRole('alert')).toHaveTextContent("NorthStar can't reach its server");
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

  it('offers View, Edit, Publish and Delete in a Draft plan’s actions menu', async () => {
    const { user, router } = renderPage();

    await screen.findAllByRole('article');
    await user.click(screen.getByRole('button', { name: 'Actions for FY2026 Mobility China' }));
    const items = (await screen.findAllByRole('menuitem')).map((item) => item.textContent);
    expect(items).toEqual(['View', 'Edit', 'Publish', 'Delete']);
    await user.click(screen.getByRole('menuitem', { name: 'Edit' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/plans/P-1002/edit'));
  });
});
