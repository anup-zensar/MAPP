import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { mockApi } from '@/test/mockApi';
import type { MockHandler } from '@/test/mockApi';
import { findMessages, renderWithProviders } from '@/test/render';
import { testUser } from '@/test/users';
import type { BinItem } from './binText';
import { RecycleBinPage } from './RecycleBinPage';
import { binItem, lineItemEntry, restorePreview } from './testData';

const BIN = 'GET /api/recycle-bin';
const PARENT = { type: 'initiative', typeLabel: 'initiative', id: 'I-1009', name: 'Trade shows' };

function binPage(items: BinItem[]) {
  return { items, total: items.length };
}

function renderPage(answers: Record<string, MockHandler>, route = '/recycle-bin') {
  const api = mockApi(answers);
  const rendered = renderWithProviders(<RecycleBinPage />, {
    route,
    path: '/recycle-bin',
    user: testUser(),
  });
  return { ...rendered, api };
}

/** The rows of the list, by the name in their heading. */
function rowNames(): string[] {
  return screen.getAllByRole('heading', { level: 2 }).map((heading) => heading.textContent ?? '');
}

describe('RecycleBinPage', () => {
  it('shows the header with the count line and Empty recycle bin, and every row (FR-014, FR-015)', async () => {
    renderPage({ [BIN]: { body: binPage([binItem(), lineItemEntry()]) } });

    expect(
      screen.getByRole('heading', { level: 1, name: 'Deleted, not gone.' }),
    ).toBeInTheDocument();
    expect(document.title).toBe('Recycle bin · NorthStar');
    expect(
      await screen.findByText(
        '2 items in the recycle bin · Items stay here until you restore or delete them',
      ),
    ).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Empty recycle bin' })).toBeInTheDocument();
    expect(rowNames()).toEqual(['Trade shows', 'Print ads']);
  });

  it('says so when the bin is empty, without Empty recycle bin (FR-014)', async () => {
    renderPage({ [BIN]: { body: binPage([]) } });

    expect(
      await screen.findByText(
        'Nothing in the recycle bin — when you delete a plan, campaign, initiative, activity or line item, it comes here so you can bring it back.',
      ),
    ).toBeInTheDocument();
    expect(screen.getByText(/^0 items in the recycle bin/)).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Empty recycle bin' })).not.toBeInTheDocument();
  });

  it('searches the contents of each entry too, and counts what it shows (FR-016)', async () => {
    const { user } = renderPage({ [BIN]: { body: binPage([binItem(), lineItemEntry()]) } });

    await user.type(await screen.findByRole('searchbox', { name: 'Search name or ID' }), 'a-1042');

    expect(rowNames()).toEqual(['Trade shows']);
    expect(screen.getByText("Contains A-1042 'Fleet expo stand'")).toBeInTheDocument();
    expect(screen.getByText('Showing 1 of 2')).toBeInTheDocument();
  });

  it('filters by record type with the pills (FR-016)', async () => {
    const { user } = renderPage({ [BIN]: { body: binPage([binItem(), lineItemEntry()]) } });

    const pills = await screen.findByRole('group', { name: 'Record type' });
    await user.click(within(pills).getByRole('button', { name: 'Line items' }));

    expect(rowNames()).toEqual(['Print ads']);
    expect(screen.getByText('Showing 1 of 2')).toBeInTheDocument();
  });

  it('says when the filters hide everything, with Clear filters', async () => {
    const { user } = renderPage({ [BIN]: { body: binPage([binItem(), lineItemEntry()]) } });

    await user.type(await screen.findByRole('searchbox', { name: 'Search name or ID' }), 'zzz');
    expect(screen.getByText('Nothing matches these filters')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Clear filters' }));

    expect(rowNames()).toEqual(['Trade shows', 'Print ads']);
    expect(screen.getByRole('searchbox', { name: 'Search name or ID' })).toHaveValue('');
  });

  it('highlights the row holding the record of a link from the activity log (FR-008)', async () => {
    renderPage(
      { [BIN]: { body: binPage([lineItemEntry(), binItem()]) } },
      '/recycle-bin?record=A-1042',
    );

    const heading = await screen.findByRole('heading', { level: 2, name: 'Trade shows' });
    expect(heading).toHaveFocus();
    expect(heading.closest('li')).toHaveAttribute('data-highlighted', 'true');
    expect(screen.getByText("Activity 'Fleet expo stand' (A-1042)")).toBeVisible();
  });

  it('says when the linked record is no longer in the recycle bin (FR-008)', async () => {
    renderPage({ [BIN]: { body: binPage([binItem()]) } }, '/recycle-bin?record=A-9999');

    expect(
      await screen.findByText('That item is no longer in the recycle bin.'),
    ).toBeInTheDocument();
  });

  it('shows placeholders while the bin loads', () => {
    renderPage({ [BIN]: () => new Promise<never>(() => {}) });

    expect(screen.getAllByRole('status')).toHaveLength(4);
  });

  it('says why the bin could not be loaded, with Try again', async () => {
    renderPage({ [BIN]: { status: 500 } });

    expect(await screen.findByRole('alert')).toHaveTextContent(
      "NorthStar can't reach its server right now.",
    );
    expect(screen.getByRole('button', { name: 'Try again' })).toBeInTheDocument();
  });

  it('restores a row and asks for the bin again (FR-018, FR-026)', async () => {
    let restored = false;
    const { user, api } = renderPage({
      [BIN]: () => ({ body: binPage(restored ? [lineItemEntry()] : [binItem(), lineItemEntry()]) }),
      'GET /api/recycle-bin/BIN-1001/restore-preview': { body: restorePreview() },
      'POST /api/recycle-bin/BIN-1001/restore': () => {
        restored = true;
        return {
          body: {
            recordType: 'initiative',
            recordTypeLabel: 'initiative',
            recordId: 'I-1009',
            recordName: 'Trade shows',
            details: '',
          },
        };
      },
    });

    await user.click(
      await screen.findByRole('button', { name: "Restore initiative 'Trade shows'" }),
    );
    await user.click(await screen.findByRole('button', { name: 'Restore initiative' }));

    expect(await findMessages()).toHaveTextContent("Initiative 'Trade shows' restored");
    await waitFor(() => expect(rowNames()).toEqual(['Print ads']));
    expect(api.requests.filter((request) => request.path === '/api/recycle-bin')).toHaveLength(2);
  });

  it('deletes a row permanently from its menu (FR-028)', async () => {
    const { user, api } = renderPage({
      [BIN]: { body: binPage([binItem()]) },
      'DELETE /api/recycle-bin/BIN-1001': {
        body: {
          recordType: 'initiative',
          recordTypeLabel: 'initiative',
          recordId: 'I-1009',
          recordName: 'Trade shows',
        },
      },
    });

    await user.click(
      await screen.findByRole('button', { name: "Actions for initiative 'Trade shows'" }),
    );
    await user.click(await screen.findByRole('menuitem', { name: 'Delete permanently' }));
    await user.click(
      within(
        screen.getByRole('alertdialog', { name: "Delete 'Trade shows' permanently?" }),
      ).getByRole('button', { name: 'Delete permanently' }),
    );

    expect(await findMessages()).toHaveTextContent("Initiative 'Trade shows' deleted permanently");
    expect(api.requests.some((request) => request.method === 'DELETE')).toBe(true);
  });

  it('opens Delete permanently from a restore that cannot happen (FR-024)', async () => {
    const blocked = lineItemEntry({
      id: 'BIN-1001',
      restoreState: 'parentDeletedPermanently',
      missingParent: PARENT,
    });
    const { user } = renderPage({ [BIN]: { body: binPage([blocked]) } });

    await user.click(await screen.findByRole('button', { name: "Restore line item 'Print ads'" }));
    await user.click(
      within(
        screen.getByRole('alertdialog', { name: "This line item can't be restored" }),
      ).getByRole('button', { name: 'Delete permanently' }),
    );

    expect(
      await screen.findByRole('alertdialog', { name: "Delete 'Print ads' permanently?" }),
    ).toBeInTheDocument();
  });

  it('empties every listed entry, also those the search hides (FR-029)', async () => {
    const { user, api } = renderPage({
      [BIN]: { body: binPage([binItem(), lineItemEntry()]) },
      'POST /api/recycle-bin/empty': { body: { count: 2 } },
    });

    await user.type(await screen.findByRole('searchbox', { name: 'Search name or ID' }), 'print');
    await user.click(screen.getByRole('button', { name: 'Empty recycle bin' }));
    expect(screen.getByRole('alertdialog', { name: 'Empty the recycle bin?' })).toHaveTextContent(
      'This permanently deletes 2 items',
    );
    await user.click(
      within(screen.getByRole('alertdialog')).getByRole('button', { name: 'Empty recycle bin' }),
    );

    expect(await findMessages()).toHaveTextContent(
      'The recycle bin is empty — 2 items deleted permanently',
    );
    expect(api.bodiesOf('POST /api/recycle-bin/empty')).toEqual([
      { entryIds: ['BIN-1001', 'BIN-1002'] },
    ]);
  });
});
