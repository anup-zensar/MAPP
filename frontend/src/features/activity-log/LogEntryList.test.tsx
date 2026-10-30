import { screen, within } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { closestTo, nth } from '@/testing/dom';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { TODAY_NOON, logEntry } from '@/testing/fixtures/activity-log';
import { LogEntryList } from './LogEntryList';
import type { LogEntryListProps } from './LogEntryList';

const ENTRIES = [
  logEntry({ id: 'LOG-5', seq: 5, at: '2026-10-19T11:00:00Z', recordName: 'Five' }),
  logEntry({ id: 'LOG-4', seq: 4, at: '2026-10-19T08:00:00Z', recordName: 'Four' }),
  logEntry({ id: 'LOG-3', seq: 3, at: '2026-10-18T17:00:00Z', recordName: 'Three' }),
  logEntry({ id: 'LOG-2', seq: 2, at: '2026-10-12T10:00:00Z', recordName: 'Two' }),
];

function renderList(props: Partial<LogEntryListProps> = {}, user = testUser()) {
  return renderWithProviders(
    <LogEntryList
      entries={ENTRIES}
      total={4}
      hasMore={false}
      onLoadMore={() => {}}
      isLoadingMore={false}
      {...props}
    />,
    { user },
  );
}

describe('LogEntryList', () => {
  beforeEach(() => {
    // Only the date is fixed; timers run as usual.
    vi.useFakeTimers({ toFake: ['Date'], now: TODAY_NOON });
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('groups the entries by day, newest first, each day with its count (FR-004)', () => {
    renderList();

    const headings = screen.getAllByRole('heading', { level: 2 });
    expect(headings.map((heading) => heading.textContent)).toEqual([
      'Today · 2 entries',
      'Yesterday · 1 entry',
      'Monday 12 Oct 2026 · 1 entry',
    ]);
    const today = closestTo(nth(headings, 0), 'section');
    expect(
      within(today)
        .getAllByRole('listitem')
        .map((item) => within(item).getByRole('link').textContent),
    ).toEqual(['Five', 'Four']);
  });

  it("038 FR-058: day headings follow the person's date format", () => {
    const user = testUser();
    renderList({}, { ...user, preferences: { ...user.preferences, dateFormat: 'YYYY-MM-DD' } });

    expect(screen.getAllByRole('heading', { level: 2 }).map((h) => h.textContent)).toEqual([
      'Today · 2 entries',
      'Yesterday · 1 entry',
      'Monday 2026-10-12 · 1 entry',
    ]);
  });

  it('says how many are loaded and offers Load more while there are more (FR-009)', async () => {
    const onLoadMore = vi.fn();
    const { user } = renderList({ total: 1204, hasMore: true, onLoadMore });

    expect(screen.getByText('Showing the last 4 of 1,204')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Load more' }));
    expect(onLoadMore).toHaveBeenCalledTimes(1);
  });

  it('has no Load more when everything is loaded (FR-009)', () => {
    renderList();

    expect(screen.queryByRole('button', { name: 'Load more' })).not.toBeInTheDocument();
    expect(screen.queryByText(/Showing the last/)).not.toBeInTheDocument();
  });
});
