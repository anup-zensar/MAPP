import { screen, within } from '@testing-library/react';
import type { UserEvent } from '@testing-library/user-event';
import type { HttpHandler } from 'msw';
import type { components } from '@/api/generated/schema';
import { SampleRecordsPage } from '@/features/dev';
import type { CurrentUser } from '@/session/useSession';
import { closestTo } from '../dom';
import { mockApi } from '../mock-api';
import { renderWithProviders } from '../render';
import { reply } from '../replies';
import type { Answer } from '../replies';
import { testUser } from './users';

type SampleFolder = components['schemas']['SampleFolder'];
type SampleNote = components['schemas']['SampleNote'];
type RecycleBinEntry = components['schemas']['RecycleBinEntry'];
type ActivityLogEntry = components['schemas']['ActivityLogEntry'];

export const PRIYA: CurrentUser = testUser();

export const PRIYA_REF = { id: 'U-1002', name: 'Priya Raman' };
export const DANIEL_REF = { id: 'U-1003', name: 'Daniel Okoro' };

/** Created by Priya on 12 Mar 2026 at 09:00 and not changed since. */
export const MADE_BY_PRIYA = {
  createdAt: '2026-03-12T09:00:00Z',
  createdBy: PRIYA_REF,
  updatedAt: '2026-03-12T09:00:00Z',
  updatedBy: PRIYA_REF,
};

export const AGENDA: SampleNote = {
  id: 'SN-1003',
  folderId: 'SF-1001',
  title: 'Agenda',
  version: 1,
  ...MADE_BY_PRIYA,
};

export const MINUTES: SampleNote = {
  id: 'SN-1004',
  folderId: 'SF-1001',
  title: 'Minutes',
  version: 1,
  ...MADE_BY_PRIYA,
};

export const IDEAS: SampleFolder = {
  id: 'SF-1001',
  name: 'Ideas',
  notes: [AGENDA, MINUTES],
  version: 1,
  ...MADE_BY_PRIYA,
};

/** A folder without notes that Daniel renamed at 14:32. */
export const PLANS: SampleFolder = {
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
export const IDEAS_RENAMED_BY_DANIEL: SampleFolder = {
  ...IDEAS,
  name: 'Ideas 2',
  version: 2,
  updatedAt: '2026-03-12T14:32:00Z',
  updatedBy: DANIEL_REF,
};

export const IDEAS_IN_BIN: RecycleBinEntry = {
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
export const AGENDA_IN_BIN: RecycleBinEntry = {
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
export const LOGGED = { comment: '', details: '', path: [], changes: [] };

export const IDEAS_CREATED: ActivityLogEntry = {
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

export const IDEAS_RENAMED: ActivityLogEntry = {
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

export const AGENDA_CREATED: ActivityLogEntry = {
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

export const UNREACHABLE =
  "SP&A One can't reach its server right now. Check that it is running and try again.";

/** A list as the API answers it. */
export function listOf<T>(items: T[]) {
  return { items, total: items.length };
}

export interface PageAnswers {
  folders?: Answer<'GET /api/v1/sample-folders'>;
  bin?: Answer<'GET /api/v1/recycle-bin'>;
  log?: Answer<'GET /api/v1/activity-log'>;
}

/**
 * Answers the page's three lists — by default the folders "Ideas" (two notes) and "Plans", an
 * empty recycle bin and an empty activity log — and any other requests given.
 */
export function mockSampleRecords(
  {
    folders = { body: listOf([IDEAS, PLANS]) },
    bin = { body: listOf([]) },
    log = { body: listOf([]) },
  }: PageAnswers = {},
  otherAnswers: HttpHandler[] = [],
) {
  return mockApi([
    reply('GET /api/v1/sample-folders', folders),
    reply('GET /api/v1/recycle-bin', bin),
    reply('GET /api/v1/activity-log', log),
    ...otherAnswers,
  ]);
}

/** The page at its own address, for Priya, before its lists have arrived. */
export function renderSampleRecordsPageLoading() {
  return renderWithProviders(<SampleRecordsPage />, {
    user: PRIYA,
    route: '/dev/sample-records',
    path: '/dev/sample-records',
  });
}

/** The page at its own address, for Priya, once its lists have arrived. */
export async function renderSampleRecordsPage() {
  const rendered = renderSampleRecordsPageLoading();
  await screen.findByRole('region', { name: 'Sample folders' });
  return rendered;
}

/** Chooses an action in a folder's "⋯" menu. */
export async function chooseAction(user: UserEvent, folderName: string, action: string) {
  await user.click(screen.getByRole('button', { name: `Actions for ${folderName}` }));
  await user.click(await screen.findByRole('menuitem', { name: action }));
}

/** Opens the edit panel of a folder and types a new name in it. */
export async function renameIn(user: UserEvent, folderName: string, newName: string) {
  await chooseAction(user, folderName, 'Edit');
  const panel = await screen.findByRole('dialog', { name: 'Edit sample folder' });
  await user.clear(within(panel).getByLabelText('Name'));
  await user.type(within(panel).getByLabelText('Name'), newName);
  return panel;
}

/** The value of a summary card, found by its label. */
export function summaryValue(label: string) {
  return screen.getByText(label, { selector: 'p' }).nextElementSibling;
}

/** The list item that holds the given text, in the given card. */
export function itemWith(card: HTMLElement, text: string) {
  return closestTo(within(card).getByText(text), 'li');
}
