import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { expectNoAxeViolations } from '@/testing/axe';
import { formAnswers } from '@/testing/fixtures/purchase-request-forms';
import { routedPurchaseRequest, submission } from '@/testing/fixtures/purchase-requests';
import { testUser } from '@/testing/fixtures/users';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { reply } from '@/testing/replies';
import { PurchaseRequestEditPage } from './PurchaseRequestEditPage';

const SAVED = 'PUT /api/v1/purchase-requests/PR-1002';
const DANIEL = { id: 'U-1003', name: 'Daniel Okoro', initials: 'DO', active: true };

const CHANGE_CHECK = submission({
  roundKind: 'modifiedPurchaseRequest',
  available: false,
  reason: 'Approved purchase requests are changed with Submit changes.',
  workflow: { id: 'WF-1102', name: 'Purchase request change approval (MoA)', active: true },
  outcomes: [
    {
      number: 1,
      label: 'Level 1',
      outcome: 'decide',
      approvers: [DANIEL],
      source: 'MoA approver',
      mode: 'any',
      ruleText: '',
      excluded: [],
      backupUsed: false,
      noApprover: false,
      advice: '',
    },
  ],
});

function renderEdit() {
  const api = mockApi([
    ...formAnswers(),
    reply('GET /api/v1/purchase-requests/{purchaseRequestId}', { body: routedPurchaseRequest() }),
    reply('GET /api/v1/purchase-requests/{purchaseRequestId}/approval/submission', {
      body: CHANGE_CHECK,
    }),
    reply('PUT /api/v1/purchase-requests/{purchaseRequestId}', { body: routedPurchaseRequest() }),
  ]);
  return {
    api,
    ...renderWithProviders(<PurchaseRequestEditPage />, {
      route: '/purchase-requests/PR-1002/edit',
      path: '/purchase-requests/:purchaseRequestId/edit',
      user: testUser(),
    }),
  };
}

describe('038 FR-028 (M-10): Submit changes asks first', () => {
  it('says a change round starts, with its workflow and first approvers, before saving', async () => {
    const { api, user } = renderEdit();

    await user.click(await screen.findByRole('button', { name: 'Submit changes' }));

    const dialog = await screen.findByRole('alertdialog', {
      name: "Submit changes to 'PR for Stand build'?",
    });
    expect(
      await within(dialog).findByText('Approval workflow: Purchase request change approval (MoA)'),
    ).toBeVisible();
    expect(dialog).toHaveTextContent('A change approval round starts');
    expect(dialog).toHaveTextContent('Level 1 · Daniel Okoro (MoA approver)');
    expect(dialog).toHaveTextContent(
      'The approved amounts stay reserved until the change is approved.',
    );
    expect(api.bodiesOf(SAVED)).toEqual([]);

    await user.click(within(dialog).getByRole('button', { name: 'Submit changes' }));

    await waitFor(() => expect(api.bodiesOf(SAVED)).toHaveLength(1));
    expect(api.bodiesOf(SAVED)[0]).toMatchObject({ then: 'submitChanges' });
  });

  it('saves nothing when the person cancels', async () => {
    const { api, user } = renderEdit();

    await user.click(await screen.findByRole('button', { name: 'Submit changes' }));
    const dialog = await screen.findByRole('alertdialog');
    await within(dialog).findByText(/^Approval workflow:/);
    await expectNoAxeViolations();
    await user.click(within(dialog).getByRole('button', { name: 'Cancel' }));

    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
    expect(api.bodiesOf(SAVED)).toEqual([]);
  });
});
