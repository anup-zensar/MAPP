import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { useState } from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import type { HttpHandler } from 'msw';
import type { components } from '@/api/generated/schema';
import { mockApi } from '@/testing/mock-api';
import { findMessages, renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { reply } from '@/testing/replies';
import type { RecordTarget } from './record-target';
import { DeleteRecordDialog } from './DeleteRecordDialog';

type DeleteResult = components['schemas']['DeleteResult'];

const HELIX: RecordTarget = {
  level: 'campaign',
  id: 'C-1001',
  name: 'Helix launch',
  planId: 'P-1001',
  year: 2026,
  version: 3,
  counts: { initiatives: 3, activities: 12, approvedActivities: 2, lineItems: 20 },
};

const HELIX_DELETED: DeleteResult = {
  recycleBinEntry: {
    id: 'BIN-1001',
    recordType: 'campaign',
    recordTypeLabel: 'campaign',
    recordId: 'C-1001',
    recordName: 'Helix launch',
    path: [{ type: 'plan', id: 'P-1001', name: 'FY2026 Lubricants Europe' }],
    parent: { type: 'plan', id: 'P-1001', name: 'FY2026 Lubricants Europe' },
    deletedAt: '2026-03-12T09:00:00Z',
    deletedBy: { id: 'U-1002', name: 'Priya Raman' },
    contents: [],
  },
};

/** The dialog as a row or panel holds it: closing it forgets the target. */
function Harness({
  target,
  onClose,
  onDeleted,
}: {
  target: RecordTarget;
  onClose: () => void;
  onDeleted: () => void;
}) {
  const [shown, setShown] = useState<RecordTarget | null>(target);
  return (
    <DeleteRecordDialog
      target={shown}
      onClose={() => {
        onClose();
        setShown(null);
      }}
      onDeleted={onDeleted}
    />
  );
}

function renderDialog(target: RecordTarget, answers: HttpHandler[] = [], onDeleted = vi.fn()) {
  const onClose = vi.fn();
  const api = mockApi(answers);
  const rendered = renderWithProviders(
    <Harness target={target} onClose={onClose} onDeleted={onDeleted} />,
    { user: testUser() },
  );
  return { api, onClose, onDeleted, ...rendered };
}

afterEach(() => {
  vi.useRealTimers();
});

describe('DeleteRecordDialog (feature 005, US5)', () => {
  it('names what goes with the campaign, the approved activities and the recycle bin (US5#1, FR-028)', () => {
    renderDialog(HELIX);

    const dialog = screen.getByRole('alertdialog', { name: "Delete campaign 'Helix launch'?" });
    expect(dialog).toHaveTextContent('With it go 3 initiatives, 12 activities and 20 line items.');
    expect(dialog).toHaveTextContent('Plan totals will update.');
    expect(dialog).toHaveTextContent(
      '2 of these activities are approved. If they are restored later from the recycle bin, they return as Created and need approval again.',
    );
    expect(dialog).toHaveTextContent('It moves to the recycle bin; you can restore it.');
    expect(within(dialog).getByRole('button', { name: 'Cancel' })).toHaveFocus();
    expect(within(dialog).getByRole('button', { name: 'Delete campaign' }).className).toContain(
      'danger',
    );
  });

  it('uses the singular for one approved activity and leaves out what is not there (FR-028)', () => {
    renderDialog({
      ...HELIX,
      level: 'initiative',
      name: 'Digital & social',
      counts: { activities: 1, approvedActivities: 1, lineItems: 0 },
    });

    const dialog = screen.getByRole('alertdialog', {
      name: "Delete initiative 'Digital & social'?",
    });
    expect(dialog).toHaveTextContent('With it go 1 activity.');
    expect(dialog).toHaveTextContent(
      '1 of these activities is approved. If it is restored later from the recycle bin, it returns as Created and needs approval again.',
    );
    expect(within(dialog).getByRole('button', { name: 'Delete initiative' })).toBeInTheDocument();
  });

  it('says nothing about contents or approvals when nothing is below', () => {
    renderDialog({
      ...HELIX,
      counts: { initiatives: 0, activities: 0, approvedActivities: 0, lineItems: 0 },
    });

    const dialog = screen.getByRole('alertdialog');
    expect(dialog).not.toHaveTextContent('With it go');
    expect(dialog).not.toHaveTextContent('approved');
    expect(dialog).toHaveTextContent('Plan totals will update.');
  });

  it('deletes, closes the panel and offers Undo for 10 seconds, which restores it (US5#2, FR-029)', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    const { api, onClose, onDeleted } = renderDialog(HELIX, [
      reply(
        'DELETE /api/v1/campaigns/{campaignId}',
        { body: HELIX_DELETED },
        { campaignId: 'C-1001' },
      ),
      reply(
        'POST /api/v1/recycle-bin/{entryId}/undo',
        {
          body: {
            recordType: 'campaign',
            recordTypeLabel: 'campaign',
            recordId: 'C-1001',
            recordName: 'Helix launch',
            details: '',
          },
        },
        { entryId: 'BIN-1001' },
      ),
    ]);

    await user.click(screen.getByRole('button', { name: 'Delete campaign' }));

    await waitFor(() => expect(onDeleted).toHaveBeenCalled());
    expect(onClose).toHaveBeenCalled();
    expect(api.requests.map((r) => `${r.method} ${r.path}`)).toContain(
      'DELETE /api/v1/campaigns/C-1001',
    );
    const messages = await findMessages();
    expect(messages).toHaveTextContent("Campaign 'Helix launch' deleted");

    vi.advanceTimersByTime(9000);
    await user.click(within(messages).getByRole('button', { name: 'Undo' }));

    await waitFor(() => expect(messages).toHaveTextContent("Campaign 'Helix launch' restored"));
    expect(api.bodiesOf('POST /api/v1/recycle-bin/BIN-1001/undo')).toHaveLength(1);
  });

  it('says so when the campaign is already gone (404)', async () => {
    const { user, onClose, onDeleted } = renderDialog(HELIX, [
      reply(
        'DELETE /api/v1/campaigns/{campaignId}',
        {
          status: 404,
          body: {
            error: {
              code: 'not_found',
              message: 'This campaign no longer exists — someone may have deleted it.',
              fields: [],
            },
          },
        },
        { campaignId: 'C-1001' },
      ),
    ]);

    await user.click(screen.getByRole('button', { name: 'Delete campaign' }));

    expect(await findMessages()).toHaveTextContent(
      'This campaign no longer exists — someone may have deleted it.',
    );
    expect(onClose).toHaveBeenCalled();
    expect(onDeleted).not.toHaveBeenCalled();
  });

  it('closes without deleting on Cancel', async () => {
    const { user, onClose, api } = renderDialog(HELIX);

    await user.click(screen.getByRole('button', { name: 'Cancel' }));

    expect(onClose).toHaveBeenCalled();
    expect(api.requests.filter((r) => r.method === 'DELETE')).toHaveLength(0);
  });
});
