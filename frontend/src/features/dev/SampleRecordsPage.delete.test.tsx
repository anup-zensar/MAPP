import { screen, waitFor, within } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { findMessages } from '@/testing/render';
import {
  AGENDA,
  AGENDA_CREATED,
  AGENDA_IN_BIN,
  chooseAction,
  IDEAS,
  IDEAS_CREATED,
  IDEAS_IN_BIN,
  IDEAS_RENAMED,
  itemWith,
  listOf,
  mockSampleRecords,
  renderSampleRecordsPage,
  UNREACHABLE,
} from '@/testing/fixtures/dev';
import { reply } from '@/testing/replies';

afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
  localStorage.clear();
  delete document.documentElement.dataset.theme;
});

describe('SampleRecordsPage: delete, Undo and restore', () => {
  it('deletes a folder with its notes after asking, and offers Undo (FR-024, FR-026, US5)', async () => {
    const api = mockSampleRecords({}, [
      reply(
        'DELETE /api/v1/sample-folders/{folderId}',
        { body: { recycleBinEntry: IDEAS_IN_BIN } },
        { folderId: 'SF-1001' },
      ),
    ]);
    const { user } = await renderSampleRecordsPage();

    await chooseAction(user, 'Ideas', 'Delete');
    const question = await screen.findByRole('alertdialog', {
      name: "Delete sample folder 'Ideas'?",
    });
    expect(question).toHaveAccessibleDescription(
      'It goes to the recycle bin together with its 2 sample notes. You can restore it.',
    );
    expect(within(question).getByRole('button', { name: 'Cancel' })).toHaveFocus();
    await user.click(within(question).getByRole('button', { name: 'Delete sample folder' }));

    const messages = await findMessages();
    expect(messages).toHaveTextContent("Sample folder 'Ideas' deleted");
    expect(within(messages).getByRole('button', { name: 'Undo' })).toBeInTheDocument();
    await waitFor(() =>
      expect(screen.getByRole('status')).toHaveTextContent(
        "Sample folder 'Ideas' deleted Restore it from Recently deleted on this page",
      ),
    );
    expect(api.requests).toContainEqual({
      method: 'DELETE',
      path: '/api/v1/sample-folders/SF-1001',
      search: '',
      body: undefined,
    });
  });

  it.each([
    [
      'its one note',
      { ...IDEAS, notes: [AGENDA] },
      'It goes to the recycle bin together with its sample note. You can restore it.',
    ],
    ['nothing', { ...IDEAS, notes: [] }, 'It goes to the recycle bin. You can restore it.'],
  ])(
    'says that a folder goes to the recycle bin with %s (plan "User-visible messages")',
    async (_case, folder, text) => {
      mockSampleRecords({ folders: { body: listOf([folder]) } });
      const { user } = await renderSampleRecordsPage();

      await chooseAction(user, 'Ideas', 'Delete');

      expect(
        await screen.findByRole('alertdialog', { name: "Delete sample folder 'Ideas'?" }),
      ).toHaveAccessibleDescription(text);
    },
  );

  it('keeps the folder when the delete is cancelled (design system §6.5)', async () => {
    const api = mockSampleRecords();
    const { user } = await renderSampleRecordsPage();
    await chooseAction(user, 'Ideas', 'Delete');

    await user.click(await screen.findByRole('button', { name: 'Cancel' }));

    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
    expect(api.requests.filter((request) => request.method === 'DELETE')).toEqual([]);
  });

  it('says when a delete cannot reach the server, and tries again (FR-020)', async () => {
    let attempts = 0;
    mockSampleRecords({}, [
      reply(
        'DELETE /api/v1/sample-folders/{folderId}',
        () =>
          ++attempts === 1
            ? Promise.reject(new TypeError('Failed to fetch'))
            : { body: { recycleBinEntry: IDEAS_IN_BIN } },
        { folderId: 'SF-1001' },
      ),
    ]);
    const { user } = await renderSampleRecordsPage();
    await chooseAction(user, 'Ideas', 'Delete');
    await user.click(await screen.findByRole('button', { name: 'Delete sample folder' }));

    const messages = await findMessages();
    await waitFor(() => expect(messages).toHaveTextContent(UNREACHABLE));
    await user.click(within(messages).getByRole('button', { name: 'Try again' }));

    await waitFor(() =>
      expect(messages).toHaveTextContent("Sample folder 'Ideas' deleted"),
    );
  });

  it('lists what was deleted, with what went with it, who deleted it and when (plan "Developer pages")', async () => {
    mockSampleRecords({ bin: { body: listOf([IDEAS_IN_BIN, AGENDA_IN_BIN]) } });
    await renderSampleRecordsPage();

    const card = screen.getByRole('region', { name: 'Recently deleted' });
    expect(
      within(card).getByText(
        "Sample folder 'Ideas' (SF-1001), with 2 sample notes — deleted by Priya Raman, 12 Mar 2026, 09:30",
      ),
    ).toBeInTheDocument();
    expect(
      within(card).getByText(
        "Sample note 'Agenda' (SN-1003) — deleted by Daniel Okoro, 12 Mar 2026, 09:20",
      ),
    ).toBeInTheDocument();
    expect(
      within(card).getByRole('button', { name: 'Restore sample folder Ideas' }),
    ).toHaveTextContent('Restore');
  });

  it('038 FR-052 (L-27): offers Restore only to people who may restore', async () => {
    const refused = { available: false, reason: "Restoring isn't part of your role (Viewer)." };
    const access = { restore: refused, deletePermanently: refused };
    mockSampleRecords({ bin: { body: listOf([{ ...IDEAS_IN_BIN, access }]) } });
    await renderSampleRecordsPage();

    const card = screen.getByRole('region', { name: 'Recently deleted' });
    expect(within(card).getByText(/Sample folder 'Ideas' \(SF-1001\)/)).toBeInTheDocument();
    expect(within(card).queryByRole('button', { name: /^Restore/ })).not.toBeInTheDocument();
  });

  it('restores from Recently deleted and says what the restore changed (FR-025)', async () => {
    const api = mockSampleRecords({ bin: { body: listOf([IDEAS_IN_BIN]) } }, [
      reply(
        'POST /api/v1/recycle-bin/{entryId}/restore',
        {
          body: {
            recordType: 'sampleFolder',
            recordTypeLabel: 'sample folder',
            recordId: 'SF-1001',
            recordName: 'Ideas (restored)',
            details: "Renamed to 'Ideas (restored)' because the name was taken.",
          },
        },
        { entryId: 'BIN-1001' },
      ),
    ]);
    const { user } = await renderSampleRecordsPage();

    await user.click(screen.getByRole('button', { name: 'Restore sample folder Ideas' }));

    expect(await findMessages()).toHaveTextContent(
      "Sample folder 'Ideas' restored. Renamed to 'Ideas (restored)' because the name was taken.",
    );
    expect(api.bodiesOf('POST /api/v1/recycle-bin/BIN-1001/restore')).toHaveLength(1);
  });

  it('says why a restore was refused (FR-025, US5 scenario 4)', async () => {
    mockSampleRecords({ bin: { body: listOf([AGENDA_IN_BIN]) } }, [
      reply(
        'POST /api/v1/recycle-bin/{entryId}/restore',
        {
          status: 409,
          body: {
            error: {
              code: 'restore_parent_first',
              message: "Restore 'Ideas' first",
              fields: [],
              parent: { type: 'sampleFolder', id: 'SF-1001', name: 'Ideas' },
            },
          },
        },
        { entryId: 'BIN-1002' },
      ),
    ]);
    const { user } = await renderSampleRecordsPage();

    await user.click(screen.getByRole('button', { name: 'Restore sample note Agenda' }));

    const messages = await findMessages();
    expect(messages).toHaveTextContent("Restore 'Ideas' first");
    expect(within(messages).queryByRole('button', { name: 'Try again' })).not.toBeInTheDocument();
  });
});

describe('SampleRecordsPage: activity log', () => {
  it('shows the latest 20 entries: who did what to which record, where, when and what changed (FR-022)', async () => {
    const api = mockSampleRecords({
      log: { body: listOf([AGENDA_CREATED, IDEAS_RENAMED, IDEAS_CREATED]) },
    });
    await renderSampleRecordsPage();

    const card = screen.getByRole('region', { name: 'Activity log' });
    const created = itemWith(card, "Priya Raman created sample folder 'Ideas' (SF-1001)");
    expect(created).toHaveTextContent('12 Mar 2026, 09:00');
    expect(created).not.toHaveTextContent('in ');

    const renamed = itemWith(card, "Daniel Okoro changed sample folder 'Ideas 2' (SF-1001)");
    expect(renamed).toHaveTextContent('12 Mar 2026, 14:32');
    expect(within(renamed).getByText('Name: Ideas → Ideas 2')).toBeInTheDocument();

    const noteCreated = itemWith(card, "Priya Raman created sample note 'Agenda' (SN-1003)");
    expect(noteCreated).toHaveTextContent('in Ideas 2');

    expect(api.requests).toContainEqual({
      method: 'GET',
      path: '/api/v1/activity-log',
      search: '?limit=20',
      body: undefined,
    });
  });
});
