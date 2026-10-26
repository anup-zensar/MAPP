import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { ActionsMenu } from '@/ui/ActionsMenu/ActionsMenu';
import { mockApi } from '@/testing/mock-api';
import { findMessages, renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { destinations, launchFilm } from '@/testing/fixtures/activities';
import type { Activity } from './api';
import { useActivityActions } from './useActivityActions';

const DELETED = {
  recycleBinEntry: {
    id: 'BIN-1001',
    recordType: 'activity',
    recordTypeLabel: 'activity',
    recordId: 'A-1042',
    recordName: 'Launch film',
    path: [],
    parent: { type: 'initiative', id: 'I-1001', name: 'Digital & social' },
    deletedAt: '2026-03-12T09:00:00Z',
    deletedBy: { id: 'U-1002', name: 'Priya Raman' },
    contents: [],
  },
};

function More({ activity }: { activity: Activity }) {
  const more = useActivityActions(activity);
  return (
    <>
      <ActionsMenu label={activity.name} items={more.items} />
      {more.dialogs}
      {/* Stands in for the page's History section (feature 011), which View history brings up. */}
      <section id="history">
        <h2>History</h2>
      </section>
    </>
  );
}

function openMenu(answers: Parameters<typeof mockApi>[0] = {}) {
  const api = mockApi({
    'DELETE /api/v1/activities/A-1042': { body: DELETED },
    'POST /api/v1/recycle-bin/BIN-1001/undo': { body: {} },
    // No GSAP data for it yet: the delete question says nothing of transactions (015 FR-029).
    'GET /api/v1/financials/records/A-1042': {
      status: 404,
      body: {
        error: { code: 'not_found', message: 'No financial figures for A-1042.', fields: [] },
      },
    },
    ...answers,
  });
  const rendered = renderWithProviders(<More activity={launchFilm()} />, {
    route: '/plans/P-1001/activities/A-1042',
    user: testUser(),
  });
  return { ...rendered, api };
}

describe('useActivityActions (feature 006, FR-029 – FR-032)', () => {
  it('offers Copy, Move, Split activity, View history and Delete, last after a line', async () => {
    const { user } = openMenu();

    await user.click(screen.getByRole('button', { name: 'Actions for Launch film' }));
    expect((await screen.findAllByRole('menuitem')).map((item) => item.textContent)).toEqual([
      'Copy',
      'Move',
      'Split activity',
      'View history',
      'Delete',
    ]);
  });

  it('copies from More and opens the copy (US5#2, FR-030)', async () => {
    const { user, router } = openMenu({
      'GET /api/v1/activities/A-1042/destinations': { body: destinations() },
      'POST /api/v1/activities/A-1042/copy': {
        status: 201,
        body: launchFilm({ id: 'A-1043', name: 'Launch film (copy)' }),
      },
    });

    await user.click(screen.getByRole('button', { name: 'Actions for Launch film' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Copy' }));
    const dialog = await screen.findByRole('dialog', { name: "Copy activity 'Launch film'" });
    await within(dialog).findByRole('group', { name: 'Copy to' });
    await user.click(within(dialog).getByRole('button', { name: 'Copy activity' }));

    await waitFor(() =>
      expect(router.state.location.pathname).toBe('/plans/P-1001/activities/A-1043'),
    );
    expect(await findMessages()).toHaveTextContent("Activity 'Launch film (copy)' created");
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
  });

  it('moves from More (US5#1, FR-029)', async () => {
    const { user, api } = openMenu({
      'GET /api/v1/activities/A-1042/destinations': { body: destinations() },
      'POST /api/v1/activities/A-1042/move': {
        body: launchFilm({
          initiative: { type: 'initiative', id: 'I-1002', name: 'Print & outdoor' },
        }),
      },
    });

    await user.click(screen.getByRole('button', { name: 'Actions for Launch film' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Move' }));
    const dialog = await screen.findByRole('dialog', { name: "Move activity 'Launch film'" });
    await user.click(await within(dialog).findByRole('radio', { name: 'Print & outdoor' }));
    await user.click(within(dialog).getByRole('button', { name: 'Move activity' }));

    expect(await findMessages()).toHaveTextContent(
      "Activity 'Launch film' moved to 'Print & outdoor'",
    );
    expect(api.bodiesOf('POST /api/v1/activities/A-1042/move')).toEqual([
      { initiativeId: 'I-1002', version: 3 },
    ]);
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
  });

  it('deletes after asking, opens the plan page and offers Undo, which brings it back', async () => {
    const { user, api, router } = openMenu();

    await user.click(screen.getByRole('button', { name: 'Actions for Launch film' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Delete' }));
    const question = await screen.findByRole('alertdialog', {
      name: "Delete activity 'Launch film'?",
    });
    expect(api.bodiesOf('DELETE /api/v1/activities/A-1042')).toEqual([]);
    await user.click(within(question).getByRole('button', { name: 'Delete activity' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/plans/P-1001'));
    expect(
      await screen.findByText("Activity 'Launch film' moved to the recycle bin"),
    ).toBeVisible();
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();

    const messages = await findMessages();
    await user.click(within(messages).getByRole('button', { name: 'Undo' }));
    expect(await screen.findByText("Activity 'Launch film' restored")).toBeVisible();
    expect(api.requests.map((request) => `${request.method} ${request.path}`)).toContain(
      'POST /api/v1/recycle-bin/BIN-1001/undo',
    );
  });

  it('says why a delete did not work', async () => {
    const { user, router } = openMenu({
      'DELETE /api/v1/activities/A-1042': {
        status: 404,
        body: {
          error: {
            code: 'not_found',
            message: 'This activity no longer exists — someone may have deleted it.',
            fields: [],
          },
        },
      },
    });

    await user.click(screen.getByRole('button', { name: 'Actions for Launch film' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Delete' }));
    await user.click(await screen.findByRole('button', { name: 'Delete activity' }));

    expect(
      await screen.findByText('This activity no longer exists — someone may have deleted it.'),
    ).toBeVisible();
    expect(router.state.location.pathname).toBe('/plans/P-1001/activities/A-1042');
  });

  it('brings the page’s History section up from View history (FR-032, 011 FR-011)', async () => {
    const { user, router } = openMenu();

    await user.click(screen.getByRole('button', { name: 'Actions for Launch film' }));
    await user.click(await screen.findByRole('menuitem', { name: 'View history' }));

    await waitFor(() =>
      expect(screen.getByRole('heading', { level: 2, name: 'History' })).toHaveFocus(),
    );
    expect(router.state.location.pathname).toBe('/plans/P-1001/activities/A-1042');
  });
});
