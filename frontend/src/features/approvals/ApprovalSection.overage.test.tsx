import { screen, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { card, renderDialog, renderSection } from '@/testing/fixtures/approval-screens';
import { approvalOf } from '@/testing/fixtures/approvals';
import { testUser } from '@/testing/fixtures/users';
import { expectNoAxeViolations } from '@/testing/axe';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { reply } from '@/testing/replies';
import type { ActivityApproval } from './api';
import { ApprovalStatusAction } from './ApprovalStatusAction';

const WARNING = "Line items of activity 'Launch film' are 25,000.00 USD above its planned cost";
const REFUSAL = `${WARNING} — ask the owner to lower them, or decline.`;
const OVERAGE = {
  activity: { type: 'activity', id: 'A-1042', name: 'Launch film' },
  overBy: { amount: '25000.00', currency: 'USD' },
};
const APPROVER = { isOwner: false, isCurrentApprover: true, onBehalfOf: [] };

function warned(): ActivityApproval {
  return approvalOf({ viewer: APPROVER, overPlannedCost: [OVERAGE] });
}

function refused(): ActivityApproval {
  const approval = warned();
  return {
    ...approval,
    actions: { ...approval.actions, approve: { available: false, reason: REFUSAL } },
  };
}

describe('038 FR-019 (M-18): Warn — the approver sees the overage before approving', () => {
  it('shows the warning per activity in the Approval section, with Approve offered', async () => {
    renderSection({ body: warned() });

    expect(await within(card()).findByText(WARNING)).toBeVisible();
    expect(within(card()).getByRole('button', { name: 'Approve' })).toBeEnabled();
    await expectNoAxeViolations();
  });

  it('repeats the warning in the Approve dialog', async () => {
    renderDialog({ mode: 'approve', approval: warned() });

    const dialog = await screen.findByRole('alertdialog', { name: "Approve 'Launch film'?" });
    expect(dialog).toHaveTextContent(WARNING);
  });

  it('shows nothing when no activity is above its planned cost', async () => {
    renderSection({ body: approvalOf({ viewer: APPROVER }) });

    await within(card()).findByRole('button', { name: 'Approve' });
    expect(card()).not.toHaveTextContent('above its planned cost');
  });
});

describe('038 FR-020 (M-18): Refuse — Approve is shown disabled with the reason', () => {
  it('in the Approval section, with Decline still offered', async () => {
    renderSection({ body: refused() });

    const approve = await within(card()).findByRole('button', { name: 'Approve' });
    expect(approve).toBeDisabled();
    expect(approve).toHaveAccessibleDescription(REFUSAL);
    expect(within(card()).getByRole('button', { name: 'Decline' })).toBeEnabled();
    expect(within(card()).getAllByText(REFUSAL)).toHaveLength(1);
    await expectNoAxeViolations();
  });

  it('in the activity header', async () => {
    mockApi([
      reply(
        'GET /api/v1/activities/{activityId}/approval',
        { body: refused() },
        { activityId: 'A-1042' },
      ),
    ]);
    renderWithProviders(
      <ApprovalStatusAction
        activity={{ id: 'A-1042', name: 'Launch film', status: 'submitted', version: 3 }}
        onShowProblems={vi.fn()}
      />,
      { user: testUser() },
    );

    const approve = await screen.findByRole('button', { name: 'Approve' });
    expect(approve).toBeDisabled();
    expect(approve).toHaveAccessibleDescription(REFUSAL);
    expect(screen.getByRole('button', { name: 'Decline' })).toBeEnabled();
  });
});
