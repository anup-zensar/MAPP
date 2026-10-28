import { screen, waitFor, within } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { nth } from '@/testing/dom';
import { findMessages } from '@/testing/render';
import {
  allWorkOptions,
  defaultPreferences,
  emptyResult,
  gridResult,
} from '@/testing/fixtures/all-work';
import {
  manyActivities,
  renderAllWorkPage,
  rowNames,
  viewQueries,
} from '@/testing/fixtures/all-work-page';
import { reply } from '@/testing/replies';
import { resetAllWorkStore } from './all-work-store';

afterEach(() => {
  resetAllWorkStore();
  vi.restoreAllMocks();
});

describe('AllWorkPage (US1)', () => {
  it('shows the header, the six cards and the Grid of the current year (FR-001, FR-003, FR-009)', async () => {
    const { api } = renderAllWorkPage();

    expect(screen.getByRole('heading', { level: 1, name: 'All work' })).toBeInTheDocument();
    expect(screen.getByText('Filter, compare and export all planned work.')).toBeInTheDocument();
    expect(document.title).toBe('All work · SP&A One');
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

    const query = nth(viewQueries(api), 0);
    expect(query.get('year')).toBe('2026');
    expect(query.get('view')).toBe('grid');
    expect(query.getAll('column')).toEqual(defaultPreferences().columns);
  });

  it('starts from the year the API calls current and the person’s remembered columns (FR-009, FR-019)', async () => {
    const { api } = renderAllWorkPage([
      reply('GET /api/v1/all-work/options', {
        body: allWorkOptions({ year: 2027, currentYear: 2027 }),
      }),
      reply('GET /api/v1/all-work/preferences', {
        body: { columns: ['plannedCost', 'id'], sort: null },
      }),
    ]);

    await waitFor(() => expect(viewQueries(api)).toHaveLength(1));
    const query = nth(viewQueries(api), 0);
    expect(query.get('year')).toBe('2027');
    expect(query.getAll('column')).toEqual(['plannedCost', 'id']);
  });

  it('shows grey shapes while loading, then the error with Try again', async () => {
    let calls = 0;
    const { user } = renderAllWorkPage([
      reply('GET /api/v1/all-work', () => {
        calls += 1;
        return calls === 1 ? { status: 500 } : { body: gridResult() };
      }),
    ]);

    expect(screen.getAllByText('Loading…').length).toBeGreaterThan(0);
    expect(await screen.findByRole('alert')).toHaveTextContent("SP&A One can't reach its server");
    await user.click(screen.getByRole('button', { name: 'Try again' }));

    expect(await screen.findByRole('link', { name: 'Helix launch' })).toBeInTheDocument();
  });

  it('says that the year has no plans yet, with the way to the Plans page', async () => {
    renderAllWorkPage([
      reply('GET /api/v1/all-work/options', { body: allWorkOptions({ plans: [] }) }),
      reply('GET /api/v1/all-work', {
        body: emptyResult({ count: { shown: 0, total: 0, noun: 'activities' } }),
      }),
    ]);

    expect(await screen.findByText('No plans in 2026 yet')).toBeInTheDocument();
    expect(screen.getByText("Plans of 2026 appear here once they're created.")).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Go to Plans' })).toHaveAttribute('href', '/plans');
  });

  it('applies the filters of the address once, and offers Clear filters when nothing matches (D-122)', async () => {
    const { api, user, router } = renderAllWorkPage(
      [
        reply('GET /api/v1/all-work', (request) =>
          new URL(request.url).searchParams.has('activityStatus')
            ? { body: emptyResult() }
            : { body: gridResult() },
        ),
      ],
      '/all-work?activityStatus=complete&owner=U-1003',
    );

    expect(await screen.findByText('Nothing matches these filters')).toBeInTheDocument();
    expect(screen.getByText('Try removing a filter or changing the search.')).toBeInTheDocument();
    expect(nth(viewQueries(api), 0).getAll('activityStatus')).toEqual(['complete']);
    expect(nth(viewQueries(api), 0).getAll('owner')).toEqual(['U-1003']);
    expect(router.state.location.search).toBe('');

    await user.click(screen.getByRole('button', { name: 'Clear filters' }));

    expect(await screen.findByRole('link', { name: 'Helix launch' })).toBeInTheDocument();
    const last = viewQueries(api).at(-1);
    expect(last?.has('activityStatus')).toBe(false);
    expect(last?.get('year')).toBe('2026');
  });

  it('opens and closes everything, and hides the unassigned rows with the switch (FR-011, FR-012)', async () => {
    const { user } = renderAllWorkPage();
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
    const { api, user } = renderAllWorkPage();

    expect(await screen.findByText('Updated at 14:05')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Refresh' }));

    await waitFor(() => expect(viewQueries(api)).toHaveLength(2));
  });

  it('opens a row’s record after checking that it still exists (FR-016, D-123)', async () => {
    const { api, user, router } = renderAllWorkPage([
      reply(
        'GET /api/v1/all-work/records/{recordId}',
        {
          body: {
            recordId: 'I-1001',
            level: 'initiative',
            name: 'Helix digital',
            planId: 'P-1001',
          },
        },
        { recordId: 'I-1001' },
      ),
    ]);

    await user.click(await screen.findByRole('link', { name: 'Helix digital' }));

    await waitFor(() =>
      expect(router.state.location.pathname).toBe('/plans/P-1001/initiatives/I-1001'),
    );
    expect(api.requests.map((request) => request.path)).toContain(
      '/api/v1/all-work/records/I-1001',
    );
  });

  it('says that a record was deleted and refreshes the view (edge case, D-123)', async () => {
    const { api, user, router } = renderAllWorkPage([
      reply(
        'GET /api/v1/all-work/records/{recordId}',
        {
          status: 404,
          body: {
            error: {
              code: 'not_found',
              message: 'This activity was deleted — it is in the recycle bin',
              fields: [],
            },
          },
        },
        { recordId: 'A-1002' },
      ),
    ]);

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
    const { api, user, router } = renderAllWorkPage([
      reply(
        'GET /api/v1/all-work/records/{recordId}',
        {
          body: {
            recordId: 'L-1001',
            level: 'lineItem',
            name: 'Venue hire',
            planId: 'P-1001',
            activityId: 'A-1001',
          },
        },
        { recordId: 'L-1001' },
      ),
    ]);
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
    const { user } = renderAllWorkPage([reply('GET /api/v1/all-work', { body: result })]);

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
    const { api } = renderAllWorkPage(
      [
        reply('GET /api/v1/all-work', (request) =>
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
        ),
      ],
      '/all-work?search=helix',
    );

    const messages = await findMessages();
    expect(within(messages).getByText('Use 200 characters or fewer.')).toBeInTheDocument();
    expect(await screen.findByRole('link', { name: 'Helix launch' })).toBeInTheDocument();
    expect(viewQueries(api).at(-1)?.has('search')).toBe(false);
  });
});
