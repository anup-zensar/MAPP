import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { formatDateTime } from '@/lib/format';
import { mockApi } from '@/test/mockApi';
import type { MockHandler } from '@/test/mockApi';
import { renderWithProviders } from '@/test/render';
import { testUser } from '@/test/users';
import type { ActivityLogEntry, HomeScope } from './homeApi';
import { RecentChangesCard } from './RecentChangesCard';
import { LUBRICANTS_PLAN, logEntryOf } from './testData';

const page = (items: ActivityLogEntry[]) => ({ body: { items, total: items.length } });

const secondsAgo = (seconds: number) => new Date(Date.now() - seconds * 1000).toISOString();

function renderCard(scope: HomeScope = 'myTeams', answer: MockHandler = page([])) {
  const api = mockApi({ 'GET /api/v1/home/recent-changes': answer });
  const rendered = renderWithProviders(<RecentChangesCard scope={scope} />, {
    user: testUser(),
    path: '/',
  });
  return { ...rendered, api };
}

function card() {
  return screen.getByRole('region', { name: 'Recent changes' });
}

function entries() {
  return within(card()).getAllByRole('listitem');
}

/** The sentence of an entry: its first paragraph. */
function sentenceOf(entry: HTMLElement) {
  return entry.querySelector('p')?.textContent;
}

describe('RecentChangesCard', () => {
  it('says who changed which field of which record, and when (US4 scenario 3, FR-015)', async () => {
    const at = secondsAgo(20);
    renderCard('myTeams', page([logEntryOf({ at })]));

    await within(card()).findByRole('link', { name: 'Launch film' });
    const [entry] = entries();
    expect(sentenceOf(entry)).toBe(
      "Priya Raman changed planned cost of activity 'Launch film' (A-1042)",
    );
    expect(within(entry).getByRole('link', { name: 'Launch film' })).toHaveAttribute(
      'href',
      '/plans/P-1001/activities/A-1042',
    );
    expect(
      within(entry).getByRole('button', { name: `just now (${formatDateTime(at)})` }),
    ).toBeInTheDocument();
  });

  it('shows the full date and time of an entry on keyboard focus (FR-015)', async () => {
    const at = secondsAgo(20);
    const { user } = renderCard('myTeams', page([logEntryOf({ at })]));
    const time = await within(card()).findByRole('button', { name: /^just now/ });

    await user.tab(); // View activity log
    await user.tab(); // the record's name
    await user.tab();

    expect(time).toHaveFocus();
    expect(await screen.findByRole('tooltip')).toHaveTextContent(formatDateTime(at));
  });

  it('names other actions with their verb, and links each record to its page (FR-015)', async () => {
    renderCard(
      'everything',
      page([
        logEntryOf({
          id: 'LOG-1002',
          action: 'statusChanged',
          verb: 'published',
          recordType: 'plan',
          recordTypeLabel: 'plan',
          recordId: 'P-1001',
          recordName: 'FY2026 Lubricants Europe',
          path: [],
          changes: [],
          actor: { id: 'U-1003', name: 'Daniel Okoro' },
          recordNow: {
            state: 'exists',
            name: 'FY2026 Lubricants Europe',
            path: [],
            binEntryId: '',
          },
        }),
        logEntryOf({
          id: 'LOG-1003',
          action: 'created',
          verb: 'created',
          recordType: 'campaign',
          recordTypeLabel: 'campaign',
          recordId: 'C-1001',
          recordName: 'Helix launch',
          changes: [],
          recordNow: {
            state: 'exists',
            name: 'Helix launch',
            path: [LUBRICANTS_PLAN],
            binEntryId: '',
          },
        }),
      ]),
    );

    await within(card()).findByRole('link', { name: 'Helix launch' });
    expect(entries().map(sentenceOf)).toEqual([
      "Daniel Okoro published plan 'FY2026 Lubricants Europe' (P-1001)",
      "Priya Raman created campaign 'Helix launch' (C-1001)",
    ]);
    expect(within(card()).getByRole('link', { name: 'FY2026 Lubricants Europe' })).toHaveAttribute(
      'href',
      '/plans/P-1001',
    );
    expect(within(card()).getByRole('link', { name: 'Helix launch' })).toHaveAttribute(
      'href',
      '/plans/P-1001/campaigns/C-1001',
    );
  });

  it('names a deleted record without a link, saying where it is (FR-015, edge case)', async () => {
    renderCard(
      'myTeams',
      page([
        logEntryOf({
          id: 'LOG-1004',
          action: 'deleted',
          verb: 'deleted',
          recordId: 'A-1043',
          recordName: 'Trade show',
          changes: [],
          recordNow: { state: 'inRecycleBin', name: 'Trade show', path: [], binEntryId: 'BIN-1' },
        }),
        logEntryOf({
          id: 'LOG-1005',
          recordId: 'A-1044',
          recordName: 'Spring radio',
          recordNow: { state: 'deletedPermanently', name: '', path: [], binEntryId: '' },
        }),
      ]),
    );

    await within(card()).findAllByRole('listitem');
    expect(entries().map(sentenceOf)).toEqual([
      "Priya Raman deleted activity 'Trade show' (A-1043) (in recycle bin)",
      "Priya Raman changed planned cost of activity 'Spring radio' (A-1044) (permanently deleted)",
    ]);
    expect(within(card()).queryByRole('link', { name: 'Trade show' })).not.toBeInTheDocument();
    expect(within(card()).queryByRole('link', { name: 'Spring radio' })).not.toBeInTheDocument();
  });

  it('names several changed fields in one sentence (FR-015)', async () => {
    const change = (label: string) => ({
      field: label,
      label,
      old: null,
      new: null,
      oldText: '',
      newText: '',
    });
    renderCard(
      'myTeams',
      page([
        logEntryOf({ changes: [change('Name'), change('Planned cost'), change('GL code')] }),
        logEntryOf({
          id: 'LOG-1006',
          changes: ['Name', 'Region', 'Start date', 'End date', 'Owner'].map(change),
        }),
      ]),
    );

    await within(card()).findAllByRole('link', { name: 'Launch film' });
    expect(entries().map(sentenceOf)).toEqual([
      "Priya Raman changed name, planned cost and GL code of activity 'Launch film' (A-1042)",
      "Priya Raman changed name, region, start date and 2 more fields of activity 'Launch film' (A-1042)",
    ]);
  });

  it('opens the activity log and says which plans it covers (FR-015, FR-016)', async () => {
    const { api } = renderCard('myTeams');

    expect(within(card()).getByRole('link', { name: 'View activity log' })).toHaveAttribute(
      'href',
      '/activity-log',
    );
    expect(card()).toHaveTextContent("Latest changes in your teams' plans");
    expect(await within(card()).findByText('No changes yet in these plans.')).toBeInTheDocument();
    expect(api.requests.map((request) => request.search)).toEqual(['?scope=myTeams']);
  });

  it('covers all plans under Everything (FR-003)', async () => {
    const { api } = renderCard('everything');

    expect(card()).toHaveTextContent('Latest changes in all plans');
    await within(card()).findByText('No changes yet in these plans.');
    expect(api.requests.map((request) => request.search)).toEqual(['?scope=everything']);
  });

  it('shows a placeholder while loading, and Try again when it fails (FR-022)', async () => {
    let calls = 0;
    const { user } = renderCard('myTeams', () =>
      ++calls === 1 ? { status: 500 } : page([logEntryOf()]),
    );

    expect(within(card()).getByRole('status')).toHaveTextContent('Loading…');
    expect(await within(card()).findByRole('alert')).toHaveTextContent(
      "This section couldn't load",
    );
    await user.click(within(card()).getByRole('button', { name: 'Try again' }));

    expect(await within(card()).findByRole('link', { name: 'Launch film' })).toBeInTheDocument();
  });
});
