import { act, screen, waitFor, within } from '@testing-library/react';
import type { UserEvent } from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';
import type { components } from '@/api/generated/schema';
import type { CurrentUser } from '@/app/useSession';
import { mockApi } from '@/test/mockApi';
import type { MockHandler } from '@/test/mockApi';
import { findMessages, renderWithProviders } from '@/test/render';
import { SampleRecordsPage } from './SampleRecordsPage';
import { testUser } from '@/test/users';

type SampleFolder = components['schemas']['SampleFolder'];
type SampleNote = components['schemas']['SampleNote'];
type RecycleBinEntry = components['schemas']['RecycleBinEntry'];
type ActivityLogEntry = components['schemas']['ActivityLogEntry'];

const PRIYA: CurrentUser = testUser();

const PRIYA_REF = { id: 'U-1002', name: 'Priya Raman' };
const DANIEL_REF = { id: 'U-1003', name: 'Daniel Okoro' };

/** Created by Priya on 12 Mar 2026 at 09:00 and not changed since. */
const MADE_BY_PRIYA = {
  createdAt: '2026-03-12T09:00:00Z',
  createdBy: PRIYA_REF,
  updatedAt: '2026-03-12T09:00:00Z',
  updatedBy: PRIYA_REF,
};

const AGENDA: SampleNote = {
  id: 'SN-1003',
  folderId: 'SF-1001',
  title: 'Agenda',
  version: 1,
  ...MADE_BY_PRIYA,
};

const MINUTES: SampleNote = {
  id: 'SN-1004',
  folderId: 'SF-1001',
  title: 'Minutes',
  version: 1,
  ...MADE_BY_PRIYA,
};

const IDEAS: SampleFolder = {
  id: 'SF-1001',
  name: 'Ideas',
  notes: [AGENDA, MINUTES],
  version: 1,
  ...MADE_BY_PRIYA,
};

/** A folder without notes that Daniel renamed at 14:32. */
const PLANS: SampleFolder = {
  id: 'SF-1002',
  name: 'Plans',
  notes: [],
  version: 3,
  createdAt: '2026-03-12T09:00:00Z',
  createdBy: PRIYA_REF,
  updatedAt: '2026-03-12T14:32:00Z',
  updatedBy: DANIEL_REF,
};

/** "Ideas" after Daniel renamed it at 14:32, while Priya still had the first version open. */
const IDEAS_RENAMED_BY_DANIEL: SampleFolder = {
  ...IDEAS,
  name: 'Ideas 2',
  version: 2,
  updatedAt: '2026-03-12T14:32:00Z',
  updatedBy: DANIEL_REF,
};

const IDEAS_IN_BIN: RecycleBinEntry = {
  id: 'BIN-1001',
  recordType: 'sampleFolder',
  recordTypeLabel: 'sample folder',
  recordId: 'SF-1001',
  recordName: 'Ideas',
  path: [],
  parent: null,
  deletedAt: '2026-03-12T09:30:00Z',
  deletedBy: PRIYA_REF,
  contents: [{ recordType: 'sampleNote', label: 'sample notes', count: 2 }],
};

/** A note whose folder "Ideas" was deleted after it. */
const AGENDA_IN_BIN: RecycleBinEntry = {
  id: 'BIN-1002',
  recordType: 'sampleNote',
  recordTypeLabel: 'sample note',
  recordId: 'SN-1003',
  recordName: 'Agenda',
  path: [{ type: 'sampleFolder', id: 'SF-1001', name: 'Ideas' }],
  parent: { type: 'sampleFolder', id: 'SF-1001', name: 'Ideas' },
  deletedAt: '2026-03-12T09:20:00Z',
  deletedBy: DANIEL_REF,
  contents: [],
};

/** Fields every activity-log entry of these tests shares. */
const LOGGED = { comment: '', details: '', path: [], changes: [] };

const IDEAS_CREATED: ActivityLogEntry = {
  ...LOGGED,
  id: 'LOG-1001',
  seq: 1001,
  at: '2026-03-12T09:00:00Z',
  actor: PRIYA_REF,
  action: 'created',
  verb: 'created',
  recordType: 'sampleFolder',
  recordTypeLabel: 'sample folder',
  recordId: 'SF-1001',
  recordName: 'Ideas',
};

const IDEAS_RENAMED: ActivityLogEntry = {
  ...IDEAS_CREATED,
  id: 'LOG-1002',
  seq: 1002,
  at: '2026-03-12T14:32:00Z',
  actor: DANIEL_REF,
  action: 'changed',
  verb: 'changed',
  recordName: 'Ideas 2',
  changes: [
    {
      field: 'name',
      label: 'Name',
      old: 'Ideas',
      new: 'Ideas 2',
      oldText: 'Ideas',
      newText: 'Ideas 2',
    },
  ],
};

const AGENDA_CREATED: ActivityLogEntry = {
  ...LOGGED,
  id: 'LOG-1003',
  seq: 1003,
  at: '2026-03-12T14:40:00Z',
  actor: PRIYA_REF,
  action: 'created',
  verb: 'created',
  recordType: 'sampleNote',
  recordTypeLabel: 'sample note',
  recordId: 'SN-1003',
  recordName: 'Agenda',
  path: [{ type: 'sampleFolder', id: 'SF-1001', name: 'Ideas 2' }],
};

const UNREACHABLE =
  "MAPP can't reach its server right now. Check that it is running and try again.";

/** A list as the API answers it. */
function listOf<T>(items: T[]) {
  return { items, total: items.length };
}

interface PageAnswers {
  folders?: MockHandler;
  bin?: MockHandler;
  log?: MockHandler;
}

/**
 * Answers the page's three lists — by default the folders "Ideas" (two notes) and "Plans", an
 * empty recycle bin and an empty activity log — and any other requests given.
 */
function mockPage(
  {
    folders = { body: listOf([IDEAS, PLANS]) },
    bin = { body: listOf([]) },
    log = { body: listOf([]) },
  }: PageAnswers = {},
  otherAnswers: Record<string, MockHandler> = {},
) {
  return mockApi({
    'GET /api/v1/sample-folders': folders,
    'GET /api/v1/recycle-bin': bin,
    'GET /api/v1/activity-log': log,
    ...otherAnswers,
  });
}

/** The page at its own address, for Priya, before its lists have arrived. */
function renderPageLoading() {
  return renderWithProviders(<SampleRecordsPage />, {
    user: PRIYA,
    route: '/dev/sample-records',
    path: '/dev/sample-records',
  });
}

/** The page at its own address, for Priya, once its lists have arrived. */
async function renderPage() {
  const rendered = renderPageLoading();
  await screen.findByRole('region', { name: 'Sample folders' });
  return rendered;
}

/** Chooses an action in a folder's "⋯" menu. */
async function chooseAction(user: UserEvent, folderName: string, action: string) {
  await user.click(screen.getByRole('button', { name: `Actions for ${folderName}` }));
  await user.click(await screen.findByRole('menuitem', { name: action }));
}

/** Opens the edit panel of a folder and types a new name in it. */
async function renameIn(user: UserEvent, folderName: string, newName: string) {
  await chooseAction(user, folderName, 'Edit');
  const panel = await screen.findByRole('dialog', { name: 'Edit sample folder' });
  await user.clear(within(panel).getByLabelText('Name'));
  await user.type(within(panel).getByLabelText('Name'), newName);
  return panel;
}

/** The value of a summary card, found by its label. */
function summaryValue(label: string) {
  return screen.getByText(label, { selector: 'p' }).nextElementSibling;
}

/** The list item that holds the given text, in the given card. */
function itemWith(card: HTMLElement, text: string) {
  return within(card).getByText(text).closest('li') as HTMLElement;
}

afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
  localStorage.clear();
  delete document.documentElement.dataset.theme;
});

describe('SampleRecordsPage: the page', () => {
  it('says what it is for, offers New sample folder and names the tab (plan "Developer pages", FR-017)', async () => {
    mockPage();
    await renderPage();

    expect(screen.getByText('Developer')).toBeInTheDocument();
    expect(screen.getByRole('heading', { level: 1, name: 'Sample records' })).toBeInTheDocument();
    expect(
      screen.getByText(
        'Try the shared create, edit, delete, Undo and activity-log behaviour. Not part of the Plan module.',
      ),
    ).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'New sample folder' })).toBeInTheDocument();
    expect(document.title).toBe('Sample records · MAPP');
  });

  it('counts the folders, their notes and what is in the recycle bin (plan "Developer pages")', async () => {
    mockPage({ bin: { body: listOf([IDEAS_IN_BIN]) } });
    await renderPage();

    expect(summaryValue('Sample folders')).toHaveTextContent('2');
    expect(summaryValue('Sample notes')).toHaveTextContent('2');
    expect(summaryValue('In the recycle bin')).toHaveTextContent('1');
  });

  it('lists each folder with its ID, its notes and who changed it last, and when (plan "Developer pages")', async () => {
    mockPage();
    await renderPage();

    const ideas = screen.getByRole('row', { name: /^Ideas/ });
    expect(within(ideas).getByRole('rowheader')).toHaveTextContent('Ideas SF-1001');
    expect(within(ideas).getByText('Agenda')).toBeInTheDocument();
    expect(within(ideas).getByRole('button', { name: "Delete note 'Agenda'" })).toBeInTheDocument();
    expect(
      within(ideas).getByRole('button', { name: "Delete note 'Minutes'" }),
    ).toBeInTheDocument();
    expect(within(ideas).getByText('Priya Raman, 12 Mar 2026, 09:00')).toBeInTheDocument();
    expect(within(ideas).getByRole('button', { name: 'Actions for Ideas' })).toBeInTheDocument();

    const plans = screen.getByRole('row', { name: /^Plans/ });
    expect(within(plans).getByText('Daniel Okoro, 12 Mar 2026, 14:32')).toBeInTheDocument();
    expect(within(plans).queryByRole('button', { name: /^Delete note/ })).not.toBeInTheDocument();
  });

  it('offers Edit, Add note and, after a separator, Delete for each folder (design system §6.1)', async () => {
    mockPage();
    const { user } = await renderPage();

    await user.click(screen.getByRole('button', { name: 'Actions for Ideas' }));

    const items = (await screen.findAllByRole('menuitem')).map((item) => item.textContent);
    expect(items).toEqual(['Edit', 'Add note', 'Delete']);
  });

  it('invites to create the first sample folder when there are none (plan "User-visible messages")', async () => {
    mockPage({ folders: { body: listOf([]) } });
    const { user } = await renderPage();

    const card = screen.getByRole('region', { name: 'Sample folders' });
    expect(
      within(card).getByText('No sample folders yet — create your first sample folder'),
    ).toBeInTheDocument();
    await user.click(within(card).getByRole('button', { name: 'New sample folder' }));

    expect(await screen.findByRole('dialog', { name: 'New sample folder' })).toBeInTheDocument();
  });

  it('shows a loading placeholder until the lists arrive (design system §5)', async () => {
    mockPage({ folders: () => new Promise<never>(() => {}) });
    renderPageLoading();

    expect(await screen.findByRole('status')).toHaveTextContent('Loading…');
  });

  it('says when the API cannot be reached, and shows the page after Try again (FR-020)', async () => {
    let reads = 0;
    mockPage({
      folders: () =>
        ++reads === 1
          ? Promise.reject(new TypeError('Failed to fetch'))
          : { body: listOf([IDEAS, PLANS]) },
    });
    const { user } = renderPageLoading();

    expect(await screen.findByRole('alert')).toHaveTextContent(UNREACHABLE);
    await user.click(screen.getByRole('button', { name: 'Try again' }));

    expect(await screen.findByRole('row', { name: /^Ideas/ })).toBeInTheDocument();
  });

  it('leaves out Recently deleted and the activity log while they have nothing to show', async () => {
    mockPage();
    await renderPage();

    expect(screen.queryByRole('region', { name: 'Recently deleted' })).not.toBeInTheDocument();
    expect(screen.queryByRole('region', { name: 'Activity log' })).not.toBeInTheDocument();
  });
});

describe('SampleRecordsPage: create and edit', () => {
  it('creates a folder in a side panel and says so (US5, plan "User-visible messages")', async () => {
    const api = mockPage(
      {},
      { 'POST /api/v1/sample-folders': { status: 201, body: { ...PLANS, name: 'Ideas 3' } } },
    );
    const { user } = await renderPage();

    await user.click(screen.getByRole('button', { name: 'New sample folder' }));
    const panel = await screen.findByRole('dialog', { name: 'New sample folder' });
    expect(within(panel).getByText('Required')).toBeInTheDocument();
    await user.type(within(panel).getByLabelText('Name'), 'Ideas 3');
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(await findMessages()).toHaveTextContent("Sample folder 'Ideas 3' created");
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    expect(api.bodiesOf('POST /api/v1/sample-folders')).toEqual([{ name: 'Ideas 3' }]);
  });

  it('shows the API’s problem with a new name at the field and in the summary (FR-020)', async () => {
    mockPage(
      {},
      {
        'POST /api/v1/sample-folders': {
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
        },
      },
    );
    const { user } = await renderPage();

    await user.click(screen.getByRole('button', { name: 'New sample folder' }));
    const panel = await screen.findByRole('dialog', { name: 'New sample folder' });
    await user.type(within(panel).getByLabelText('Name'), 'ideas');
    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    const summary = await within(panel).findByRole('alert');
    expect(summary).toHaveTextContent('1 problem needs your attention');
    expect(within(panel).getByLabelText('Name')).toHaveAccessibleDescription(
      "Required There is already a sample folder called 'Ideas'. Choose another name.",
    );
    expect(within(panel).getByLabelText('Name')).toHaveValue('ideas');
  });

  it('opens Edit from the actions menu with the folder’s name, and saves the new name (US5)', async () => {
    const api = mockPage(
      {},
      {
        'PATCH /api/v1/sample-folders/SF-1001': { body: { ...IDEAS, name: 'Ideas 2', version: 2 } },
      },
    );
    const { user } = await renderPage();

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
    mockPage(
      {},
      {
        'PATCH /api/v1/sample-folders/SF-1001': {
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
      },
    );
    const { user } = await renderPage();
    await chooseAction(user, 'Ideas', 'Edit');
    const panel = await screen.findByRole('dialog', { name: 'Edit sample folder' });
    await user.clear(within(panel).getByLabelText('Name'));

    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    const summary = await within(panel).findByRole('alert');
    expect(summary).toHaveTextContent('2 problems need your attention');
    expect(summary).toHaveTextContent('Enter a name.');
    expect(summary).toHaveTextContent('Reload the page and try again.');
    expect(within(panel).getByLabelText('Name')).toHaveAccessibleDescription(
      'Required Enter a name.',
    );
  });

  it('saves with the version the form started from, even after the list was refreshed (FR-029, plan "Shared hooks")', async () => {
    let folderReads = 0;
    const api = mockPage(
      {
        folders: () => ({
          body: listOf([
            ++folderReads === 1 ? IDEAS : { ...IDEAS_RENAMED_BY_DANIEL, name: 'Ideas (Daniel)' },
          ]),
        }),
      },
      {
        'PATCH /api/v1/sample-folders/SF-1001': { body: { ...IDEAS, name: 'Ideas 2', version: 3 } },
      },
    );
    const { user, queryClient } = await renderPage();
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
    const api = mockPage(
      {
        folders: () => ({ body: listOf([++folderReads === 1 ? IDEAS : IDEAS_RENAMED_BY_DANIEL]) }),
      },
      {
        'PATCH /api/v1/sample-folders/SF-1001': () =>
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
      },
    );
    const { user } = await renderPage();
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
    mockPage(
      {},
      {
        'PATCH /api/v1/sample-folders/SF-1001': {
          status: 404,
          body: {
            error: {
              code: 'not_found',
              message: 'This sample folder no longer exists — someone may have deleted it.',
              fields: [],
            },
          },
        },
      },
    );
    const { user } = await renderPage();
    const panel = await renameIn(user, 'Ideas', 'Ideas 2');

    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(await within(panel).findByRole('alert')).toHaveTextContent(
      'This sample folder no longer exists — someone may have deleted it.',
    );
  });

  it('asks before closing a panel with typed changes (design system §6.2)', async () => {
    mockPage();
    const { user } = await renderPage();
    await user.click(screen.getByRole('button', { name: 'New sample folder' }));
    const panel = await screen.findByRole('dialog', { name: 'New sample folder' });
    await user.type(within(panel).getByLabelText('Name'), 'Ideas 3');

    await user.click(within(panel).getByRole('button', { name: 'Cancel' }));
    await user.click(await screen.findByRole('button', { name: 'Discard changes' }));

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  it('asks "Discard changes?" when going Back with typed changes (design system §6.4)', async () => {
    mockPage();
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

describe('SampleRecordsPage: notes', () => {
  it('adds a note to a folder from the actions menu and says so (US5)', async () => {
    const api = mockPage(
      {},
      {
        'POST /api/v1/sample-folders/SF-1001/notes': {
          status: 201,
          body: { ...AGENDA, id: 'SN-1005', title: 'Budget' },
        },
      },
    );
    const { user } = await renderPage();

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
    mockPage(
      {},
      {
        'POST /api/v1/sample-folders/SF-1001/notes': {
          status: 400,
          body: {
            error: {
              code: 'validation_failed',
              message: 'Some fields need your attention.',
              fields: [{ field: 'title', message: 'Enter a title.' }],
            },
          },
        },
      },
    );
    const { user } = await renderPage();
    await chooseAction(user, 'Ideas', 'Add note');
    const panel = await screen.findByRole('dialog', { name: 'Add note' });

    await user.click(within(panel).getByRole('button', { name: 'Save' }));

    expect(await within(panel).findByRole('alert')).toHaveTextContent('Enter a title.');
    expect(within(panel).getByLabelText('Title')).toHaveAccessibleDescription(
      'Required Enter a title.',
    );
  });

  it('deletes a note after asking, and offers Undo (FR-024, FR-026)', async () => {
    const api = mockPage(
      {},
      { 'DELETE /api/v1/sample-notes/SN-1003': { body: { recycleBinEntry: AGENDA_IN_BIN } } },
    );
    const { user } = await renderPage();

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

describe('SampleRecordsPage: delete, Undo and restore', () => {
  it('deletes a folder with its notes after asking, and offers Undo (FR-024, FR-026, US5)', async () => {
    const api = mockPage(
      {},
      { 'DELETE /api/v1/sample-folders/SF-1001': { body: { recycleBinEntry: IDEAS_IN_BIN } } },
    );
    const { user } = await renderPage();

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
    expect(messages).toHaveTextContent("Sample folder 'Ideas' moved to the recycle bin");
    expect(within(messages).getByRole('button', { name: 'Undo' })).toBeInTheDocument();
    await waitFor(() =>
      expect(screen.getByRole('status')).toHaveTextContent(
        "Sample folder 'Ideas' moved to the recycle bin Restore it from Recently deleted on this page",
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
      mockPage({ folders: { body: listOf([folder]) } });
      const { user } = await renderPage();

      await chooseAction(user, 'Ideas', 'Delete');

      expect(
        await screen.findByRole('alertdialog', { name: "Delete sample folder 'Ideas'?" }),
      ).toHaveAccessibleDescription(text);
    },
  );

  it('keeps the folder when the delete is cancelled (design system §6.5)', async () => {
    const api = mockPage();
    const { user } = await renderPage();
    await chooseAction(user, 'Ideas', 'Delete');

    await user.click(await screen.findByRole('button', { name: 'Cancel' }));

    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
    expect(api.requests.filter((request) => request.method === 'DELETE')).toEqual([]);
  });

  it('says when a delete cannot reach the server, and tries again (FR-020)', async () => {
    let attempts = 0;
    mockPage(
      {},
      {
        'DELETE /api/v1/sample-folders/SF-1001': () =>
          ++attempts === 1
            ? Promise.reject(new TypeError('Failed to fetch'))
            : { body: { recycleBinEntry: IDEAS_IN_BIN } },
      },
    );
    const { user } = await renderPage();
    await chooseAction(user, 'Ideas', 'Delete');
    await user.click(await screen.findByRole('button', { name: 'Delete sample folder' }));

    const messages = await findMessages();
    await waitFor(() => expect(messages).toHaveTextContent(UNREACHABLE));
    await user.click(within(messages).getByRole('button', { name: 'Try again' }));

    await waitFor(() =>
      expect(messages).toHaveTextContent("Sample folder 'Ideas' moved to the recycle bin"),
    );
  });

  it('lists what was deleted, with what went with it, who deleted it and when (plan "Developer pages")', async () => {
    mockPage({ bin: { body: listOf([IDEAS_IN_BIN, AGENDA_IN_BIN]) } });
    await renderPage();

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

  it('restores from Recently deleted and says what the restore changed (FR-025)', async () => {
    const api = mockPage(
      { bin: { body: listOf([IDEAS_IN_BIN]) } },
      {
        'POST /api/v1/recycle-bin/BIN-1001/restore': {
          body: {
            recordType: 'sampleFolder',
            recordTypeLabel: 'sample folder',
            recordId: 'SF-1001',
            recordName: 'Ideas (restored)',
            details: "Renamed to 'Ideas (restored)' because the name was taken.",
          },
        },
      },
    );
    const { user } = await renderPage();

    await user.click(screen.getByRole('button', { name: 'Restore sample folder Ideas' }));

    expect(await findMessages()).toHaveTextContent(
      "Sample folder 'Ideas' restored. Renamed to 'Ideas (restored)' because the name was taken.",
    );
    expect(api.bodiesOf('POST /api/v1/recycle-bin/BIN-1001/restore')).toHaveLength(1);
  });

  it('says why a restore was refused (FR-025, US5 scenario 4)', async () => {
    mockPage(
      { bin: { body: listOf([AGENDA_IN_BIN]) } },
      {
        'POST /api/v1/recycle-bin/BIN-1002/restore': {
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
      },
    );
    const { user } = await renderPage();

    await user.click(screen.getByRole('button', { name: 'Restore sample note Agenda' }));

    const messages = await findMessages();
    expect(messages).toHaveTextContent("Restore 'Ideas' first");
    expect(within(messages).queryByRole('button', { name: 'Try again' })).not.toBeInTheDocument();
  });
});

describe('SampleRecordsPage: activity log', () => {
  it('shows the latest 20 entries: who did what to which record, where, when and what changed (FR-022)', async () => {
    const api = mockPage({ log: { body: listOf([AGENDA_CREATED, IDEAS_RENAMED, IDEAS_CREATED]) } });
    await renderPage();

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
