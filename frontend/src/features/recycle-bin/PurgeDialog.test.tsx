import { screen } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it, vi } from 'vitest';
import type { HttpHandler } from 'msw';
import { mockApi } from '@/testing/mock-api';
import { findMessages, renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { binItem, lineItemEntry } from '@/testing/fixtures/recycle-bin';
import { reply } from '@/testing/replies';
import type { BinItem } from './bin-text';
import { PurgeDialog } from './PurgeDialog';

/** The dialog as the page shows it: gone once it closes. */
function Harness({ item, onClose }: { item: BinItem; onClose: () => void }) {
  const [open, setOpen] = useState(true);
  if (!open) {
    return <p>Closed</p>;
  }
  return (
    <PurgeDialog
      item={item}
      onClose={() => {
        onClose();
        setOpen(false);
      }}
    />
  );
}

function renderDialog(answers: HttpHandler[], item: BinItem = binItem()) {
  const api = mockApi(answers);
  const onClose = vi.fn();
  const rendered = renderWithProviders(<Harness item={item} onClose={onClose} />, {
    user: testUser(),
  });
  return { ...rendered, api, onClose };
}

describe('PurgeDialog', () => {
  it('says what goes with it and that it cannot be undone, with Cancel focused (FR-028)', () => {
    renderDialog([]);

    const dialog = screen.getByRole('alertdialog', { name: "Delete 'Trade shows' permanently?" });
    expect(dialog).toHaveTextContent('This also deletes 1 activity and 1 line item.');
    expect(dialog).toHaveTextContent("This can't be undone.");
    expect(dialog).not.toHaveTextContent('belong');
    expect(screen.getByRole('button', { name: 'Cancel' })).toHaveFocus();
    expect(screen.getByRole('button', { name: 'Delete permanently' })).toBeInTheDocument();
  });

  it.each([
    [
      ['BIN-1005'],
      '1 other item in the recycle bin belongs inside it and can no longer be restored.',
    ],
    [
      ['BIN-1005', 'BIN-1006'],
      '2 other items in the recycle bin belong inside it and can no longer be restored.',
    ],
  ])('warns about the entries inside it (%j) (FR-028)', (insideEntryIds, warning) => {
    renderDialog([], binItem({ insideEntryIds }));

    expect(screen.getByRole('alertdialog')).toHaveTextContent(warning);
  });

  it('leaves out the contents sentence when nothing was inside (FR-028)', () => {
    renderDialog([], lineItemEntry({ id: 'BIN-1001' }));

    expect(screen.getByRole('alertdialog')).not.toHaveTextContent('This also deletes');
  });

  it('says who already restored it, and closes (FR-032)', async () => {
    const { user } = renderDialog([
      reply(
        'DELETE /api/v1/recycle-bin/{entryId}',
        {
          status: 409,
          body: {
            error: {
              code: 'already_resolved',
              message: 'This item is no longer in the recycle bin.',
              fields: [],
              resolution: {
                how: 'restored',
                by: { id: 'U-1003', name: 'Daniel Okoro' },
                at: '2026-03-12T14:05:00Z',
              },
            },
          },
        },
        { entryId: 'BIN-1001' },
      ),
    ]);

    await user.click(screen.getByRole('button', { name: 'Delete permanently' }));

    await screen.findByText('Closed');
    expect(await findMessages()).toHaveTextContent(
      "'Trade shows' is no longer in the recycle bin — Daniel Okoro restored it on 12 Mar 2026 at 14:05",
    );
  });

  it('shows any other failure in the dialog and stays open', async () => {
    const { user, onClose } = renderDialog([
      reply('DELETE /api/v1/recycle-bin/{entryId}', { status: 500 }, { entryId: 'BIN-1001' }),
    ]);

    await user.click(screen.getByRole('button', { name: 'Delete permanently' }));

    expect(await screen.findByRole('alert')).toHaveTextContent(
      "SP&A One can't reach its server right now.",
    );
    expect(onClose).not.toHaveBeenCalled();
  });
});
