import { screen, waitFor, within } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { mockApi } from '@/test/mockApi';
import type { MockHandler } from '@/test/mockApi';
import { renderWithProviders } from '@/test/render';
import { testUser } from '@/test/users';
import { ActivityLogPage } from './ActivityLogPage';
import { TODAY_NOON, logEntry, logPage, logSummary } from './testData';

const LOG = 'GET /api/v1/activity-log';
const SUMMARY = 'GET /api/v1/activity-log/summary';

const NOT_FOUND_BODY = {
  error: {
    code: 'internal_error',
    message: 'Something went wrong. Please try again.',
  },
};

function renderPage(route: string, answers: Record<string, MockHandler>) {
  const api = mockApi({ [SUMMARY]: { body: logSummary() }, ...answers });
  const rendered = renderWithProviders(<ActivityLogPage />, {
    route,
    path: '/activity-log',
    user: testUser(),
  });
  return { ...rendered, api };
}

/** The query strings GET /api/v1/activity-log was asked with, in order. */
function logSearches(api: ReturnType<typeof mockApi>): URLSearchParams[] {
  return api.requests
    .filter((request) => request.method === 'GET' && request.path === '/api/v1/activity-log')
    .map((request) => new URLSearchParams(request.search));
}

describe('ActivityLogPage', () => {
  beforeEach(() => {
    // Only the date is fixed (Monday 19 Oct 2026, noon UTC); timers run as usual.
    vi.useFakeTimers({ toFake: ['Date'], now: TODAY_NOON });
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('shows the header and names the tab, with no create or edit action (FR-001, FR-010)', async () => {
    renderPage('/activity-log', { [LOG]: { body: logPage([logEntry()]) } });

    expect(screen.getByRole('heading', { level: 1, name: 'Activity log' })).toBeInTheDocument();
    expect(
      screen.getByText("See who changed what, and when. Entries can't be edited or removed."),
    ).toBeInTheDocument();
    expect(document.title).toBe('Activity log · MAPP');
    expect(await screen.findByRole('link', { name: 'Film shoot' })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /edit|delete|remove entry/i })).toBeNull();
  });

  it('shows the four summary cards for today and this week, in local time (FR-002)', async () => {
    const { api } = renderPage('/activity-log', { [LOG]: { body: logPage([logEntry()]) } });

    for (const name of [
      'Changes today: 5 — show these entries',
      'Changes this week: 23 — show these entries',
      'People active this week: 2 — show these entries',
      'Deletions this week: 1 — show these entries',
    ]) {
      expect(await screen.findByRole('link', { name })).toBeInTheDocument();
    }
    expect(screen.getByRole('button', { name: 'About Changes today' })).toBeInTheDocument();
    const summary = api.requests.find((request) => request.path === '/api/v1/activity-log/summary');
    expect(new URLSearchParams(summary?.search)).toEqual(
      new URLSearchParams({
        todayStart: '2026-10-19T00:00:00.000Z',
        weekStart: '2026-10-19T00:00:00.000Z',
      }),
    );
  });

  it('applies a card’s filters when it is clicked (FR-002)', async () => {
    const { user, router, api } = renderPage('/activity-log', {
      [LOG]: { body: logPage([logEntry()]) },
    });

    await user.click(
      await screen.findByRole('link', { name: 'Deletions this week: 1 — show these entries' }),
    );

    await waitFor(() =>
      expect(router.state.location.search).toBe(
        '?range=week&action=deleted&action=deletedPermanently',
      ),
    );
    await waitFor(() => expect(logSearches(api)).toHaveLength(2));
    const search = logSearches(api)[1];
    expect(search.getAll('action')).toEqual(['deleted', 'deletedPermanently']);
    expect(search.get('from')).toBe('2026-10-19T00:00:00.000Z');
  });

  it('asks the API for the filters in the address and counts the matches (FR-003)', async () => {
    const { api } = renderPage('/activity-log?person=U-1003&action=deleted&q=film', {
      [LOG]: { body: logPage([logEntry()], { total: 1, unfilteredTotal: 140 }) },
    });

    expect(await screen.findByText('1 of 140 entries')).toBeInTheDocument();
    const search = logSearches(api)[0];
    expect(search.get('actorId')).toBe('U-1003');
    expect(search.getAll('action')).toEqual(['deleted']);
    expect(search.get('q')).toBe('film');
    expect(search.get('limit')).toBe('50');
    expect(
      screen.getByRole('button', { name: 'Remove filter: Person: Daniel Okoro' }),
    ).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Remove filter: Action: Deleted' })).toBeVisible();
  });

  it('counts every entry when no filter is in use (FR-003)', async () => {
    renderPage('/activity-log', {
      [LOG]: { body: logPage([logEntry()], { total: 140, unfilteredTotal: 140 }) },
    });

    expect(await screen.findByText('140 of 140 entries')).toBeInTheDocument();
  });

  it('shows the Record filter of a History link as a chip named after the record (FR-013)', async () => {
    const { api } = renderPage('/activity-log?record=A-1042&inside=1', {
      [LOG]: {
        body: logPage([logEntry()], {
          record: { type: 'activity', id: 'A-1042', name: 'Film shoot' },
        }),
      },
    });

    expect(
      await screen.findByRole('button', {
        name: "Remove filter: Record: Activity 'Film shoot' (A-1042) and items inside",
      }),
    ).toBeInTheDocument();
    const search = logSearches(api)[0];
    expect(search.get('recordId')).toBe('A-1042');
    expect(search.get('includeInside')).toBe('true');
  });

  it('clears every filter at once (FR-003)', async () => {
    const { user, router } = renderPage('/activity-log?person=U-1003&range=week', {
      [LOG]: { body: logPage([logEntry()], { unfilteredTotal: 140 }) },
    });

    await user.click(await screen.findByRole('button', { name: 'Clear filters' }));

    await waitFor(() => expect(router.state.location.search).toBe(''));
  });

  it('groups the entries by day (FR-004)', async () => {
    renderPage('/activity-log', {
      [LOG]: {
        body: logPage([
          logEntry({ id: 'LOG-2', seq: 2 }),
          logEntry({ id: 'LOG-1', seq: 1, at: '2026-10-18T09:00:00Z' }),
        ]),
      },
    });

    expect(await screen.findByRole('heading', { name: 'Today · 1 entry' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Yesterday · 1 entry' })).toBeInTheDocument();
  });

  it('loads 50 more entries before the last one shown (FR-009)', async () => {
    const first = Array.from({ length: 50 }, (_, index) =>
      logEntry({ id: `LOG-${200 - index}`, seq: 200 - index }),
    );
    const { user, api } = renderPage('/activity-log', {
      [LOG]: (request) => {
        const beforeSeq = new URL(request.url).searchParams.get('beforeSeq');
        return beforeSeq
          ? { body: logPage([logEntry({ id: 'LOG-150', seq: 150 })], { total: 51 }) }
          : { body: logPage(first, { total: 51 }) };
      },
    });

    expect(await screen.findByText('Showing the last 50 of 51')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Load more' }));

    await waitFor(() => expect(screen.getAllByRole('listitem')).toHaveLength(51));
    expect(logSearches(api)[1].get('beforeSeq')).toBe('151');
    expect(screen.queryByRole('button', { name: 'Load more' })).not.toBeInTheDocument();
  });

  it('shows placeholders while the log loads', () => {
    renderPage('/activity-log', {
      [SUMMARY]: () => new Promise<never>(() => {}),
      [LOG]: () => new Promise<never>(() => {}),
    });

    expect(screen.getAllByRole('status').length).toBeGreaterThan(4);
  });

  it('says why the log could not be loaded, with Try again', async () => {
    let calls = 0;
    const { user } = renderPage('/activity-log', {
      [LOG]: () => {
        calls += 1;
        return calls === 1
          ? { status: 500, body: NOT_FOUND_BODY }
          : { body: logPage([logEntry()]) };
      },
    });

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Something went wrong. Please try again.',
    );
    await user.click(screen.getByRole('button', { name: 'Try again' }));

    expect(await screen.findByText('1 of 1 entry')).toBeInTheDocument();
  });

  it('says when the log is still empty', async () => {
    renderPage('/activity-log', {
      [LOG]: { body: logPage([], { total: 0, unfilteredTotal: 0 }) },
    });

    expect(await screen.findByText('No changes yet')).toBeInTheDocument();
    expect(
      screen.getByText('Changes to plans, settings and people appear here.'),
    ).toBeInTheDocument();
  });

  it('says when the filters hide everything, with Clear filters (FR-003)', async () => {
    const { user, router } = renderPage('/activity-log?q=zzz', {
      [LOG]: { body: logPage([], { total: 0, unfilteredTotal: 140 }) },
    });

    const empty = (await screen.findByText('Nothing matches these filters')).parentElement!;
    await user.click(within(empty).getByRole('button', { name: 'Clear filters' }));

    await waitFor(() => expect(router.state.location.search).toBe(''));
  });

  it('asks nothing while the custom range ends before it starts (FR-003)', async () => {
    const { api } = renderPage('/activity-log?range=custom&from=2026-10-12&to=2026-10-01', {
      [LOG]: { body: logPage([logEntry()]) },
    });

    expect(
      await screen.findByText('The end date must be on or after the start date.'),
    ).toBeInTheDocument();
    expect(logSearches(api)).toHaveLength(0);
  });
});
