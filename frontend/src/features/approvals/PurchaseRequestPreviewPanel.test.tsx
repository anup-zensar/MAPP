import { screen, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import type { HttpHandler } from 'msw';
import type { components } from '@/api/generated/schema';
import { expectNoAxeViolations } from '@/testing/axe';
import { NO_DELEGATIONS, rowOf } from '@/testing/fixtures/approvals';
import { prRow, waitingApproval } from '@/testing/fixtures/purchase-requests';
import { testUser } from '@/testing/fixtures/users';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { apiError, pending, reply } from '@/testing/replies';
import type { Answer } from '@/testing/replies';
import { ApprovalsPage } from './ApprovalsPage';
import { PurchaseRequestPreviewPanel } from './PurchaseRequestPreviewPanel';

type Preview = components['schemas']['PurchaseRequestApprovalPreview'];

const PREVIEW = 'GET /api/v1/approvals/purchase-requests/{purchaseRequestId}';
const APPROVE = 'POST /api/v1/purchase-requests/{purchaseRequestId}/approval/approve';
const DECLINE = 'POST /api/v1/purchase-requests/{purchaseRequestId}/approval/decline';

function preview(): Preview {
  return {
    purchaseRequest: prRow({ status: 'submitted', waitingForGsap: false }),
    approval: waitingApproval(),
  };
}

function renderPanel(answer: Answer<typeof PREVIEW>, extra: HttpHandler[] = []) {
  const api = mockApi([reply(PREVIEW, answer, { purchaseRequestId: 'PR-1002' }), ...extra]);
  const onClose = vi.fn();
  const onDecided = vi.fn();
  const rendered = renderWithProviders(
    <PurchaseRequestPreviewPanel
      purchaseRequestId="PR-1002"
      onClose={onClose}
      onDecided={onDecided}
    />,
    { user: testUser() },
  );
  return { ...rendered, api, onClose, onDecided };
}

describe('032 FR-029: PR in the approvals inbox', () => {
  it('shows the summary, lines, total and approval chain, with a link to the purchase request', async () => {
    renderPanel({ body: preview() });

    const panel = await screen.findByRole('dialog', { name: 'PR for Stand build' });
    expect(panel).toHaveAccessibleDescription('PR-1002 · Submitted');
    const details = within(panel).getByRole('region', { name: 'Details' });
    expect(details).toHaveTextContent('Northlight Events GmbH');
    expect(details).toHaveTextContent('Priya Raman');
    expect(details).toHaveTextContent('FY2026 Lubricants Europe');
    const lines = within(panel).getByRole('region', { name: 'Lines' });
    const row = within(lines).getByRole('row', { name: /Stand build/ });
    expect(row).toHaveTextContent('HX1006');
    expect(row).toHaveTextContent('9,200.00 EUR');
    expect(lines).toHaveTextContent('Total$10,000.00');
    expect(panel).toHaveTextContent('Daniel Okoro');
    expect(within(panel).getByRole('region', { name: 'Approval history' })).toHaveTextContent(
      'Submitted',
    );
    expect(within(panel).getByRole('link', { name: 'Open purchase request' })).toHaveAttribute(
      'href',
      '/purchase-requests/PR-1002#approval',
    );
    await expectNoAxeViolations();
  });

  it('approves the waiting level from the preview and moves on', async () => {
    const { api, user, onDecided } = renderPanel({ body: preview() }, [
      reply(APPROVE, { body: waitingApproval() }),
    ]);

    await user.click(await screen.findByRole('button', { name: 'Approve' }));
    const dialog = await screen.findByRole('alertdialog', {
      name: "Approve purchase request 'PR for Stand build'?",
    });
    expect(dialog).toHaveTextContent('Level 1 of 1 · Daniel Okoro');
    expect(dialog).toHaveTextContent(
      'This is the last level — the purchase request becomes Approved and is routed to the requisitioner.',
    );
    await expectNoAxeViolations();
    await user.click(within(dialog).getByRole('button', { name: 'Approve' }));

    expect(await screen.findByText("Purchase request 'PR for Stand build' approved")).toBeVisible();
    expect(api.bodiesOf('POST /api/v1/purchase-requests/PR-1002/approval/approve')).toEqual([
      { roundId: 'AR-1101', level: 1 },
    ]);
    expect(onDecided).toHaveBeenCalled();
  });

  it('032 FR-025: declining needs a comment, then sends it', async () => {
    const { api, user } = renderPanel({ body: preview() }, [
      reply(DECLINE, { body: waitingApproval() }),
    ]);

    await user.click(await screen.findByRole('button', { name: 'Decline' }));
    const dialog = await screen.findByRole('alertdialog', {
      name: "Decline purchase request 'PR for Stand build'?",
    });
    expect(dialog).toHaveTextContent('The money stays reserved.');
    await user.click(within(dialog).getByRole('button', { name: 'Decline' }));
    expect(await within(dialog).findByText('Add a comment to say why you decline.')).toBeVisible();
    expect(api.bodiesOf('POST /api/v1/purchase-requests/PR-1002/approval/decline')).toEqual([]);

    await user.type(
      within(dialog).getByRole('textbox', { name: /Why are you declining it/ }),
      'Wrong vendor',
    );
    await user.click(within(dialog).getByRole('button', { name: 'Decline' }));

    expect(await screen.findByText("Purchase request 'PR for Stand build' declined")).toBeVisible();
    expect(api.bodiesOf('POST /api/v1/purchase-requests/PR-1002/approval/decline')).toEqual([
      { roundId: 'AR-1101', level: 1, comment: 'Wrong vendor' },
    ]);
  });

  it('offers no decision to someone who may not decide, and says why when the API does', async () => {
    const body = preview();
    const [section] = body.approval.sections;
    if (!section) {
      throw new Error('No section');
    }
    body.approval.sections = [
      {
        ...section,
        actions: {
          ...section.actions,
          approve: { available: false, reason: 'You are not an approver of level 1.' },
          decline: { available: false, reason: '' },
        },
      },
    ];
    renderPanel({ body });

    expect(await screen.findByText('You are not an approver of level 1.')).toBeVisible();
    expect(screen.queryByRole('button', { name: 'Approve' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Decline' })).not.toBeInTheDocument();
  });

  it('shows a placeholder while the preview loads', async () => {
    renderPanel(() => pending());

    expect(await screen.findByRole('dialog', { name: 'Purchase request' })).toHaveTextContent(
      'Loading…',
    );
    await expectNoAxeViolations();
  });

  it('says so when the purchase request is gone or outside the person access', async () => {
    renderPanel({ status: 404, body: apiError('not_found', 'Purchase request not found.') });

    expect(
      await screen.findByText(
        'This purchase request is no longer here — it was deleted, or it is outside your access.',
      ),
    ).toBeVisible();
    expect(screen.queryByRole('button', { name: 'Approve' })).not.toBeInTheDocument();
  });

  it('shows a refusal of the person role in place', async () => {
    renderPanel(
      {
        status: 403,
        body: apiError('forbidden', "Approving purchase requests isn't part of your role."),
      },
      [reply('GET /api/v1/me', { body: testUser() })],
    );

    expect(
      await screen.findByText("Approving purchase requests isn't part of your role."),
    ).toBeVisible();
  });

  it('shows a failure with Try again', async () => {
    renderPanel({ status: 500, body: apiError('internal_error', 'Something went wrong.') });

    expect(await screen.findByRole('button', { name: 'Try again' })).toBeVisible();
  });

  it('032 FR-029: opens this preview, not the activity one, for an inbox row of a purchase-request kind', async () => {
    const prInboxRow = rowOf({
      activity: null,
      subject: {
        kind: 'modifiedPurchaseRequest',
        label: 'Modified purchase request',
        id: 'PR-1002',
        name: 'PR for Stand build',
        status: 'submitted',
        displayStatus: 'submitted',
      },
    });
    mockApi([
      reply('GET /api/v1/approvals/summary', {
        body: { waitingForMe: 1, allOpen: 1, approvedThisMonth: 0, declinedThisMonth: 0 },
      }),
      reply('GET /api/v1/approvals', { body: { items: [prInboxRow], total: 1 } }),
      reply('GET /api/v1/approval-delegations', NO_DELEGATIONS),
      reply(PREVIEW, { body: preview() }, { purchaseRequestId: 'PR-1002' }),
    ]);
    const { user, router } = renderWithProviders(<ApprovalsPage />, {
      route: '/approvals?tab=waiting',
      path: '/approvals',
      user: testUser(),
    });

    await user.click(await screen.findByRole('button', { name: /PR for Stand build/ }));

    const panel = await screen.findByRole('dialog', { name: 'PR for Stand build' });
    expect(within(panel).getByRole('link', { name: 'Open purchase request' })).toBeVisible();
    expect(router.state.location.search).toBe(
      '?tab=waiting&activity=PR-1002&kind=modifiedPurchaseRequest',
    );
  });
});
