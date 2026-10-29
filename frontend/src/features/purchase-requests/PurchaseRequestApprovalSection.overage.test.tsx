import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { components } from '@/api/generated/schema';
import { expectNoAxeViolations } from '@/testing/axe';
import { purchaseRequest, waitingApproval } from '@/testing/fixtures/purchase-requests';
import { testUser } from '@/testing/fixtures/users';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { reply } from '@/testing/replies';
import { PurchaseRequestApprovalSection } from './PurchaseRequestApprovalSection';

type Approval = components['schemas']['PurchaseRequestApproval'];

const WARNING =
  "Line items of activity 'Billboard winter wave' are 25,000.00 USD above its planned cost";
const REFUSAL = `${WARNING} — ask the owner to lower them, or decline.`;
const OVERAGE = {
  activity: { type: 'activity', id: 'A-1003', name: 'Billboard winter wave' },
  overBy: { amount: '25000.00', currency: 'USD' },
};

function warned(): Approval {
  return { ...waitingApproval(), overPlannedCost: [OVERAGE] };
}

function refused(): Approval {
  const approval = warned();
  const [section] = approval.sections;
  if (!section) {
    throw new Error('waitingApproval has a section');
  }
  const approve = { available: false, reason: REFUSAL };
  return { ...approval, sections: [{ ...section, actions: { ...section.actions, approve } }] };
}

function renderSection(approval: Approval) {
  mockApi([
    reply(
      'GET /api/v1/purchase-requests/{purchaseRequestId}/approval',
      { body: approval },
      { purchaseRequestId: 'PR-1002' },
    ),
  ]);
  return renderWithProviders(
    <PurchaseRequestApprovalSection purchaseRequest={purchaseRequest({ status: 'submitted' })} />,
    { user: testUser() },
  );
}

const card = () => screen.getByRole('region', { name: 'Approval' });

describe('038 FR-019 (M-18): Warn — a purchase-request approver sees the overage', () => {
  it('in the Approval section and in the Approve dialog', async () => {
    const { user } = renderSection(warned());

    expect(await within(card()).findByText(WARNING)).toBeVisible();
    await expectNoAxeViolations();
    await user.click(within(card()).getByRole('button', { name: 'Approve' }));

    expect(await screen.findByRole('alertdialog')).toHaveTextContent(WARNING);
  });
});

describe('038 FR-020 (M-18): Refuse — Approve is disabled with the reason, Decline stays', () => {
  it('in the Approval section', async () => {
    renderSection(refused());

    const approve = await within(card()).findByRole('button', { name: 'Approve' });
    expect(approve).toBeDisabled();
    expect(approve).toHaveAccessibleDescription(REFUSAL);
    expect(within(card()).getByRole('button', { name: 'Decline' })).toBeEnabled();
    expect(within(card()).getAllByText(REFUSAL)).toHaveLength(1);
  });
});
