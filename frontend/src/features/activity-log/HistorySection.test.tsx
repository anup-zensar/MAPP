import { screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { HttpHandler } from 'msw';
import type { PlanningLevel } from '@/ui/LevelTag/LevelTag';
import { nth } from '@/testing/dom';
import { mockApi } from '@/testing/mock-api';
import { findMessages, renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { logEntry, logPage } from '@/testing/fixtures/activity-log';
import { reply } from '@/testing/replies';
import { HistorySection } from './HistorySection';
import { showHistory } from './show-history';

const PREFERENCES = 'PATCH /api/v1/me/preferences';

function renderSection(
  answers: HttpHandler[],
  { level = 'activity', include = false }: { level?: PlanningLevel; include?: boolean } = {},
) {
  const api = mockApi(answers);
  const user = testUser();
  const rendered = renderWithProviders(
    <HistorySection recordType={level} recordId="A-1042" recordName="Film shoot" level={level} />,
    { user: { ...user, preferences: { ...user.preferences, historyIncludeInside: include } } },
  );
  return { ...rendered, api };
}

function logSearches(api: ReturnType<typeof mockApi>): URLSearchParams[] {
  return api.requests
    .filter((request) => request.method === 'GET' && request.path === '/api/v1/activity-log')
    .map((request) => new URLSearchParams(request.search));
}

describe('HistorySection', () => {
  it('lists the record’s entries, 20 at a time, with Load more (FR-011)', async () => {
    const first = Array.from({ length: 20 }, (_, index) =>
      logEntry({ id: `LOG-${100 - index}`, seq: 100 - index }),
    );
    const { user, api } = renderSection([
      reply('GET /api/v1/activity-log', (request) =>
        new URL(request.url).searchParams.get('beforeSeq')
          ? { body: logPage([logEntry({ id: 'LOG-50', seq: 50 })], { total: 21 }) }
          : { body: logPage(first, { total: 21 }) },
      ),
    ]);

    expect(screen.getByRole('heading', { level: 2, name: 'History' })).toBeInTheDocument();
    expect(screen.getByText('Every change to this activity, newest first')).toBeInTheDocument();
    expect(await screen.findByText('Showing the last 20 of 21')).toBeInTheDocument();
    expect(nth(screen.getAllByRole('heading', { level: 3 }), 0)).toHaveTextContent(/entries$/);
    const search = nth(logSearches(api), 0);
    expect(search.get('recordId')).toBe('A-1042');
    expect(search.get('limit')).toBe('20');
    expect(search.has('includeInside')).toBe(false);

    await user.click(screen.getByRole('button', { name: 'Load more' }));

    await waitFor(() => expect(screen.getAllByRole('listitem')).toHaveLength(21));
    expect(nth(logSearches(api), 1).get('beforeSeq')).toBe('81');
  });

  it.each([
    ['plan', 'Include items inside'],
    ['campaign', 'Include items inside'],
    ['initiative', 'Include items inside'],
    ['activity', 'Include line items'],
  ] as const)('offers %s pages "%s" (FR-012)', (level, label) => {
    renderSection([reply('GET /api/v1/activity-log', { body: logPage([]) })], { level });

    expect(screen.getByRole('checkbox', { name: label })).not.toBeChecked();
  });

  it('offers no include option on a line item (FR-012)', () => {
    renderSection([reply('GET /api/v1/activity-log', { body: logPage([]) })], {
      level: 'lineItem',
    });

    expect(screen.queryByRole('checkbox')).not.toBeInTheDocument();
    expect(screen.getByText('Every change to this line item, newest first')).toBeInTheDocument();
  });

  it('includes the line items at once and remembers the choice for the person (FR-012, FR-013)', async () => {
    const { user, api } = renderSection([
      reply('GET /api/v1/activity-log', { body: logPage([logEntry()]) }),
      reply('PATCH /api/v1/me/preferences', { status: 204 }),
    ]);
    expect(screen.getByRole('link', { name: 'Open in activity log' })).toHaveAttribute(
      'href',
      '/activity-log?record=A-1042',
    );

    await user.click(screen.getByRole('checkbox', { name: 'Include line items' }));

    expect(screen.getByRole('checkbox', { name: 'Include line items' })).toBeChecked();
    await waitFor(() =>
      expect(api.bodiesOf(PREFERENCES)).toEqual([{ historyIncludeInside: true }]),
    );
    await waitFor(() => expect(logSearches(api)).toHaveLength(2));
    expect(nth(logSearches(api), 1).get('includeInside')).toBe('true');
    expect(screen.getByRole('link', { name: 'Open in activity log' })).toHaveAttribute(
      'href',
      '/activity-log?record=A-1042&inside=1',
    );
  });

  it('starts with the person’s remembered choice (FR-012)', async () => {
    const { api } = renderSection(
      [reply('GET /api/v1/activity-log', { body: logPage([logEntry()]) })],
      { include: true },
    );

    expect(screen.getByRole('checkbox', { name: 'Include line items' })).toBeChecked();
    await waitFor(() => expect(nth(logSearches(api), 0).get('includeInside')).toBe('true'));
  });

  it('says when the choice could not be saved, with Try again (FR-012)', async () => {
    const { user } = renderSection([
      reply('GET /api/v1/activity-log', { body: logPage([logEntry()]) }),
      reply('PATCH /api/v1/me/preferences', { status: 500 }),
    ]);

    await user.click(screen.getByRole('checkbox', { name: 'Include line items' }));

    const messages = await findMessages();
    expect(messages).toHaveTextContent("Your choice couldn't be saved.");
    expect(screen.getByRole('button', { name: 'Try again' })).toBeInTheDocument();
  });

  it('says when nothing was recorded yet', async () => {
    renderSection([reply('GET /api/v1/activity-log', { body: logPage([]) })]);

    expect(await screen.findByText('No changes recorded yet.')).toBeInTheDocument();
  });

  it('says why the history could not be loaded, with Try again', async () => {
    renderSection([reply('GET /api/v1/activity-log', { status: 500 })]);

    expect(await screen.findByRole('alert')).toHaveTextContent(
      "SP&A One can't reach its server right now.",
    );
    expect(screen.getByRole('button', { name: 'Try again' })).toBeInTheDocument();
  });

  it('is where View history takes people: its heading gets the focus (FR-011)', async () => {
    renderSection([reply('GET /api/v1/activity-log', { body: logPage([]) })]);

    showHistory();

    await waitFor(() =>
      expect(screen.getByRole('heading', { level: 2, name: 'History' })).toHaveFocus(),
    );
  });

  it('draws as a part of a side panel, with an h3 and its own id (FR-011)', async () => {
    mockApi([reply('GET /api/v1/activity-log', { body: logPage([logEntry()]) })]);
    renderWithProviders(
      <HistorySection
        recordType="campaign"
        recordId="C-1001"
        recordName="Helix launch"
        level="campaign"
        id="history-C-1001"
        variant="panel"
      />,
      { user: testUser() },
    );

    const heading = screen.getByRole('heading', { level: 3, name: 'History' });
    expect(screen.getByRole('region', { name: 'History' })).toBeInTheDocument();
    expect(screen.getByText('Every change to this campaign, newest first')).toBeInTheDocument();
    // The day headings sit one level below the section's heading.
    expect(await screen.findByRole('heading', { level: 4, name: /· 1 entry$/ })).toBeVisible();

    showHistory('history-C-1001');

    await waitFor(() => expect(heading).toHaveFocus());
    expect(heading.closest('#history-C-1001')).not.toBeNull();
  });
});
