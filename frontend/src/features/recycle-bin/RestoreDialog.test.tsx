import { screen, waitFor } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it, vi } from 'vitest';
import type { HttpHandler } from 'msw';
import { SESSION_KEY } from '@/session/useSession';
import { mockApi } from '@/testing/mock-api';
import { findMessages, renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { binItem, errorBody, lineItemEntry, restorePreview } from '@/testing/fixtures/recycle-bin';
import { reply } from '@/testing/replies';
import { RestoreDialog } from './RestoreDialog';
import type { BinItem } from './bin-text';

const RESTORED = {
  recordType: 'initiative',
  recordTypeLabel: 'initiative',
  recordId: 'I-1009',
  recordName: 'Trade shows',
  details: '',
};

const PARENT = { type: 'initiative', typeLabel: 'initiative', id: 'I-1009', name: 'Trade shows' };

/** The dialog as the page shows it: gone once it closes. */
function Harness({
  item,
  onClose,
  onPurge,
}: {
  item: BinItem;
  onClose: () => void;
  onPurge: (item: BinItem) => void;
}) {
  const [open, setOpen] = useState(true);
  if (!open) {
    return <p>Closed</p>;
  }
  return (
    <RestoreDialog
      item={item}
      onClose={() => {
        onClose();
        setOpen(false);
      }}
      onPurge={onPurge}
    />
  );
}

function renderDialog(answers: HttpHandler[], item: BinItem = binItem()) {
  const api = mockApi(answers);
  const onClose = vi.fn();
  const onPurge = vi.fn();
  const rendered = renderWithProviders(
    <Harness item={item} onClose={onClose} onPurge={onPurge} />,
    { user: testUser() },
  );
  return { ...rendered, api, onClose, onPurge };
}

describe('RestoreDialog', () => {
  it('says where it returns, what changes and the notes, then restores (FR-018, FR-019, FR-022)', async () => {
    const { user, api, onClose } = renderDialog([
      reply(
        'GET /api/v1/recycle-bin/{entryId}/restore-preview',
        {
          body: restorePreview({
            changes: [
              {
                recordType: 'activity',
                recordTypeLabel: 'activity',
                recordId: 'A-1042',
                recordName: 'Fleet expo stand',
                restored: true,
                field: 'status',
                label: 'Status',
                oldText: 'Approved',
                newText: 'Created',
              },
              {
                recordType: 'initiative',
                recordTypeLabel: 'initiative',
                recordId: 'I-1009',
                recordName: 'Trade shows',
                restored: true,
                field: 'name',
                label: 'Name',
                oldText: 'Trade shows',
                newText: 'Trade shows (restored)',
              },
            ],
            notes: ["'Fleet expo stand' comes back as Created — submit it for approval again."],
          }),
        },
        { entryId: 'BIN-1001' },
      ),
      reply(
        'POST /api/v1/recycle-bin/{entryId}/restore',
        { body: RESTORED },
        { entryId: 'BIN-1001' },
      ),
    ]);

    const dialog = await screen.findByRole('alertdialog', {
      name: "Restore initiative 'Trade shows'?",
    });
    expect(
      await screen.findByText(
        'It returns to FY2026 Lubricants Europe › Helix launch with 1 activity and 1 line item.',
      ),
    ).toBeInTheDocument();
    expect(dialog).toHaveTextContent(
      "Activity 'Fleet expo stand' (A-1042) comes back as Created and must be submitted for approval again.",
    );
    expect(dialog).toHaveTextContent(
      "It comes back as 'Trade shows (restored)' because the name 'Trade shows' is taken.",
    );
    expect(dialog).toHaveTextContent(
      "'Fleet expo stand' comes back as Created — submit it for approval again.",
    );

    await user.click(screen.getByRole('button', { name: 'Restore initiative' }));

    await waitFor(() => expect(onClose).toHaveBeenCalled());
    expect(api.requests.find((request) => request.method === 'POST')?.search).toBe(
      '?withParents=false',
    );
  });

  it('names the restored record with View, and asks every page again but the session (FR-026)', async () => {
    const { user, router, queryClient } = renderDialog([
      reply(
        'GET /api/v1/recycle-bin/{entryId}/restore-preview',
        { body: restorePreview() },
        { entryId: 'BIN-1001' },
      ),
      reply(
        'POST /api/v1/recycle-bin/{entryId}/restore',
        { body: RESTORED },
        { entryId: 'BIN-1001' },
      ),
    ]);
    queryClient.setQueryData(['plans'], { items: [], total: 0 });

    await user.click(await screen.findByRole('button', { name: 'Restore initiative' }));

    const messages = await findMessages();
    expect(messages).toHaveTextContent("Initiative 'Trade shows' restored");
    await waitFor(() => expect(queryClient.getQueryState(['plans'])?.isInvalidated).toBe(true));
    expect(queryClient.getQueryState(SESSION_KEY)?.isInvalidated).toBe(false);

    await user.click(screen.getByRole('button', { name: 'View' }));
    await waitFor(() =>
      expect(router.state.location.pathname).toBe('/plans/P-1001/initiatives/I-1009'),
    );
  });

  it('brings the parent back first: Restore both (FR-023)', async () => {
    const item = lineItemEntry({
      id: 'BIN-1001',
      restoreState: 'parentInRecycleBin',
      missingParent: PARENT,
      parentEntryIds: ['BIN-1000'],
    });
    const { user, api } = renderDialog(
      [
        reply(
          'GET /api/v1/recycle-bin/{entryId}/restore-preview',
          {
            body: restorePreview({ entry: item, parentEntries: [binItem({ id: 'BIN-1000' })] }),
          },
          { entryId: 'BIN-1001' },
        ),
        reply(
          'POST /api/v1/recycle-bin/{entryId}/restore',
          { body: { ...RESTORED, recordType: 'lineItem', recordTypeLabel: 'line item' } },
          { entryId: 'BIN-1001' },
        ),
      ],
      item,
    );

    expect(
      await screen.findByText(
        "Its initiative 'Trade shows' is also in the recycle bin. Restoring brings these back first:",
      ),
    ).toBeInTheDocument();
    expect(
      screen.getByText("Initiative 'Trade shows' (I-1009) with 1 activity and 1 line item"),
    ).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Restore both' }));

    await waitFor(() =>
      expect(api.requests.find((request) => request.method === 'POST')?.search).toBe(
        '?withParents=true',
      ),
    );
  });

  it('offers Restore all <N> when more parents come back first (FR-023)', async () => {
    renderDialog([
      reply(
        'GET /api/v1/recycle-bin/{entryId}/restore-preview',
        {
          body: restorePreview({
            entry: binItem({ missingParent: PARENT, restoreState: 'parentInRecycleBin' }),
            parentEntries: [binItem({ id: 'BIN-998' }), binItem({ id: 'BIN-999' })],
          }),
        },
        { entryId: 'BIN-1001' },
      ),
    ]);

    expect(await screen.findByRole('button', { name: 'Restore all 3' })).toBeInTheDocument();
  });

  it('cannot restore when the parent was deleted permanently, and offers Delete permanently (FR-024)', async () => {
    const item = lineItemEntry({
      id: 'BIN-1001',
      restoreState: 'parentDeletedPermanently',
      missingParent: PARENT,
    });
    const { user, api, onClose, onPurge } = renderDialog([], item);

    expect(
      screen.getByRole('alertdialog', { name: "This line item can't be restored" }),
    ).toHaveTextContent(
      "This line item can't be restored because its initiative 'Trade shows' was permanently deleted. You can delete it permanently instead.",
    );
    expect(api.requests).toHaveLength(0);

    await user.click(screen.getByRole('button', { name: 'Delete permanently' }));

    expect(onClose).toHaveBeenCalled();
    expect(onPurge).toHaveBeenCalledWith(item);
  });

  it('says who already resolved the entry, and closes (FR-032)', async () => {
    const { onClose } = renderDialog([
      reply(
        'GET /api/v1/recycle-bin/{entryId}/restore-preview',
        {
          status: 409,
          body: errorBody('already_resolved', 'This item is no longer in the recycle bin.', {
            resolution: {
              how: 'deletedPermanently',
              by: { id: 'U-1003', name: 'Daniel Okoro' },
              at: '2026-03-12T14:05:00Z',
            },
          }),
        },
        { entryId: 'BIN-1001' },
      ),
    ]);

    await screen.findByText('Closed');
    const messages = await findMessages();
    expect(messages).toHaveTextContent(
      "'Trade shows' is no longer in the recycle bin — Daniel Okoro deleted it permanently on 12 Mar 2026 at 14:05",
    );
    expect(onClose).toHaveBeenCalled();
  });

  it('closes with the API’s words when the parent is missing after all (FR-023, FR-024)', async () => {
    const message = "Restore the initiative 'Trade shows' first.";
    const { user, onClose } = renderDialog([
      reply(
        'GET /api/v1/recycle-bin/{entryId}/restore-preview',
        { body: restorePreview() },
        { entryId: 'BIN-1001' },
      ),
      reply(
        'POST /api/v1/recycle-bin/{entryId}/restore',
        {
          status: 409,
          body: errorBody('restore_parent_first', message, { parent: PARENT }),
        },
        { entryId: 'BIN-1001' },
      ),
    ]);

    await user.click(await screen.findByRole('button', { name: 'Restore initiative' }));

    expect(await findMessages()).toHaveTextContent(message);
    expect(onClose).toHaveBeenCalled();
  });

  it('shows any other failure in the dialog and stays open', async () => {
    const { user, onClose } = renderDialog([
      reply(
        'GET /api/v1/recycle-bin/{entryId}/restore-preview',
        { body: restorePreview() },
        { entryId: 'BIN-1001' },
      ),
      reply(
        'POST /api/v1/recycle-bin/{entryId}/restore',
        { status: 400, body: errorBody('bad_request', 'That did not work.') },
        { entryId: 'BIN-1001' },
      ),
    ]);

    await user.click(await screen.findByRole('button', { name: 'Restore initiative' }));

    expect(await screen.findByRole('alert')).toHaveTextContent('That did not work.');
    expect(onClose).not.toHaveBeenCalled();
  });

  it('shows placeholders while the preview loads', () => {
    renderDialog([
      reply(
        'GET /api/v1/recycle-bin/{entryId}/restore-preview',
        () => new Promise<never>(() => {}),
        { entryId: 'BIN-1001' },
      ),
    ]);

    expect(screen.getByRole('status')).toHaveTextContent('Loading…');
    expect(screen.getByRole('button', { name: 'Restore initiative' })).toHaveAttribute(
      'aria-busy',
      'true',
    );
  });
});
