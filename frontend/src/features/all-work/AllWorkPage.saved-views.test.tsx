import { act, screen, waitFor, within } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import type { HttpHandler } from 'msw';
import { nth } from '@/testing/dom';
import { gridResult, savedView } from '@/testing/fixtures/all-work';
import { budgetResult } from '@/testing/fixtures/all-work-budget';
import { renderAllWorkPage, viewQueries } from '@/testing/fixtures/all-work-page';
import { http, reply } from '@/testing/replies';
import { resetAllWorkStore } from './all-work-store';
import type { SavedView } from './all-work-types';

// More than the 500 ms the page waits before it saves the person's columns and sort.
const PAST_THE_SAVE_DELAY_MS = 600;

afterEach(() => {
  resetAllWorkStore();
  vi.restoreAllMocks();
  vi.useRealTimers();
});

describe('AllWorkPage saved views (US6)', () => {
  const declined = savedView();

  /**
   * The saved-view answers of a small in-memory server: the list answers what was saved and
   * changed, as the API does.
   */
  function viewAnswers(initial = [declined]): HttpHandler[] {
    let items: SavedView[] = [...initial];
    return [
      reply('GET /api/v1/saved-views', () => ({ body: { items, total: items.length } })),
      http.post('/api/v1/saved-views', async ({ request, response }) => {
        const body = await request.json();
        const view = savedView({
          id: 'SV-1009',
          name: body.name,
          state: body.state,
          isDefault: body.isDefault ?? false,
        });
        items = [...items, view];
        return response(201).json(view);
      }),
      http.patch('/api/v1/saved-views/{viewId}', async ({ params, request, response }) => {
        const body = await request.json();
        items = items.map((view) => (view.id === params.viewId ? { ...view, ...body } : view));
        const updated = items.find((view) => view.id === params.viewId);
        return updated ? response(200).json(updated) : undefined;
      }),
    ];
  }

  async function chooseView(user: ReturnType<typeof renderAllWorkPage>['user'], name: string) {
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
    const { api } = renderAllWorkPage(viewAnswers([byDefault]));

    expect(
      await screen.findByText(
        'Some parts of this view no longer exist and were left out: Owner: Jane Doe.',
      ),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'View: Declined activities Modified' }),
    ).toBeVisible();
    await waitFor(() => expect(viewQueries(api)).toHaveLength(1));
    const query = nth(viewQueries(api), 0);
    expect(query.getAll('activityStatus')).toEqual(['declined']);
    expect(query.has('owner')).toBe(false);
  });

  it('saves what is shown under a name and marks it as the active view (US6 scenario 1, FR-029)', async () => {
    const { api, user } = renderAllWorkPage([
      ...viewAnswers([]),
      reply('GET /api/v1/all-work', (request) =>
        new URL(request.url).searchParams.get('view') === 'budget'
          ? { body: budgetResult() }
          : { body: gridResult() },
      ),
    ]);
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
    expect(api.bodiesOf('POST /api/v1/saved-views')).toEqual([
      expect.objectContaining({ state: expect.objectContaining({ view: 'budget' }) }),
    ]);
  });

  it('marks the active view Modified and updates it with what is shown (US6 scenario 2, FR-030)', async () => {
    const { api, user } = renderAllWorkPage(viewAnswers());
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
    expect(api.bodiesOf('PATCH /api/v1/saved-views/SV-1001')).toEqual([
      expect.objectContaining({
        state: expect.objectContaining({
          filters: expect.objectContaining({ activityStatuses: [] }),
        }),
      }),
    ]);
    expect(await screen.findByRole('button', { name: 'View: Declined activities' })).toBeVisible();
  });

  it('does not remember columns or sort while a view is applied (FR-021)', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    const { api, user } = renderAllWorkPage(viewAnswers());
    await screen.findByRole('link', { name: 'Helix launch' });
    await chooseView(user, 'Declined activities');
    await screen.findByRole('button', { name: 'View: Declined activities' });

    await user.click(screen.getByRole('button', { name: 'Planned cost' }));

    expect(
      await screen.findByRole('button', { name: 'View: Declined activities Modified' }),
    ).toBeVisible();
    await act(() => vi.advanceTimersByTimeAsync(PAST_THE_SAVE_DELAY_MS));
    expect(api.bodiesOf('PUT /api/v1/all-work/preferences')).toEqual([]);
  });

  it('goes back to the standard view (FR-009)', async () => {
    const { api, user } = renderAllWorkPage(viewAnswers());
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
