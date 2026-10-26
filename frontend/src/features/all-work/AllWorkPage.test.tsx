import { screen, waitFor, within } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { mockApi } from '@/testing/mockApi';
import type { MockHandler } from '@/testing/mockApi';
import { findMessages, renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/users';
import { AllWorkPage } from './AllWorkPage';
import { resetAllWorkStore } from './allWorkStore';
import type {
  AllWorkResult,
  AllWorkRow,
  SavedView,
  SavedViewCreate,
  SavedViewUpdate,
  ViewState,
} from './allWorkTypes';
import {
  allWorkOptions,
  budgetResult,
  calendarResult,
  defaultPreferences,
  emptyResult,
  gridResult,
  PAGE_ANSWERS,
  row,
  savedView,
} from './testData';

afterEach(() => {
  resetAllWorkStore();
  vi.restoreAllMocks();
});

function renderPage(
  answers: Record<string, MockHandler> = {},
  route = '/all-work',
): ReturnType<typeof renderWithProviders> & { api: ReturnType<typeof mockApi> } {
  const api = mockApi({
    ...PAGE_ANSWERS,
    'GET /api/v1/all-work': { body: gridResult() },
    ...answers,
  });
  const rendered = renderWithProviders(<AllWorkPage />, {
    user: testUser(),
    route,
    path: '/all-work',
  });
  return { ...rendered, api };
}

/** The query strings sent to GET /api/v1/all-work, in order. */
function viewQueries(api: ReturnType<typeof mockApi>): URLSearchParams[] {
  return api.requests
    .filter((request) => request.method === 'GET' && request.path === '/api/v1/all-work')
    .map((request) => new URLSearchParams(request.search));
}

function rowNames(): string[] {
  const table = screen.getByRole('table', { name: 'All work grid' });
  return within(table)
    .getAllByRole('rowheader')
    .map((header) => within(header).queryByRole('link')?.textContent ?? header.textContent ?? '');
}

describe('AllWorkPage (US1)', () => {
  it('shows the header, the six cards and the Grid of the current year (FR-001, FR-003, FR-009)', async () => {
    const { api } = renderPage();

    expect(screen.getByRole('heading', { level: 1, name: 'All work' })).toBeInTheDocument();
    expect(screen.getByText('Filter, compare and export all planned work.')).toBeInTheDocument();
    expect(document.title).toBe('All work · MAPP');
    // Nothing is created here: Views and Export are the page's only actions (FR-002).
    expect(await screen.findByRole('button', { name: 'Export' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Views' })).toBeInTheDocument();

    expect(
      await screen.findByRole('link', { name: 'FY2026 Lubricants Europe' }),
    ).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: 'Grid' })).toHaveAttribute('aria-selected', 'true');
    const summary = screen.getByRole('region', { name: 'Summary' });
    expect(within(summary).getByRole('button', { name: '$17K ($17,000.00)' })).toBeInTheDocument();
    expect(rowNames()).toEqual([
      'FY2026 Lubricants Europe',
      'Helix launch',
      'Helix digital',
      'Helix launch event',
      'Helix social ads',
      'Rimula fleet',
      'Fleet days',
      'Fleet roadshow',
      'FY2026 Mobility China',
      'Retail push',
      'Total · 3 activities',
    ]);

    const [query] = viewQueries(api);
    expect(query.get('year')).toBe('2026');
    expect(query.get('view')).toBe('grid');
    expect(query.getAll('column')).toEqual(defaultPreferences().columns);
  });

  it('starts from the year the API calls current and the person’s remembered columns (FR-009, FR-019)', async () => {
    const { api } = renderPage({
      'GET /api/v1/all-work/options': { body: allWorkOptions({ year: 2027, currentYear: 2027 }) },
      'GET /api/v1/all-work/preferences': {
        body: { columns: ['plannedCost', 'id'], sort: null },
      },
    });

    await waitFor(() => expect(viewQueries(api)).toHaveLength(1));
    const [query] = viewQueries(api);
    expect(query.get('year')).toBe('2027');
    expect(query.getAll('column')).toEqual(['plannedCost', 'id']);
  });

  it('shows grey shapes while loading, then the error with Try again', async () => {
    let calls = 0;
    const { user } = renderPage({
      'GET /api/v1/all-work': () => {
        calls += 1;
        return calls === 1 ? { status: 500 } : { body: gridResult() };
      },
    });

    expect(screen.getAllByText('Loading…').length).toBeGreaterThan(0);
    expect(await screen.findByRole('alert')).toHaveTextContent("MAPP can't reach its server");
    await user.click(screen.getByRole('button', { name: 'Try again' }));

    expect(await screen.findByRole('link', { name: 'Helix launch' })).toBeInTheDocument();
  });

  it('says that the year has no plans yet, with the way to the Plans page', async () => {
    renderPage({
      'GET /api/v1/all-work/options': { body: allWorkOptions({ plans: [] }) },
      'GET /api/v1/all-work': {
        body: emptyResult({ count: { shown: 0, total: 0, noun: 'activities' } }),
      },
    });

    expect(await screen.findByText('No plans in 2026 yet')).toBeInTheDocument();
    expect(screen.getByText("Plans of 2026 appear here once they're created.")).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Go to Plans' })).toHaveAttribute('href', '/plans');
  });

  it('applies the filters of the address once, and offers Clear filters when nothing matches (D-122)', async () => {
    const { api, user, router } = renderPage(
      {
        'GET /api/v1/all-work': (request) =>
          new URL(request.url).searchParams.has('activityStatus')
            ? { body: emptyResult() }
            : { body: gridResult() },
      },
      '/all-work?activityStatus=complete&owner=U-1003',
    );

    expect(await screen.findByText('Nothing matches these filters')).toBeInTheDocument();
    expect(screen.getByText('Try removing a filter or changing the search.')).toBeInTheDocument();
    expect(viewQueries(api)[0].getAll('activityStatus')).toEqual(['complete']);
    expect(viewQueries(api)[0].getAll('owner')).toEqual(['U-1003']);
    expect(router.state.location.search).toBe('');

    await user.click(screen.getByRole('button', { name: 'Clear filters' }));

    expect(await screen.findByRole('link', { name: 'Helix launch' })).toBeInTheDocument();
    const last = viewQueries(api).at(-1);
    expect(last?.has('activityStatus')).toBe(false);
    expect(last?.get('year')).toBe('2026');
  });

  it('opens and closes everything, and hides the unassigned rows with the switch (FR-011, FR-012)', async () => {
    const { user } = renderPage();
    await screen.findByRole('link', { name: 'Helix launch' });

    await user.click(screen.getByRole('button', { name: 'Expand all' }));
    expect(rowNames()).toContain('Venue hire');
    expect(
      rowNames().filter((name) => name === 'Unassigned planned cost (Sys_Gen_Balance)'),
    ).toHaveLength(2);

    const unassigned = screen.getByRole('switch', { name: 'Unassigned planned cost' });
    expect(unassigned).toHaveAttribute('aria-checked', 'true');
    await user.click(unassigned);
    expect(unassigned).toHaveAttribute('aria-checked', 'false');
    expect(rowNames()).not.toContain('Unassigned planned cost (Sys_Gen_Balance)');
    expect(rowNames()).toContain('Venue hire');

    await user.click(screen.getByRole('button', { name: 'Collapse all' }));
    expect(rowNames()).toEqual([
      'FY2026 Lubricants Europe',
      'FY2026 Mobility China',
      'Total · 3 activities',
    ]);
  });

  it('says when the data was read and asks again with Refresh', async () => {
    const { api, user } = renderPage();

    expect(await screen.findByText('Updated at 14:05')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Refresh' }));

    await waitFor(() => expect(viewQueries(api)).toHaveLength(2));
  });

  it('opens a row’s record after checking that it still exists (FR-016, D-123)', async () => {
    const { api, user, router } = renderPage({
      'GET /api/v1/all-work/records/I-1001': {
        body: { recordId: 'I-1001', level: 'initiative', name: 'Helix digital', planId: 'P-1001' },
      },
    });

    await user.click(await screen.findByRole('link', { name: 'Helix digital' }));

    await waitFor(() =>
      expect(router.state.location.pathname).toBe('/plans/P-1001/initiatives/I-1001'),
    );
    expect(api.requests.map((request) => request.path)).toContain(
      '/api/v1/all-work/records/I-1001',
    );
  });

  it('says that a record was deleted and refreshes the view (edge case, D-123)', async () => {
    const { api, user, router } = renderPage({
      'GET /api/v1/all-work/records/A-1002': {
        status: 404,
        body: {
          error: {
            code: 'not_found',
            message: 'This activity was deleted — it is in the recycle bin',
          },
        },
      },
    });

    await user.click(await screen.findByRole('link', { name: 'Helix social ads' }));

    const messages = await findMessages();
    expect(
      within(messages).getByText('This activity was deleted — it is in the recycle bin'),
    ).toBeInTheDocument();
    await waitFor(() => expect(viewQueries(api)).toHaveLength(2));
    expect(router.state.location.pathname).toBe('/all-work');
  });

  it('keeps the view, the expansion and the scroll position when coming back (FR-009, FR-016)', async () => {
    const scrollTo = vi.spyOn(window, 'scrollTo').mockImplementation(() => {});
    const { api, user, router } = renderPage({
      'GET /api/v1/all-work/records/L-1001': {
        body: {
          recordId: 'L-1001',
          level: 'lineItem',
          name: 'Venue hire',
          planId: 'P-1001',
          activityId: 'A-1001',
        },
      },
    });
    await user.click(await screen.findByRole('button', { name: 'Expand Helix launch event' }));
    Object.defineProperty(window, 'scrollY', { value: 640, configurable: true });

    await user.click(screen.getByRole('link', { name: 'Venue hire' }));
    await waitFor(() => expect(router.state.location.pathname).toBe('/line-items/L-1001'));
    await router.navigate('/all-work');

    expect(await screen.findByRole('link', { name: 'Venue hire' })).toBeInTheDocument();
    expect(scrollTo).toHaveBeenCalledWith(0, 640);
    // The view was kept: the first visit's answer is reused, nothing is asked again.
    expect(viewQueries(api)).toHaveLength(1);
    Object.defineProperty(window, 'scrollY', { value: 0, configurable: true });
  });

  it('shows the rows 50 to a page, with the totals row on every page (D-115)', async () => {
    const result = manyActivities(60);
    const { user } = renderPage({ 'GET /api/v1/all-work': { body: result } });

    expect(await screen.findByText('Rows 1–50 of 63')).toBeInTheDocument();
    expect(rowNames()).toHaveLength(51);
    await user.click(screen.getByRole('button', { name: 'Next' }));

    expect(screen.getByText('Rows 51–63 of 63')).toBeInTheDocument();
    expect(
      screen.getByText('Continued: FY2026 Lubricants Europe › Helix launch › Helix digital'),
    ).toBeInTheDocument();
    expect(screen.getByRole('rowheader', { name: 'Total · 60 activities' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Next' })).toBeDisabled();

    await user.selectOptions(screen.getByRole('combobox', { name: 'Rows per page' }), '100');
    expect(screen.getByText('Rows 1–63 of 63')).toBeInTheDocument();
  });

  it('names a refused filter in a message and sets it back (plan "States")', async () => {
    const { api } = renderPage(
      {
        'GET /api/v1/all-work': (request) =>
          new URL(request.url).searchParams.has('search')
            ? {
                status: 400,
                body: {
                  error: {
                    code: 'validation_failed',
                    message: 'Use 200 characters or fewer.',
                    fields: [{ field: 'search', message: 'Use 200 characters or fewer.' }],
                  },
                },
              }
            : { body: gridResult() },
      },
      '/all-work?search=helix',
    );

    const messages = await findMessages();
    expect(within(messages).getByText('Use 200 characters or fewer.')).toBeInTheDocument();
    expect(await screen.findByRole('link', { name: 'Helix launch' })).toBeInTheDocument();
    expect(viewQueries(api).at(-1)?.has('search')).toBe(false);
  });
});

describe('AllWorkPage filters and search (US2)', () => {
  /** The answer to "Activity status: Declined": one branch, its parents as context. */
  function declinedOnly(): AllWorkResult {
    const keep = ['P-1001', 'C-1001', 'I-1001', 'A-1002', 'L-1002', 'A-1002:unassigned'];
    return gridResult({
      rows: gridResult()
        .rows.filter((candidate) => keep.includes(candidate.key))
        .map((candidate) =>
          ['P-1001', 'C-1001', 'I-1001'].includes(candidate.key)
            ? { ...candidate, role: 'context' as const }
            : candidate,
        ),
      summary: { ...gridResult().summary, campaigns: 0, initiatives: 0, activities: 1 },
      count: { shown: 1, total: 3, noun: 'activities' },
    });
  }

  it('applies a filter at once: the rows, the chip, the count and the cards follow (US2 scenario 2, FR-004, FR-005)', async () => {
    const { api, user } = renderPage({
      'GET /api/v1/all-work': (request) =>
        new URL(request.url).searchParams.get('activityStatus') === 'declined'
          ? { body: declinedOnly() }
          : { body: gridResult() },
    });
    expect(await screen.findByText('3 of 3 activities')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Activity status' }));
    await user.click(
      within(await screen.findByRole('dialog', { name: 'Activity status' })).getByRole('checkbox', {
        name: 'Declined',
      }),
    );
    await user.keyboard('{Escape}');

    expect(await screen.findByText('1 of 3 activities')).toBeInTheDocument();
    expect(viewQueries(api).at(-1)?.getAll('activityStatus')).toEqual(['declined']);
    expect(
      screen.getByRole('button', { name: 'Remove filter: Activity status: Declined' }),
    ).toBeInTheDocument();
    expect(rowNames()).toEqual([
      'FY2026 Lubricants Europe',
      'Helix launch',
      'Helix digital',
      'Helix social ads',
      'Total · 3 activities',
    ]);
    expect(screen.getAllByText('Shown for context')).toHaveLength(3);
    const summary = screen.getByRole('region', { name: 'Summary' });
    expect(within(summary).getByText('Campaigns').parentElement).toHaveTextContent('0');
  });

  it('finds an activity by its ID: its path opens and the text is marked (US2 scenario 1, FR-006, SC-001)', async () => {
    const found = gridResult({
      rows: declinedOnly().rows.map((candidate) =>
        candidate.key === 'A-1002'
          ? { ...candidate, role: 'match' as const, searchHit: { fields: ['id' as const] } }
          : candidate,
      ),
      count: { shown: 1, total: 3, noun: 'activities' },
    });
    const { api, user } = renderPage({
      'GET /api/v1/all-work': (request) =>
        new URL(request.url).searchParams.get('search') === 'A-1002'
          ? { body: found }
          : { body: gridResult() },
    });
    await screen.findByText('3 of 3 activities');

    await user.type(screen.getByRole('searchbox', { name: 'Search' }), 'A-1002');

    expect(await screen.findByText('1 of 3 activities')).toBeInTheDocument();
    expect(viewQueries(api).map((query) => query.get('search'))).not.toContain('A');
    // The path to the hit is open; the activity itself stays closed.
    expect(rowNames()).toEqual([
      'FY2026 Lubricants Europe',
      'Helix launch',
      'Helix digital',
      'Helix social ads',
      'Total · 3 activities',
    ]);
    const activity = screen
      .getByRole('rowheader', { name: /Helix social ads/ })
      .closest('tr') as HTMLElement;
    expect([...activity.querySelectorAll('mark')].map((mark) => mark.textContent)).toEqual([
      'A-1002',
    ]);
  });

  it('keeps the filters and the search while the person moves around MAPP (FR-009)', async () => {
    const { api, user, router } = renderPage({
      'GET /api/v1/all-work': (request) =>
        new URL(request.url).searchParams.has('activityStatus')
          ? { body: declinedOnly() }
          : { body: gridResult() },
    });
    await screen.findByText('3 of 3 activities');
    await user.click(screen.getByRole('button', { name: 'Activity status' }));
    await user.click(
      within(await screen.findByRole('dialog')).getByRole('checkbox', { name: 'Declined' }),
    );
    await screen.findByText('1 of 3 activities');

    await router.navigate('/plans');
    await screen.findByText('Another page');
    await router.navigate('/all-work');

    expect(await screen.findByText('1 of 3 activities')).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'Remove filter: Activity status: Declined' }),
    ).toBeInTheDocument();
    expect(viewQueries(api)).toHaveLength(2);
  });

  it('counts initiatives under "Initiatives without activities" (US2 scenario 4, FR-007, FR-008)', async () => {
    const result = gridResult({
      rows: gridResult().rows.filter((candidate) => ['P-1002', 'C-1003'].includes(candidate.key)),
      count: { shown: 1, total: 2, noun: 'initiatives' },
    });
    renderPage(
      { 'GET /api/v1/all-work': { body: result } },
      '/all-work?showLevels=initiativesWithoutActivities',
    );

    expect(await screen.findByText('1 of 2 initiatives')).toBeInTheDocument();
    expect(
      screen.getByRole('button', {
        name: 'Remove filter: Show levels: Initiatives without activities',
      }),
    ).toBeInTheDocument();
  });
});

describe('AllWorkPage columns and sort (US3)', () => {
  it('shows a column ticked in the chooser and remembers the choice for the person (US3 scenario 1, FR-021)', async () => {
    const { api, user } = renderPage({
      'PUT /api/v1/all-work/preferences': (request) =>
        request.json().then((body: unknown) => ({ body })),
    });
    await screen.findByRole('link', { name: 'Helix launch' });

    await user.click(screen.getByRole('button', { name: 'Columns' }));
    const panel = await screen.findByRole('dialog', { name: 'Columns' });
    await user.click(within(panel).getByRole('checkbox', { name: 'Cluster' }));

    await waitFor(() =>
      expect(viewQueries(api).at(-1)?.getAll('column')).toEqual([
        ...defaultPreferences().columns,
        'attr.SA-1001',
      ]),
    );
    await waitFor(() =>
      expect(api.bodiesOf('PUT /api/v1/all-work/preferences')).toEqual([
        { columns: [...defaultPreferences().columns, 'attr.SA-1001'], sort: null },
      ]),
    );
  });

  it('asks for the rows sorted by a heading, and remembers the sort (US3 scenario 2, FR-015)', async () => {
    const { api, user } = renderPage({
      'PUT /api/v1/all-work/preferences': (request) =>
        request.json().then((body: unknown) => ({ body })),
    });
    await screen.findByRole('link', { name: 'Helix launch' });

    await user.click(screen.getByRole('button', { name: 'Planned cost' }));
    await waitFor(() => expect(viewQueries(api).at(-1)?.get('sort')).toBe('plannedCost:asc'));
    await user.click(screen.getByRole('button', { name: 'Planned cost' }));
    await waitFor(() => expect(viewQueries(api).at(-1)?.get('sort')).toBe('plannedCost:desc'));

    await waitFor(() =>
      expect(api.bodiesOf('PUT /api/v1/all-work/preferences').at(-1)).toEqual({
        columns: defaultPreferences().columns,
        sort: { column: 'plannedCost', direction: 'desc' },
      }),
    );
  });
});

describe('AllWorkPage Budget & cost (US4)', () => {
  /** Answers each view with its own result. */
  function byView(request: Request) {
    const params = new URL(request.url).searchParams;
    if (params.get('view') === 'budget') {
      return { body: budgetResult(params.get('period') === 'quarter' ? 'quarter' : 'month') };
    }
    return { body: gridResult() };
  }

  it('switches views keeping the filters, the search and the expansion (FR-001)', async () => {
    const { api, user } = renderPage({ 'GET /api/v1/all-work': byView });
    await user.click(await screen.findByRole('button', { name: 'Expand Helix launch event' }));
    await user.click(screen.getByRole('button', { name: 'Plan status' }));
    await user.click(
      within(await screen.findByRole('dialog', { name: 'Plan status' })).getByRole('checkbox', {
        name: 'Published',
      }),
    );
    await user.keyboard('{Escape}');

    await user.click(screen.getByRole('tab', { name: 'Budget & cost' }));

    const budget = await screen.findByRole('table', { name: 'Budget & cost' });
    expect(within(budget).getByRole('rowheader', { name: /Venue hire/ })).toBeInTheDocument();
    const query = viewQueries(api).at(-1);
    expect(query?.get('view')).toBe('budget');
    expect(query?.get('period')).toBe('month');
    expect(query?.getAll('planStatus')).toEqual(['published']);
    expect(query?.has('column')).toBe(false);
    expect(
      screen.getByRole('button', { name: 'Remove filter: Plan status: Published' }),
    ).toBeVisible();
    expect(screen.queryByRole('button', { name: 'Columns' })).not.toBeInTheDocument();

    await user.click(screen.getByRole('tab', { name: 'Grid' }));
    const grid = await screen.findByRole('table', { name: 'All work grid' });
    expect(within(grid).getByRole('rowheader', { name: /Venue hire/ })).toBeInTheDocument();
  });

  it('shows quarters with the Monthly · Quarterly switch (US4 scenario 3, FR-022)', async () => {
    const { api, user } = renderPage({ 'GET /api/v1/all-work': byView });
    await screen.findByRole('link', { name: 'Helix launch' });
    await user.click(screen.getByRole('tab', { name: 'Budget & cost' }));
    await screen.findByRole('columnheader', { name: 'Mar' });

    const period = screen.getByRole('group', { name: 'Period' });
    expect(within(period).getByRole('button', { name: 'Monthly' })).toHaveAttribute(
      'aria-pressed',
      'true',
    );
    await user.click(within(period).getByRole('button', { name: 'Quarterly' }));

    expect(await screen.findByRole('columnheader', { name: 'Q2' })).toBeInTheDocument();
    expect(viewQueries(api).at(-1)?.get('period')).toBe('quarter');
    const footer = screen
      .getByRole('rowheader', { name: 'Total · 3 activities' })
      .closest('tr') as HTMLElement;
    expect(within(footer).getByText('$14,000.00')).toBeInTheDocument();
    expect(within(footer).getByText('$18,000.00')).toBeInTheDocument();
  });
});

describe('AllWorkPage Calendar (US5)', () => {
  it('shows the Calendar timeline, and the Month layout with the switch (FR-025, FR-028)', async () => {
    const { api, user } = renderPage({
      'GET /api/v1/all-work': (request) =>
        new URL(request.url).searchParams.get('view') === 'calendar'
          ? { body: calendarResult() }
          : { body: gridResult() },
    });
    await screen.findByRole('link', { name: 'Helix launch' });

    await user.click(screen.getByRole('tab', { name: 'Calendar' }));

    expect(await screen.findByRole('table', { name: 'Calendar 2026' })).toBeInTheDocument();
    const query = viewQueries(api).at(-1);
    expect(query?.get('view')).toBe('calendar');
    expect(query?.has('column')).toBe(false);
    expect(query?.has('period')).toBe(false);
    expect(
      screen.queryByRole('switch', { name: 'Unassigned planned cost' }),
    ).not.toBeInTheDocument();

    const layout = screen.getByRole('group', { name: 'Calendar layout' });
    await user.click(within(layout).getByRole('button', { name: 'Month' }));

    expect(screen.getByRole('heading', { name: 'October 2026' })).toBeInTheDocument();
    expect(within(layout).getByRole('button', { name: 'Month' })).toHaveAttribute(
      'aria-pressed',
      'true',
    );
    await user.click(within(layout).getByRole('button', { name: 'Timeline' }));
    expect(screen.getByRole('table', { name: 'Calendar 2026' })).toBeInTheDocument();
  });
});

describe('AllWorkPage saved views (US6)', () => {
  const declined = savedView();

  /**
   * The saved-view answers of a small in-memory server: the list answers what was saved and
   * changed, as the API does.
   */
  function viewAnswers(initial = [declined]): Record<string, MockHandler> {
    let items: SavedView[] = [...initial];
    return {
      'GET /api/v1/saved-views': () => ({ body: { items, total: items.length } }),
      'POST /api/v1/saved-views': async (request) => {
        const body = (await request.json()) as SavedViewCreate;
        const view = savedView({
          id: 'SV-1009',
          name: body.name,
          state: body.state,
          isDefault: body.isDefault ?? false,
        });
        items = [...items, view];
        return { status: 201, body: view };
      },
      'PATCH /api/v1/saved-views/SV-1001': async (request) => {
        const body = (await request.json()) as SavedViewUpdate;
        items = items.map((view) => (view.id === 'SV-1001' ? { ...view, ...body } : view));
        return { body: items.find((view) => view.id === 'SV-1001') };
      },
    };
  }

  async function chooseView(user: ReturnType<typeof renderPage>['user'], name: string) {
    await user.click(screen.getByRole('button', { name: /^View/ }));
    await user.click(await screen.findByRole('menuitemcheckbox', { name }));
  }

  it('starts a new visit from the default view and names what was left out (US6 scenario 3, FR-009, FR-031)', async () => {
    const byDefault = savedView({
      isDefault: true,
      state: {
        ...declined.state,
        filters: {
          ...declined.state.filters,
          owners: [{ id: 'U-0999', label: 'Jane Doe' }],
        },
      },
    });
    const { api } = renderPage(viewAnswers([byDefault]));

    expect(
      await screen.findByText(
        'Some parts of this view no longer exist and were left out: Owner: Jane Doe.',
      ),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'View: Declined activities Modified' }),
    ).toBeVisible();
    await waitFor(() => expect(viewQueries(api)).toHaveLength(1));
    const [query] = viewQueries(api);
    expect(query.getAll('activityStatus')).toEqual(['declined']);
    expect(query.has('owner')).toBe(false);
  });

  it('saves what is shown under a name and marks it as the active view (US6 scenario 1, FR-029)', async () => {
    const { api, user } = renderPage({
      ...viewAnswers([]),
      'GET /api/v1/all-work': (request) =>
        new URL(request.url).searchParams.get('view') === 'budget'
          ? { body: budgetResult() }
          : { body: gridResult() },
    });
    await screen.findByRole('link', { name: 'Helix launch' });
    await user.click(screen.getByRole('tab', { name: 'Budget & cost' }));
    await screen.findByRole('table', { name: 'Budget & cost' });

    await user.click(screen.getByRole('button', { name: 'Views' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Save current view…' }));
    const dialog = await screen.findByRole('dialog', { name: 'Save current view' });
    await user.type(within(dialog).getByRole('textbox', { name: 'Name' }), 'Lubricants by month');
    await user.click(within(dialog).getByRole('button', { name: 'Save view' }));

    expect(await screen.findByText("View 'Lubricants by month' saved")).toBeInTheDocument();
    expect(await screen.findByRole('button', { name: 'View: Lubricants by month' })).toBeVisible();
    const [body] = api.bodiesOf('POST /api/v1/saved-views') as { state: ViewState }[];
    expect(body.state.view).toBe('budget');
  });

  it('marks the active view Modified and updates it with what is shown (US6 scenario 2, FR-030)', async () => {
    const { api, user } = renderPage(viewAnswers());
    await screen.findByRole('link', { name: 'Helix launch' });
    await chooseView(user, 'Declined activities');
    await waitFor(() =>
      expect(viewQueries(api).at(-1)?.getAll('activityStatus')).toEqual(['declined']),
    );
    expect(screen.getByRole('button', { name: 'View: Declined activities' })).toBeVisible();

    await user.click(
      screen.getByRole('button', { name: 'Remove filter: Activity status: Declined' }),
    );
    const modified = await screen.findByRole('button', {
      name: 'View: Declined activities Modified',
    });
    await user.click(modified);
    await user.click(await screen.findByRole('menuitem', { name: 'Update view' }));

    expect(await screen.findByText("View 'Declined activities' updated")).toBeInTheDocument();
    const [body] = api.bodiesOf('PATCH /api/v1/saved-views/SV-1001') as { state: ViewState }[];
    expect(body.state.filters.activityStatuses).toEqual([]);
    expect(await screen.findByRole('button', { name: 'View: Declined activities' })).toBeVisible();
  });

  it('does not remember columns or sort while a view is applied (FR-021)', async () => {
    const { api, user } = renderPage(viewAnswers());
    await screen.findByRole('link', { name: 'Helix launch' });
    await chooseView(user, 'Declined activities');
    await screen.findByRole('button', { name: 'View: Declined activities' });

    await user.click(screen.getByRole('button', { name: 'Planned cost' }));

    expect(
      await screen.findByRole('button', { name: 'View: Declined activities Modified' }),
    ).toBeVisible();
    await new Promise((resolve) => setTimeout(resolve, 600));
    expect(api.bodiesOf('PUT /api/v1/all-work/preferences')).toEqual([]);
  });

  it('goes back to the standard view (FR-009)', async () => {
    const { api, user } = renderPage(viewAnswers());
    await screen.findByRole('link', { name: 'Helix launch' });
    await chooseView(user, 'Declined activities');
    await screen.findByRole('button', { name: 'View: Declined activities' });

    await user.click(screen.getByRole('button', { name: 'View: Declined activities' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Standard view' }));

    expect(await screen.findByRole('button', { name: 'Views' })).toBeVisible();
    // The standard view's answer was read at the start and is shown again from memory.
    expect(
      screen.queryByRole('button', { name: 'Remove filter: Activity status: Declined' }),
    ).not.toBeInTheDocument();
    expect(await screen.findByText('3 of 3 activities')).toBeInTheDocument();
    expect(viewQueries(api)).toHaveLength(2);
  });
});

/** A plan with one campaign and initiative holding `count` activities. */
function manyActivities(count: number): AllWorkResult {
  const [plan, campaign, initiative] = gridResult().rows;
  const activities: AllWorkRow[] = Array.from({ length: count }, (_, index) =>
    row({
      key: `A-${2000 + index}`,
      level: 'activity',
      name: `Activity ${index + 1}`,
      depth: 3,
      parentKey: 'I-1001',
      planId: 'P-1001',
      status: 'created',
      owner: 'Priya Raman',
    }),
  );
  return gridResult({
    rows: [plan, campaign, initiative, ...activities],
    totals: { ...gridResult().totals, activities: count },
  });
}
