import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { mockApi } from '@/testing/mockApi';
import type { MockHandler } from '@/testing/mockApi';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/users';
import { APPROVED_NO_RECALL } from './messages';
import { RecallDialog } from './RecallDialog';
import { approvalOf, DANIEL, roundOf } from './testData';

const RECALL = 'POST /api/v1/activities/A-1042/approval/recall';

/** "Launch film" after the recall: Created, round 1 Recalled. */
const RECALLED = approvalOf({
  status: 'created',
  displayStatus: 'created',
  rounds: [roundOf({ outcome: 'recalled', endedAt: '2026-03-12T10:00:00Z' })],
});

function renderDialog(answers: Record<string, MockHandler> = {}) {
  const api = mockApi(answers);
  const onClose = vi.fn();
  const rendered = renderWithProviders(<RecallDialog approval={approvalOf()} onClose={onClose} />, {
    user: testUser(),
  });
  return { ...rendered, api, onClose };
}

describe('RecallDialog (feature 008, US5)', () => {
  it('says what a recall does, with an optional reason and Cancel focused (FR-021)', () => {
    renderDialog();

    const dialog = screen.getByRole('alertdialog', {
      name: "Recall the submission of 'Launch film'?",
    });
    expect(dialog).toHaveTextContent(
      "It goes back to Created and leaves the approvers' lists. You can submit it again later.",
    );
    expect(within(dialog).getByRole('textbox', { name: 'Reason (optional)' })).toBeInTheDocument();
    expect(within(dialog).getByRole('button', { name: 'Cancel' })).toHaveFocus();
    expect(within(dialog).getByRole('button', { name: 'Recall submission' })).toBeInTheDocument();
  });

  it('recalls the round the person saw, with the reason, and says so (FR-021)', async () => {
    const { user, api, onClose } = renderDialog({ [RECALL]: { body: RECALLED } });

    await user.type(
      screen.getByRole('textbox', { name: 'Reason (optional)' }),
      '  Budget moves to Q3  ',
    );
    await user.click(screen.getByRole('button', { name: 'Recall submission' }));

    expect(await screen.findByText("Submission of 'Launch film' recalled")).toBeInTheDocument();
    expect(api.bodiesOf(RECALL)).toEqual([{ roundId: 'AR-1001', reason: 'Budget moves to Q3' }]);
    expect(onClose).toHaveBeenCalled();
  });

  it('sends no reason when none is given', async () => {
    const { user, api } = renderDialog({ [RECALL]: { body: RECALLED } });

    await user.click(screen.getByRole('button', { name: 'Recall submission' }));

    await waitFor(() => expect(api.bodiesOf(RECALL)).toEqual([{ roundId: 'AR-1001' }]));
  });

  it('explains a round that ended meanwhile, with its time, and changes nothing (FR-018)', async () => {
    const { user, onClose } = renderDialog({
      [RECALL]: {
        status: 409,
        body: {
          error: {
            code: 'version_conflict',
            message: 'Approval was restarted by Daniel Okoro',
            fields: [],
            conflict: { changedBy: DANIEL.name, changedAt: new Date().toISOString() },
          },
        },
      },
    });

    await user.click(screen.getByRole('button', { name: 'Recall submission' }));

    expect(
      await screen.findByText(/^Approval was restarted by Daniel Okoro at \d\d:\d\d\.$/),
    ).toBeInTheDocument();
    expect(onClose).toHaveBeenCalled();
  });

  it('shows why an Approved activity cannot be recalled (FR-021)', async () => {
    const { user } = renderDialog({
      [RECALL]: {
        status: 400,
        body: { error: { code: 'bad_request', message: APPROVED_NO_RECALL, fields: [] } },
      },
    });

    await user.click(screen.getByRole('button', { name: 'Recall submission' }));

    expect(await screen.findByText(APPROVED_NO_RECALL)).toBeInTheDocument();
  });

  it('shows a problem with the reason under the field and keeps the dialog open', async () => {
    const { user, onClose } = renderDialog({
      [RECALL]: {
        status: 400,
        body: {
          error: {
            code: 'validation_failed',
            message: 'Check the reason.',
            fields: [{ field: 'reason', message: 'Use 2,000 characters or fewer.' }],
          },
        },
      },
    });

    await user.type(screen.getByRole('textbox', { name: 'Reason (optional)' }), 'Too long');
    await user.click(screen.getByRole('button', { name: 'Recall submission' }));

    await waitFor(() =>
      expect(
        screen.getByRole('textbox', { name: 'Reason (optional)' }),
      ).toHaveAccessibleDescription(expect.stringContaining('Use 2,000 characters or fewer.')),
    );
    expect(onClose).not.toHaveBeenCalled();
  });
});
