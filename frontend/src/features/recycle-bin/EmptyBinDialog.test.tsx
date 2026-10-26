import { screen } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it } from 'vitest';
import type { HttpHandler } from 'msw';
import { mockApi } from '@/testing/mock-api';
import { findMessages, renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { binItem, lineItemEntry } from '@/testing/fixtures/recycle-bin';
import { reply } from '@/testing/replies';
import type { BinItem } from './bin-text';
import { EmptyBinDialog } from './EmptyBinDialog';

const EMPTY = 'POST /api/v1/recycle-bin/empty';

/** The dialog as the page shows it: gone once it closes. */
function Harness({ items }: { items: BinItem[] }) {
  const [open, setOpen] = useState(true);
  return open ? <EmptyBinDialog items={items} onClose={() => setOpen(false)} /> : <p>Closed</p>;
}

function renderDialog(answers: HttpHandler[], items: BinItem[]) {
  const api = mockApi(answers);
  return { ...renderWithProviders(<Harness items={items} />, { user: testUser() }), api };
}

describe('EmptyBinDialog', () => {
  it('counts the entries and what is inside them, with Cancel focused (FR-029)', () => {
    renderDialog([], [binItem(), lineItemEntry()]);

    const dialog = screen.getByRole('alertdialog', { name: 'Empty the recycle bin?' });
    expect(dialog).toHaveTextContent(
      'This permanently deletes 2 items and everything inside them: 1 activity and 1 line item.',
    );
    expect(dialog).toHaveTextContent("This can't be undone.");
    expect(screen.getByRole('button', { name: 'Cancel' })).toHaveFocus();
  });

  it('deletes exactly the entries it listed and says how many (FR-029, FR-030)', async () => {
    const { user, api } = renderDialog(
      [reply('POST /api/v1/recycle-bin/empty', { body: { count: 2 } })],
      [binItem(), lineItemEntry()],
    );

    await user.click(screen.getByRole('button', { name: 'Empty recycle bin' }));

    await screen.findByText('Closed');
    expect(api.bodiesOf(EMPTY)).toEqual([{ entryIds: ['BIN-1001', 'BIN-1002'] }]);
    expect(await findMessages()).toHaveTextContent(
      'The recycle bin is empty — 2 items deleted permanently',
    );
  });

  it.each([
    [1, 'The recycle bin is empty — 1 item deleted permanently'],
    [0, 'Nothing was deleted — someone else already emptied the recycle bin.'],
  ])('says so when %i entries were deleted (FR-030)', async (count, message) => {
    const { user } = renderDialog(
      [reply('POST /api/v1/recycle-bin/empty', { body: { count } })],
      [binItem()],
    );

    await user.click(screen.getByRole('button', { name: 'Empty recycle bin' }));

    await screen.findByText('Closed');
    expect(await findMessages()).toHaveTextContent(message);
  });

  it('shows a failure in the dialog and stays open', async () => {
    const { user } = renderDialog(
      [reply('POST /api/v1/recycle-bin/empty', { status: 500 })],
      [binItem()],
    );

    await user.click(screen.getByRole('button', { name: 'Empty recycle bin' }));

    expect(await screen.findByRole('alert')).toHaveTextContent(
      "MAPP can't reach its server right now.",
    );
    expect(screen.getByRole('alertdialog')).toBeInTheDocument();
  });
});
