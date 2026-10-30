import { act, screen, waitFor, within } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { findMessages, renderWithProviders } from '@/testing/render';
import {
  chooseAction,
  IDEAS,
  IDEAS_RENAMED_BY_DANIEL,
  listOf,
  mockSampleRecords,
  PLANS,
  PRIYA,
  renameIn,
  renderSampleRecordsPage,
} from '@/testing/fixtures/dev';
import { reply } from '@/testing/replies';
import { SampleRecordsPage } from './SampleRecordsPage';

afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
  localStorage.clear();
  delete document.documentElement.dataset.theme;
});

describe('SampleRecordsPage: create and edit', () => {
  it('creates a folder in a side panel and says so (US5, plan "User-visible messages")', async () => {
    const api = mockSampleRecords({}, [
      reply('POST /api/v1/sample-folders', {
        status: 201,
        body: { ...PLANS, name: 'Ideas 3' },
      }),
    ]);
    const { user } = await renderSampleRecordsPage();

    await user.click(screen.getByRole('button', { name: 'New sample folder' }));
    const panel = await screen.findByRole('dialog', { name: 'New sample folder' });
    expect(within(panel).getByLabelText('Name')).toHaveAttribute('aria-required', 'true');
    await user.type(within(panel).getByLabelText('Name'), 'Ideas 3');
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(await findMessages()).toHaveTextContent("Sample folder 'Ideas 3' created");
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    expect(api.bodiesOf('POST /api/v1/sample-folders')).toEqual([{ name: 'Ideas 3' }]);
  });

  it('shows the API’s problem with a new name at the field and in the summary (FR-020)', async () => {
    mockSampleRecords({}, [
      reply('POST /api/v1/sample-folders', {
        status: 400,
        body: {
          error: {
            code: 'validation_failed',
            message: 'Some fields need your attention.',
            fields: [
              {
                field: 'name',
                message: "There is already a sample folder called 'Ideas'. Choose another name.",
              },
            ],
          },
        },
      }),
    ]);
    const { user } = await renderSampleRecordsPage();

    await user.click(screen.getByRole('button', { name: 'New sample folder' }));
    const panel = await screen.findByRole('dialog', { name: 'New sample folder' });
    await user.type(within(panel).getByLabelText('Name'), 'ideas');
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    const summary = await within(panel).findByRole('alert');
    expect(summary).toHaveTextContent('1 problem needs your attention');
    expect(within(panel).getByLabelText('Name')).toHaveAccessibleDescription(
      "required There is already a sample folder called 'Ideas'. Choose another name.",
    );
    expect(within(panel).getByLabelText('Name')).toHaveValue('ideas');
  });

  it('opens Edit from the actions menu with the folder’s name, and saves the new name (US5)', async () => {
    const api = mockSampleRecords({}, [
      reply(
        'PATCH /api/v1/sample-folders/{folderId}',
        { body: { ...IDEAS, name: 'Ideas 2', version: 2 } },
        { folderId: 'SF-1001' },
      ),
    ]);
    const { user } = await renderSampleRecordsPage();

    await chooseAction(user, 'Ideas', 'Edit');
    const panel = await screen.findByRole('dialog', { name: 'Edit sample folder' });
    expect(within(panel).getByLabelText('Name')).toHaveValue('Ideas');
    await user.clear(within(panel).getByLabelText('Name'));
    await user.type(within(panel).getByLabelText('Name'), 'Ideas 2');
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(await findMessages()).toHaveTextContent("Sample folder 'Ideas 2' saved");
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    expect(api.bodiesOf('PATCH /api/v1/sample-folders/SF-1001')).toEqual([
      { name: 'Ideas 2', version: 1 },
    ]);
  });

  it('shows every problem at once, at the field and in the summary (FR-020, design system §6.4)', async () => {
    mockSampleRecords({}, [
      reply(
        'PATCH /api/v1/sample-folders/{folderId}',
        {
          status: 400,
          body: {
            error: {
              code: 'validation_failed',
              message: 'Some fields need your attention.',
              fields: [
                { field: 'name', message: 'Enter a name.' },
                { field: 'version', message: 'Reload the page and try again.' },
              ],
            },
          },
        },
        { folderId: 'SF-1001' },
      ),
    ]);
    const { user } = await renderSampleRecordsPage();
    await chooseAction(user, 'Ideas', 'Edit');
    const panel = await screen.findByRole('dialog', { name: 'Edit sample folder' });
    await user.clear(within(panel).getByLabelText('Name'));

    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    const summary = await within(panel).findByRole('alert');
    expect(summary).toHaveTextContent('2 problems need your attention');
    expect(summary).toHaveTextContent('Enter a name.');
    expect(summary).toHaveTextContent('Reload the page and try again.');
    expect(within(panel).getByLabelText('Name')).toHaveAccessibleDescription(
      'required Enter a name.',
    );
  });

  it('saves with the version the form started from, even after the list was refreshed (FR-029, plan "Shared hooks")', async () => {
    let folderReads = 0;
    const api = mockSampleRecords(
      {
        folders: () => ({
          body: listOf([
            ++folderReads === 1 ? IDEAS : { ...IDEAS_RENAMED_BY_DANIEL, name: 'Ideas (Daniel)' },
          ]),
        }),
      },
      [
        reply(
          'PATCH /api/v1/sample-folders/{folderId}',
          { body: { ...IDEAS, name: 'Ideas 2', version: 3 } },
          { folderId: 'SF-1001' },
        ),
      ],
    );
    const { user, queryClient } = await renderSampleRecordsPage();
    await chooseAction(user, 'Ideas', 'Edit');
    const panel = await screen.findByRole('dialog', { name: 'Edit sample folder' });

    // Meanwhile the list is refreshed and brings Daniel's change.
    await act(() => queryClient.invalidateQueries({ queryKey: ['sample-folders'] }));
    await waitFor(() => expect(screen.getByText('Ideas (Daniel)')).toBeInTheDocument());
    expect(within(panel).getByLabelText('Name')).toHaveValue('Ideas');
    await user.clear(within(panel).getByLabelText('Name'));
    await user.type(within(panel).getByLabelText('Name'), 'Ideas 2');
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    await waitFor(() =>
      expect(api.bodiesOf('PATCH /api/v1/sample-folders/SF-1001')).toEqual([
        { name: 'Ideas 2', version: 1 },
      ]),
    );
  });

  it('refuses a save from an out-of-date copy, keeps the typed name, and the next Save replaces their change (FR-029)', async () => {
    vi.setSystemTime(new Date('2026-03-12T15:00:00Z'));
    let folderReads = 0;
    let saves = 0;
    const api = mockSampleRecords(
      {
        folders: () => ({ body: listOf([++folderReads === 1 ? IDEAS : IDEAS_RENAMED_BY_DANIEL]) }),
      },
      [
        reply(
          'PATCH /api/v1/sample-folders/{folderId}',
          () =>
            ++saves === 1
              ? {
                  status: 409,
                  body: {
                    error: {
                      code: 'version_conflict',
                      message: 'Not saved — Daniel Okoro changed this sample folder',
                      fields: [],
                      conflict: { changedBy: 'Daniel Okoro', changedAt: '2026-03-12T14:32:00Z' },
                    },
                  },
                }
              : { body: { ...IDEAS_RENAMED_BY_DANIEL, name: 'My ideas', version: 3 } },
          { folderId: 'SF-1001' },
        ),
      ],
    );
    const { user } = await renderSampleRecordsPage();
    const panel = await renameIn(user, 'Ideas', 'My ideas');

    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    const message = await within(panel).findByRole('alert');
    expect(message).toHaveTextContent(
      'Not saved — Daniel Okoro changed this sample folder at 14:32',
    );
    expect(message).toHaveTextContent(
      'Your changes are still here. Save again to replace their changes, or Cancel to see theirs.',
    );
    expect(within(panel).getByLabelText('Name')).toHaveValue('My ideas');

    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(await findMessages()).toHaveTextContent("Sample folder 'My ideas' saved");
    expect(api.bodiesOf('PATCH /api/v1/sample-folders/SF-1001')).toEqual([
      { name: 'My ideas', version: 1 },
      { name: 'My ideas', version: 2 },
    ]);
  });

  it('says when the folder no longer exists (plan "User-visible messages")', async () => {
    mockSampleRecords({}, [
      reply(
        'PATCH /api/v1/sample-folders/{folderId}',
        {
          status: 404,
          body: {
            error: {
              code: 'not_found',
              message: 'This sample folder no longer exists — someone may have deleted it.',
              fields: [],
            },
          },
        },
        { folderId: 'SF-1001' },
      ),
    ]);
    const { user } = await renderSampleRecordsPage();
    const panel = await renameIn(user, 'Ideas', 'Ideas 2');

    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(await within(panel).findByRole('alert')).toHaveTextContent(
      'This sample folder no longer exists — someone may have deleted it.',
    );
  });

  it('asks before closing a panel with typed changes (design system §6.2)', async () => {
    mockSampleRecords();
    const { user } = await renderSampleRecordsPage();
    await user.click(screen.getByRole('button', { name: 'New sample folder' }));
    const panel = await screen.findByRole('dialog', { name: 'New sample folder' });
    await user.type(within(panel).getByLabelText('Name'), 'Ideas 3');

    await user.click(within(panel).getByRole('button', { name: 'Cancel' }));
    await user.click(await screen.findByRole('button', { name: 'Discard changes' }));

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  it('asks "Discard changes?" when going Back with typed changes (design system §6.4)', async () => {
    mockSampleRecords();
    const { user, router } = renderWithProviders(<SampleRecordsPage />, {
      user: PRIYA,
      route: '/',
      path: '/dev/sample-records',
    });
    await act(() => router.navigate('/dev/sample-records'));
    await user.click(await screen.findByRole('button', { name: 'New sample folder' }));
    const panel = await screen.findByRole('dialog', { name: 'New sample folder' });
    await user.type(within(panel).getByLabelText('Name'), 'Ideas 3');

    await act(() => router.navigate(-1));

    const question = screen.getByRole('alertdialog', { name: 'Discard changes?' });
    await user.click(within(question).getByRole('button', { name: 'Keep editing' }));
    expect(router.state.location.pathname).toBe('/dev/sample-records');
    expect(within(panel).getByLabelText('Name')).toHaveValue('Ideas 3');
  });
});
