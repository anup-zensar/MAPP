import { screen, waitFor, within } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { closestTo } from '@/testing/dom';
import { defaultPreferences, gridResult } from '@/testing/fixtures/all-work';
import { budgetResult, calendarResult } from '@/testing/fixtures/all-work-budget';
import {
  PREFERENCES_SAVED,
  renderAllWorkPage,
  viewQueries,
} from '@/testing/fixtures/all-work-page';
import { reply } from '@/testing/replies';
import { resetAllWorkStore } from './all-work-store';

afterEach(() => {
  resetAllWorkStore();
  vi.restoreAllMocks();
});

describe('AllWorkPage columns and sort (US3)', () => {
  it('shows a column ticked in the chooser and remembers the choice for the person (US3 scenario 1, FR-021)', async () => {
    const { api, user } = renderAllWorkPage([PREFERENCES_SAVED]);
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
    const { api, user } = renderAllWorkPage([PREFERENCES_SAVED]);
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
    const { api, user } = renderAllWorkPage([reply('GET /api/v1/all-work', byView)]);
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
    const { api, user } = renderAllWorkPage([reply('GET /api/v1/all-work', byView)]);
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
    const footer = closestTo(screen.getByRole('rowheader', { name: 'Total · 3 activities' }), 'tr');
    expect(within(footer).getByText('$14,000.00')).toBeInTheDocument();
    expect(within(footer).getByText('$18,000.00')).toBeInTheDocument();
  });
});

describe('AllWorkPage Calendar (US5)', () => {
  it('shows the Calendar timeline, and the Month layout with the switch (FR-025, FR-028)', async () => {
    const { api, user } = renderAllWorkPage([
      reply('GET /api/v1/all-work', (request) =>
        new URL(request.url).searchParams.get('view') === 'calendar'
          ? { body: calendarResult() }
          : { body: gridResult() },
      ),
    ]);
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
