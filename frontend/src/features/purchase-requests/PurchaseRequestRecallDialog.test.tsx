import { screen, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { expectNoAxeViolations } from '@/testing/axe';
import { purchaseRequestApproval } from '@/testing/fixtures/purchase-requests';
import { testUser } from '@/testing/fixtures/users';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { apiError, reply } from '@/testing/replies';
import type { Answer } from '@/testing/replies';
import { PurchaseRequestRecallDialog } from './PurchaseRequestRecallDialog';

const RECALL = 'POST /api/v1/purchase-requests/{purchaseRequestId}/approval/recall';

function renderDialog(answer: Answer<typeof RECALL>) {
  const api = mockApi([reply(RECALL, answer)]);
  const onClose = vi.fn();
  const rendered = renderWithProviders(
    <PurchaseRequestRecallDialog
      purchaseRequestId="PR-1002"
      name="PR for Stand build"
      kind="modifiedPurchaseRequest"
      roundId="AR-1102"
      onClose={onClose}
    />,
    { user: testUser() },
  );
  return { api, onClose, ...rendered };
}

describe('032 FR-024/FR-037: recall a change in review', () => {
  it('says the approved version is restored and sends the round and the reason', async () => {
    const { api, onClose, user } = renderDialog({ body: purchaseRequestApproval() });

    const dialog = screen.getByRole('alertdialog', {
      name: "Recall the change of 'PR for Stand build'?",
    });
    expect(dialog).toHaveTextContent(
      'The open level is cancelled. The approved version is restored.',
    );
    await expectNoAxeViolations();
    await user.type(within(dialog).getByLabelText('Reason (optional)'), ' Wrong amount ');
    await user.click(within(dialog).getByRole('button', { name: 'Recall' }));

    expect(await screen.findByText("Change of 'PR for Stand build' recalled")).toBeVisible();
    expect(api.bodiesOf('POST /api/v1/purchase-requests/PR-1002/approval/recall')).toEqual([
      { roundId: 'AR-1102', reason: 'Wrong amount' },
    ]);
    expect(onClose).toHaveBeenCalled();
  });

  it('shows why a recall is refused and stays open', async () => {
    const message = 'The round was decided at 10:42 — reload to see it.';
    const { onClose, user } = renderDialog({
      status: 409,
      body: apiError('already_resolved', message),
    });

    const dialog = screen.getByRole('alertdialog');
    await user.click(within(dialog).getByRole('button', { name: 'Recall' }));

    expect(await within(dialog).findByRole('alert')).toHaveTextContent(message);
    expect(onClose).not.toHaveBeenCalled();
  });
});
