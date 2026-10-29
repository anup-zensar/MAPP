import { screen, within } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { findMessages } from '@/testing/render';
import {
  AGENDA,
  AGENDA_IN_BIN,
  chooseAction,
  mockSampleRecords,
  renderSampleRecordsPage,
} from '@/testing/fixtures/dev';
import { reply } from '@/testing/replies';

afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
  localStorage.clear();
  delete document.documentElement.dataset.theme;
});

describe('SampleRecordsPage: notes', () => {
  it('adds a note to a folder from the actions menu and says so (US5)', async () => {
    const api = mockSampleRecords({}, [
      reply(
        'POST /api/v1/sample-folders/{folderId}/notes',
        {
          status: 201,
          body: { ...AGENDA, id: 'SN-1005', title: 'Budget' },
        },
        { folderId: 'SF-1001' },
      ),
    ]);
    const { user } = await renderSampleRecordsPage();

    await chooseAction(user, 'Ideas', 'Add note');
    const panel = await screen.findByRole('dialog', { name: 'Add note' });
    await user.type(within(panel).getByLabelText('Title'), 'Budget');
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(await findMessages()).toHaveTextContent("Sample note 'Budget' added");
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    expect(api.bodiesOf('POST /api/v1/sample-folders/SF-1001/notes')).toEqual([
      { title: 'Budget' },
    ]);
  });

  it('shows the API’s problem with a note title at the field (FR-020)', async () => {
    mockSampleRecords({}, [
      reply(
        'POST /api/v1/sample-folders/{folderId}/notes',
        {
          status: 400,
          body: {
            error: {
              code: 'validation_failed',
              message: 'Some fields need your attention.',
              fields: [{ field: 'title', message: 'Enter a title.' }],
            },
          },
        },
        { folderId: 'SF-1001' },
      ),
    ]);
    const { user } = await renderSampleRecordsPage();
    await chooseAction(user, 'Ideas', 'Add note');
    const panel = await screen.findByRole('dialog', { name: 'Add note' });

    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(await within(panel).findByRole('alert')).toHaveTextContent('Enter a title.');
    expect(within(panel).getByLabelText('Title')).toHaveAccessibleDescription(
      'required Enter a title.',
    );
  });

  it('deletes a note after asking, and offers Undo (FR-024, FR-026)', async () => {
    const api = mockSampleRecords({}, [
      reply(
        'DELETE /api/v1/sample-notes/{noteId}',
        { body: { recycleBinEntry: AGENDA_IN_BIN } },
        { noteId: 'SN-1003' },
      ),
    ]);
    const { user } = await renderSampleRecordsPage();

    await user.click(screen.getByRole('button', { name: "Delete note 'Agenda'" }));
    const question = await screen.findByRole('alertdialog', {
      name: "Delete sample note 'Agenda'?",
    });
    expect(question).toHaveAccessibleDescription('It goes to the recycle bin. You can restore it.');
    await user.click(within(question).getByRole('button', { name: 'Delete sample note' }));

    const messages = await findMessages();
    expect(messages).toHaveTextContent("Sample note 'Agenda' moved to the recycle bin");
    expect(within(messages).getByRole('button', { name: 'Undo' })).toBeInTheDocument();
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
    expect(api.requests).toContainEqual({
      method: 'DELETE',
      path: '/api/v1/sample-notes/SN-1003',
      search: '',
      body: undefined,
    });
  });
});
