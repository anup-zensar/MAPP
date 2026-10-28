import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { HttpHandler } from 'msw';
import type { components } from '@/api/generated/schema';
import { expectNoAxeViolations } from '@/testing/axe';
import {
  APPROVAL_ACTIONS,
  HIDDEN,
  OPEN_ROUND,
  YES,
  approvalSection,
  no,
  purchaseRequest,
  purchaseRequestApproval,
  routedPurchaseRequest,
  waitingApproval,
} from '@/testing/fixtures/purchase-requests';
import { testUser } from '@/testing/fixtures/users';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { apiError, pending, reply } from '@/testing/replies';
import type { Answer } from '@/testing/replies';
import buttonStyles from '@/ui/Button/Button.module.css';
import { PurchaseRequestApprovalSection } from './PurchaseRequestApprovalSection';

type PurchaseRequest = components['schemas']['PurchaseRequest'];

const GET = 'GET /api/v1/purchase-requests/{purchaseRequestId}/approval';
const RECALL = 'POST /api/v1/purchase-requests/{purchaseRequestId}/approval/recall';

function renderSection(
  answer: Answer<typeof GET>,
  pr: PurchaseRequest = purchaseRequest(),
  extra: HttpHandler[] = [],
) {
  const api = mockApi([reply(GET, answer, { purchaseRequestId: 'PR-1002' }), ...extra]);
  return {
    api,
    ...renderWithProviders(<PurchaseRequestApprovalSection purchaseRequest={pr} />, {
      user: testUser(),
    }),
  };
}

function approvalCard(): HTMLElement {
  return screen.getByRole('region', { name: 'Approval' });
}

const NOT_EDITABLE = purchaseRequest({
  status: 'submitted',
  access: { ...purchaseRequest().access, edit: no('') },
});

describe('032 FR-023–FR-026: PR approval section', () => {
  it('shows a placeholder while the approval loads', async () => {
    renderSection(() => pending());

    expect(within(approvalCard()).getByRole('status')).toHaveTextContent('Loading…');
    await expectNoAxeViolations();
  });

  it('shows a failure with Try again', async () => {
    renderSection({ status: 500, body: apiError('internal_error', 'Something went wrong.') });

    expect(await screen.findByRole('button', { name: 'Try again' })).toBeVisible();
  });

  it('says in place what the server says when the approval is not for this person', async () => {
    renderSection({ status: 404, body: apiError('not_found', 'Purchase request not found.') });

    expect(await screen.findByText('Purchase request not found.')).toBeVisible();
  });

  it('before the first submission: the workflow, the levels to come and Submit', async () => {
    renderSection({ body: purchaseRequestApproval() });

    const card = approvalCard();
    expect(await within(card).findByText('Not submitted yet')).toBeVisible();
    expect(card).toHaveTextContent('Approval workflow: Purchase request approval (MoA)');
    expect(card).toHaveTextContent('Daniel Okoro');
    expect(within(card).getByRole('button', { name: 'Submit for approval' })).toBeVisible();
    expect(within(card).getByText('No approval history yet.')).toBeVisible();
    await expectNoAxeViolations();
  });

  it('032 FR-027: the approver of the waiting level sees Approve (the only primary) and Decline', async () => {
    renderSection({ body: waitingApproval() }, NOT_EDITABLE);

    const card = approvalCard();
    const approve = await within(card).findByRole('button', { name: 'Approve' });
    expect(approve).toHaveClass(String(buttonStyles.primary));
    expect(within(card).getByRole('button', { name: 'Decline' })).not.toHaveClass(
      String(buttonStyles.primary),
    );
    expect(card).toHaveTextContent('Waiting');
    expect(within(card).getByRole('region', { name: 'Approval history' })).toHaveTextContent(
      'Submitted',
    );
    await expectNoAxeViolations();
  });

  it('keeps Edit the page primary: Approve is secondary when the person may also edit', async () => {
    renderSection({ body: waitingApproval() }, purchaseRequest({ status: 'submitted' }));

    expect(await screen.findByRole('button', { name: 'Approve' })).not.toHaveClass(
      String(buttonStyles.primary),
    );
  });

  it('032 FR-025: declining needs a comment before anything is sent', async () => {
    const { api, user } = renderSection({ body: waitingApproval() }, NOT_EDITABLE);

    await user.click(await screen.findByRole('button', { name: 'Decline' }));
    const dialog = await screen.findByRole('alertdialog');
    await user.click(within(dialog).getByRole('button', { name: 'Decline' }));

    expect(await within(dialog).findByText('Add a comment to say why you decline.')).toBeVisible();
    expect(api.requests.filter((request) => request.method === 'POST')).toEqual([]);
  });

  it('032 FR-024: the owner recalls the open round, which returns the PR to Created', async () => {
    const recallable = purchaseRequestApproval({
      status: 'submitted',
      sections: [
        approvalSection({
          rounds: [OPEN_ROUND],
          actions: { ...APPROVAL_ACTIONS, submit: HIDDEN, resubmit: YES, recall: YES },
        }),
      ],
    });
    const { api, user } = renderSection({ body: recallable }, purchaseRequest(), [
      reply(RECALL, { body: purchaseRequestApproval() }),
    ]);

    await user.click(await screen.findByRole('button', { name: 'Recall' }));
    const dialog = await screen.findByRole('alertdialog', {
      name: "Recall the submission of 'PR for Stand build'?",
    });
    expect(dialog).toHaveTextContent('The open level is cancelled. It returns to Created.');
    await expectNoAxeViolations();
    await user.click(within(dialog).getByRole('button', { name: 'Recall' }));

    expect(await screen.findByText("Submission of 'PR for Stand build' recalled")).toBeVisible();
    expect(api.bodiesOf('POST /api/v1/purchase-requests/PR-1002/approval/recall')).toEqual([
      { roundId: 'AR-1101' },
    ]);
  });

  it('032 FR-026, FR-036: a change in review comes first, then the first approval, each with its history', async () => {
    const change = approvalSection({
      kind: 'modifiedPurchaseRequest',
      workflow: { id: 'WF-1102', name: 'Purchase request change approval (MoA)', active: true },
      rounds: [{ ...OPEN_ROUND, id: 'AR-1102' }],
      actions: { ...APPROVAL_ACTIONS, submit: HIDDEN, recall: YES },
    });
    const first = approvalSection({
      rounds: [{ ...OPEN_ROUND, outcome: 'approved', endedAt: '2026-10-27T11:00:00Z' }],
      actions: {
        ...APPROVAL_ACTIONS,
        submit: no('Edit the purchase request and choose Submit changes.'),
      },
    });
    const { user } = renderSection(
      {
        body: purchaseRequestApproval({
          status: 'inReview',
          changePending: true,
          sections: [change, first],
        }),
      },
      routedPurchaseRequest({ status: 'inReview', changePending: true }),
    );

    const headings = await within(approvalCard()).findAllByRole('heading', { level: 3 });
    expect(headings.map((heading) => heading.textContent)).toEqual([
      'Approval of changes',
      'Approval history',
      'First approval',
      'Approval history',
    ]);
    expect(
      within(approvalCard()).getByText('Edit the purchase request and choose Submit changes.'),
    ).toBeVisible();
    expect(approvalCard()).toHaveTextContent(
      'Approval workflow: Purchase request change approval (MoA)',
    );

    await user.click(screen.getByRole('button', { name: 'Recall' }));
    expect(await screen.findByRole('alertdialog')).toHaveTextContent(
      'The open level is cancelled. The approved version is restored.',
    );
  });

  it('032 FR-025: shows the latest decline with its comment', async () => {
    renderSection({
      body: purchaseRequestApproval({
        status: 'declined',
        sections: [
          approvalSection({
            rounds: [{ ...OPEN_ROUND, outcome: 'declined' }],
            latestDecline: {
              roundNumber: 1,
              level: 1,
              levelLabel: 'MoA approver',
              actor: { id: 'U-1003', name: 'Daniel Okoro' },
              onBehalfOf: [],
              at: '2026-10-27T11:00:00Z',
              comment: 'Wrong vendor',
            },
            actions: { ...APPROVAL_ACTIONS, submit: HIDDEN, resubmit: YES },
          }),
        ],
      }),
    });

    expect(await screen.findByText(/^Declined by Daniel Okoro on 27 Oct 2026/)).toBeVisible();
    expect(screen.getByText('Wrong vendor')).toBeVisible();
    expect(screen.getByRole('button', { name: 'Re-submit' })).toBeVisible();
  });
});
