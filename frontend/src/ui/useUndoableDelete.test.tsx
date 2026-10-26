import { act, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';
import type { components } from '@/api/generated/schema';
import type { CurrentUser } from '@/session/useSession';
import { mockApi } from '@/testing/mockApi';
import { findMessages, renderWithProviders } from '@/testing/render';
import { useUndoableDelete } from './useUndoableDelete';
import { testUser } from '@/testing/users';

type DeleteResult = components['schemas']['DeleteResult'];

const PRIYA: CurrentUser = testUser();

/** The API's answer to deleting the sample folder "Ideas", which held two notes. */
const IDEAS_DELETED: DeleteResult = {
  recycleBinEntry: {
    id: 'BIN-1001',
    recordType: 'sampleFolder',
    recordTypeLabel: 'sample folder',
    recordId: 'SF-1001',
    recordName: 'Ideas',
    path: [],
    parent: null,
    deletedAt: '2026-03-12T09:00:00Z',
    deletedBy: { id: 'U-1002', name: 'Priya Raman' },
    contents: [{ recordType: 'sampleNote', label: 'sample notes', count: 2 }],
  },
};

/** The API's answer to deleting the sample note "Agenda" of the folder "Plans". */
const AGENDA_DELETED: DeleteResult = {
  recycleBinEntry: {
    id: 'BIN-1002',
    recordType: 'sampleNote',
    recordTypeLabel: 'sample note',
    recordId: 'SN-1003',
    recordName: 'Agenda',
    path: [{ type: 'sampleFolder', id: 'SF-1002', name: 'Plans' }],
    parent: { type: 'sampleFolder', id: 'SF-1002', name: 'Plans' },
    deletedAt: '2026-03-12T09:01:00Z',
    deletedBy: { id: 'U-1002', name: 'Priya Raman' },
    contents: [],
  },
};

const IDEAS_RESTORED = {
  recordType: 'sampleFolder',
  recordTypeLabel: 'sample folder',
  recordId: 'SF-1001',
  recordName: 'Ideas',
  details: '',
};

const UNDO_IDEAS = 'POST /api/v1/recycle-bin/BIN-1001/undo';

const IDEAS_MOVED = "Sample folder 'Ideas' moved to the recycle bin";

const UNREACHABLE =
  "MAPP can't reach its server right now. Check that it is running and try again.";

interface DeleteButtonProps {
  /** What the API answered to the delete. */
  result: DeleteResult;
  label?: string;
  altText?: string;
}

/** A button that does what a page does after a delete: it shows the message with Undo. */
function DeleteButton({ result, label = 'Delete', altText }: DeleteButtonProps) {
  const afterDelete = useUndoableDelete({
    invalidate: [['sample-folders'], ['recycle-bin']],
    altText,
  });
  return (
    <button type="button" onClick={() => afterDelete(result)}>
      {label}
    </button>
  );
}

/**
 * Deletes "Ideas" and clicks Undo in its message. Returns the messages area and a spy on the
 * refreshing of lists, which watches from before the Undo.
 */
async function deleteIdeasAndUndo() {
  const rendered = renderWithProviders(<DeleteButton result={IDEAS_DELETED} />, { user: PRIYA });
  const invalidate = vi.spyOn(rendered.queryClient, 'invalidateQueries');
  await rendered.user.click(screen.getByRole('button', { name: 'Delete' }));
  const messages = await findMessages();
  await rendered.user.click(within(messages).getByRole('button', { name: 'Undo' }));
  return { ...rendered, messages, invalidate };
}

afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
});

describe('useUndoableDelete', () => {
  it('says what moved to the recycle bin and offers Undo (FR-026, design system §6.5)', async () => {
    const { user } = renderWithProviders(<DeleteButton result={IDEAS_DELETED} />, {
      user: PRIYA,
    });

    await user.click(screen.getByRole('button', { name: 'Delete' }));

    const messages = await findMessages();
    expect(messages).toHaveTextContent(IDEAS_MOVED);
    expect(within(messages).getByRole('button', { name: 'Undo' })).toBeInTheDocument();
  });

  it('tells screen readers where else to restore it: the recycle bin, unless told otherwise (FR-026)', async () => {
    const { user } = renderWithProviders(
      <>
        <DeleteButton result={IDEAS_DELETED} label="Delete Ideas" />
        <DeleteButton
          result={AGENDA_DELETED}
          label="Delete Agenda"
          altText="Restore it from Recently deleted on this page"
        />
      </>,
      { user: PRIYA },
    );

    await user.click(screen.getByRole('button', { name: 'Delete Ideas' }));
    await waitFor(() =>
      expect(screen.getByRole('status')).toHaveTextContent(
        `${IDEAS_MOVED} Restore it from the recycle bin`,
      ),
    );

    await user.click(screen.getByRole('button', { name: 'Delete Agenda' }));
    await waitFor(() =>
      expect(screen.getByRole('status')).toHaveTextContent(
        "Sample note 'Agenda' moved to the recycle bin Restore it from Recently deleted on this page",
      ),
    );
  });

  it('offers Undo for 10 seconds (FR-026, SC-005)', async () => {
    // Fake timers that also tick on their own, as Testing Library waits for real time.
    vi.useFakeTimers({ shouldAdvanceTime: true });
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    renderWithProviders(<DeleteButton result={IDEAS_DELETED} />, { user: PRIYA });
    await user.click(screen.getByRole('button', { name: 'Delete' }));

    act(() => vi.advanceTimersByTime(9_900));
    expect(screen.getByRole('button', { name: 'Undo' })).toBeInTheDocument();

    act(() => vi.advanceTimersByTime(200));
    expect(screen.queryByRole('button', { name: 'Undo' })).not.toBeInTheDocument();
  });

  it('replaces the earlier Undo message with the new one, wherever it came from (FR-026, D-037)', async () => {
    const { user } = renderWithProviders(
      <>
        <DeleteButton result={IDEAS_DELETED} label="Delete Ideas" />
        <DeleteButton result={AGENDA_DELETED} label="Delete Agenda" />
      </>,
      { user: PRIYA },
    );

    await user.click(screen.getByRole('button', { name: 'Delete Ideas' }));
    await user.click(screen.getByRole('button', { name: 'Delete Agenda' }));

    const messages = await findMessages();
    expect(messages).toHaveTextContent("Sample note 'Agenda' moved to the recycle bin");
    expect(messages).not.toHaveTextContent(IDEAS_MOVED);
    expect(within(messages).getAllByRole('button', { name: 'Undo' })).toHaveLength(1);
  });

  it('brings the record back with Undo and says so (FR-026, US5 scenario 3)', async () => {
    const api = mockApi({ [UNDO_IDEAS]: { body: IDEAS_RESTORED } });

    const { messages } = await deleteIdeasAndUndo();

    await waitFor(() => expect(messages).toHaveTextContent("Sample folder 'Ideas' restored"));
    expect(messages).not.toHaveTextContent(IDEAS_MOVED);
    expect(api.requests).toEqual([
      { method: 'POST', path: '/api/v1/recycle-bin/BIN-1001/undo', search: '', body: undefined },
    ]);
  });

  it('refreshes the lists it was given after an Undo (FR-026)', async () => {
    mockApi({ [UNDO_IDEAS]: { body: IDEAS_RESTORED } });

    const { invalidate } = await deleteIdeasAndUndo();

    await waitFor(() => expect(invalidate).toHaveBeenCalledTimes(2));
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['sample-folders'] });
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['recycle-bin'] });
  });

  it.each([
    ['undo_expired', 'Undo is no longer available — restore it from the recycle bin', {}],
    [
      'restore_parent_first',
      "Restore 'Plans' first",
      { parent: { type: 'sampleFolder', id: 'SF-1002', name: 'Plans' } },
    ],
    [
      'name_taken',
      "Can't undo — there is now another sample folder called 'Ideas'. Restore it from the recycle bin instead.",
      {},
    ],
  ])(
    'says why an Undo was refused, %s, and refreshes the lists (FR-025, FR-026, edge case)',
    async (code, message, extra) => {
      mockApi({
        [UNDO_IDEAS]: { status: 409, body: { error: { code, message, fields: [], ...extra } } },
      });

      const { messages, invalidate } = await deleteIdeasAndUndo();

      await waitFor(() => expect(messages).toHaveTextContent(message));
      // A refusal is final: trying again would not help.
      expect(within(messages).queryByRole('button', { name: 'Try again' })).not.toBeInTheDocument();
      await waitFor(() => expect(invalidate).toHaveBeenCalledWith({ queryKey: ['recycle-bin'] }));
    },
  );

  it('says who already undid or restored it, in the person’s own time (FR-025)', async () => {
    vi.setSystemTime(new Date('2026-03-12T09:06:00Z'));
    mockApi({
      [UNDO_IDEAS]: {
        status: 409,
        body: {
          error: {
            code: 'already_resolved',
            message: "'Ideas' is no longer in the recycle bin.",
            fields: [],
            resolution: {
              how: 'restored',
              by: { id: 'U-1003', name: 'Daniel Okoro' },
              at: '2026-03-12T09:05:00Z',
            },
          },
        },
      },
    });

    const { messages } = await deleteIdeasAndUndo();

    await waitFor(() =>
      expect(messages).toHaveTextContent(
        "'Ideas' is no longer in the recycle bin — Daniel Okoro restored it at 09:05",
      ),
    );
  });

  it('offers Try again when the server cannot be reached (FR-020)', async () => {
    let attempts = 0;
    const api = mockApi({
      [UNDO_IDEAS]: () =>
        ++attempts === 1
          ? Promise.reject(new TypeError('Failed to fetch'))
          : { body: IDEAS_RESTORED },
    });
    const { user, messages } = await deleteIdeasAndUndo();
    await waitFor(() => expect(messages).toHaveTextContent(UNREACHABLE));
    await findMessages();

    await user.click(within(messages).getByRole('button', { name: 'Try again' }));

    await waitFor(() => expect(messages).toHaveTextContent("Sample folder 'Ideas' restored"));
    expect(api.bodiesOf(UNDO_IDEAS)).toHaveLength(2);
  });

  it('leaves a lost session to the sign-in page, without a message of its own (US4 scenario 2)', async () => {
    mockApi({
      [UNDO_IDEAS]: {
        status: 401,
        body: { error: { code: 'session_expired', message: 'Please sign in again', fields: [] } },
      },
    });

    const { messages } = await deleteIdeasAndUndo();

    expect(await screen.findByText('Sign-in page')).toBeInTheDocument();
    expect(messages).not.toHaveTextContent('Please sign in again');
  });
});
