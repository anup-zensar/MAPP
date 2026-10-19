import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { ActionsMenu } from '@/components';
import { mockApi } from '@/test/mockApi';
import { findMessages, renderWithProviders } from '@/test/render';
import { testUser } from '@/test/users';
import type { Activity } from './activitiesApi';
import { launchFilm } from './testData';
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
    </>
  );
}

function openMenu(answers: Parameters<typeof mockApi>[0] = {}) {
  const api = mockApi({
    'DELETE /api/activities/A-1042': { body: DELETED },
    'POST /api/recycle-bin/BIN-1001/undo': { body: {} },
    ...answers,
  });
  const rendered = renderWithProviders(<More activity={launchFilm()} />, {
    route: '/plans/P-1001/activities/A-1042',
    user: testUser(),
  });
  return { ...rendered, api };
}

describe('useActivityActions (feature 006, FR-031, FR-032)', () => {
  it('deletes after asking, opens the plan page and offers Undo, which brings it back', async () => {
    const { user, api, router } = openMenu();

    await user.click(screen.getByRole('button', { name: 'Actions for Launch film' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Delete' }));
    const question = await screen.findByRole('alertdialog', {
      name: "Delete activity 'Launch film'?",
    });
    expect(api.bodiesOf('DELETE /api/activities/A-1042')).toEqual([]);
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
      'POST /api/recycle-bin/BIN-1001/undo',
    );
  });

  it('says why a delete did not work', async () => {
    const { user, router } = openMenu({
      'DELETE /api/activities/A-1042': {
        status: 404,
        body: {
          code: 'not_found',
          message: 'This activity no longer exists — someone may have deleted it.',
          fields: [],
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

  it('opens the activity log of the activity from View history (FR-032)', async () => {
    const { user, router } = openMenu();

    await user.click(screen.getByRole('button', { name: 'Actions for Launch film' }));
    await user.click(await screen.findByRole('menuitem', { name: 'View history' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/activity-log'));
    expect(router.state.location.search).toBe('?recordType=activity&recordId=A-1042');
  });
});
