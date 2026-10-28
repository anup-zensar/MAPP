import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { HttpHandler } from 'msw';
import type { components } from '@/api/generated/schema';
import { expectNoAxeViolations } from '@/testing/axe';
import {
  HIDDEN,
  PR_ACCESS,
  YES,
  no,
  purchaseRequest,
  purchaseRequestApproval,
  submission,
} from '@/testing/fixtures/purchase-requests';
import { testUser } from '@/testing/fixtures/users';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { apiError, pending, reply } from '@/testing/replies';
import type { Answer } from '@/testing/replies';
import { SubmitPurchaseRequestButton } from './SubmitPurchaseRequestDialog';

type PurchaseRequest = components['schemas']['PurchaseRequest'];

const CHECK = 'GET /api/v1/purchase-requests/{purchaseRequestId}/approval/submission';
const SUBMIT = 'POST /api/v1/purchase-requests/{purchaseRequestId}/approval/submit';
const SUBMIT_PATH = 'POST /api/v1/purchase-requests/PR-1002/approval/submit';

function renderButton(
  check: Answer<typeof CHECK>,
  pr: PurchaseRequest = purchaseRequest(),
  extra: HttpHandler[] = [],
) {
  const api = mockApi([reply(CHECK, check, { purchaseRequestId: 'PR-1002' }), ...extra]);
  return {
    api,
    ...renderWithProviders(<SubmitPurchaseRequestButton purchaseRequest={pr} />, {
      user: testUser(),
    }),
  };
}

async function openDialog(user: ReturnType<typeof renderButton>['user'], label: string) {
  await user.click(screen.getByRole('button', { name: label }));
  return screen.findByRole('dialog');
}

describe('032 FR-021: submit', () => {
  it('shows the workflow and its levels, then submits with the version and the comment', async () => {
    const { api, user } = renderButton({ body: submission() }, purchaseRequest(), [
      reply(SUBMIT, { body: purchaseRequestApproval({ status: 'submitted' }) }),
    ]);

    const dialog = await openDialog(user, 'Submit for approval');
    expect(
      await within(dialog).findByRole('heading', {
        name: "Submit 'PR for Stand build' for approval?",
      }),
    ).toBeVisible();
    expect(dialog).toHaveTextContent('Approval workflow: Purchase request approval (MoA)');
    expect(dialog).toHaveTextContent('Daniel Okoro');
    await expectNoAxeViolations();
    await user.type(
      within(dialog).getByRole('textbox', { name: /Comment for the approvers/ }),
      'Urgent for the November stand',
    );
    await user.click(within(dialog).getByRole('button', { name: 'Submit for approval' }));

    expect(
      await screen.findByText("Purchase request 'PR for Stand build' submitted for approval"),
    ).toBeVisible();
    expect(api.bodiesOf(SUBMIT_PATH)).toEqual([
      { version: 3, comment: 'Urgent for the November stand' },
    ]);
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  it('lists every problem at once, each linked to its field or record, and offers no submit', async () => {
    const problems: components['schemas']['PurchaseRequestProblem'][] = [
      {
        field: '',
        group: 'activity',
        recordId: 'A-1004',
        message:
          "Activity 'Trade magazine ads' must be approved before this purchase request can be submitted.",
      },
      {
        field: '',
        group: 'lineItem',
        recordId: 'L-1006',
        message: "Line item L-1006 'Stand build' has no SWBS code yet (Ready to send to GSAP).",
      },
      {
        field: 'moaApproverId',
        group: 'purchaseRequest',
        recordId: 'PR-1002',
        message: 'MoA approver',
      },
      {
        field: 'lines[0].overallLimit',
        group: 'line',
        recordId: 'PRL-1002',
        message: 'Overall limit',
      },
    ];
    const { user } = renderButton({ body: submission({ ready: false, problems }) });

    const dialog = await openDialog(user, 'Submit for approval');
    expect(await within(dialog).findByText('Submitting needs these things first:')).toBeVisible();
    const links = within(dialog).getAllByRole('link');
    expect(links.map((link) => [link.textContent, link.getAttribute('href')])).toEqual([
      [problems[0]?.message, '/activities/A-1004'],
      [problems[1]?.message, '/line-items/L-1006'],
      ['MoA approver', '/purchase-requests/PR-1002/edit?field=moaApproverId'],
      ['Overall limit', '/purchase-requests/PR-1002/edit?field=lines%5B0%5D.overallLimit'],
    ]);
    expect(within(dialog).queryByRole('button', { name: 'Submit for approval' })).toBeNull();
    await expectNoAxeViolations();
  });

  it('labels a new round Re-submit', async () => {
    const declined = purchaseRequest({
      status: 'declined',
      access: { ...PR_ACCESS, submit: HIDDEN, resubmit: YES },
    });
    const { user } = renderButton({ body: submission({ kind: 'resubmit' }) }, declined);

    const dialog = await openDialog(user, 'Re-submit');
    expect(
      await within(dialog).findByRole('heading', {
        name: "Re-submit 'PR for Stand build' for approval?",
      }),
    ).toBeVisible();
    expect(within(dialog).getByRole('button', { name: 'Re-submit' })).toBeVisible();
  });

  it('shows a placeholder while the check runs', async () => {
    const { user } = renderButton(() => pending());
    const dialog = await openDialog(user, 'Submit for approval');
    expect(within(dialog).getByRole('status')).toHaveTextContent('Loading…');
  });

  it('shows a failed check with Try again', async () => {
    const { user } = renderButton({
      status: 500,
      body: apiError('internal_error', 'Something went wrong.'),
    });
    const dialog = await openDialog(user, 'Submit for approval');
    expect(await within(dialog).findByRole('button', { name: 'Try again' })).toBeVisible();
  });

  it('says why a submission is not possible now', async () => {
    const { user } = renderButton({
      body: submission({
        available: false,
        reason: 'Purchase requests are switched off for your organisation.',
      }),
    });
    const dialog = await openDialog(user, 'Submit for approval');
    expect(
      await within(dialog).findByText('Purchase requests are switched off for your organisation.'),
    ).toBeVisible();
  });

  it('lists every missing field the refusal names when something changed since the check', async () => {
    const refusal = apiError('not_ready', 'This purchase request is not ready to submit.', {
      missing: [
        {
          record: { type: 'lineItem', id: 'L-1006', name: 'Stand build' },
          fields: [{ field: 'swbsCode', label: 'SWBS code', reason: 'No SWBS code yet' }],
        },
        {
          record: { type: 'purchaseRequest', id: 'PR-1002', name: 'PR for Stand build' },
          fields: [
            { field: 'moaApproverId', label: 'MoA approver' },
            { field: 'attributes.SA-1201', label: 'Vendor' },
          ],
        },
      ],
    });
    const { user } = renderButton({ body: submission() }, purchaseRequest(), [
      reply(SUBMIT, { status: 409, body: refusal }),
    ]);

    const dialog = await openDialog(user, 'Submit for approval');
    await user.click(await within(dialog).findByRole('button', { name: 'Submit for approval' }));

    expect(await within(dialog).findByText('Submitting needs these things first:')).toBeVisible();
    const links = within(dialog).getAllByRole('link');
    expect(links.map((link) => link.textContent)).toEqual([
      'Stand build: No SWBS code yet',
      'PR for Stand build: MoA approver',
      'PR for Stand build: Vendor',
    ]);
    expect(links[0]).toHaveAttribute('href', '/line-items/L-1006');
    expect(links[2]).toHaveAttribute(
      'href',
      '/purchase-requests/PR-1002/edit?field=attributes.SA-1201',
    );
  });

  it('shows any other refusal in the dialog', async () => {
    const { user } = renderButton({ body: submission() }, purchaseRequest(), [
      reply(SUBMIT, {
        status: 409,
        body: apiError('not_allowed', 'Edit the purchase request and choose Submit changes.'),
      }),
    ]);
    const dialog = await openDialog(user, 'Submit for approval');
    await user.click(await within(dialog).findByRole('button', { name: 'Submit for approval' }));

    expect(await within(dialog).findByRole('alert')).toHaveTextContent(
      'Edit the purchase request and choose Submit changes.',
    );
  });

  it('explains an unavailable submission next to the button', async () => {
    const approved = purchaseRequest({
      status: 'approved',
      access: {
        ...PR_ACCESS,
        submit: no('Edit the purchase request and choose Submit changes.'),
        resubmit: HIDDEN,
      },
    });
    renderButton({ body: submission() }, approved);

    const button = screen.getByRole('button', { name: 'Submit for approval' });
    expect(button).toBeDisabled();
    expect(button).toHaveAccessibleDescription(
      'Edit the purchase request and choose Submit changes.',
    );
  });

  it('shows nothing to a person who may not submit', () => {
    const viewer = purchaseRequest({ access: { ...PR_ACCESS, submit: HIDDEN, resubmit: HIDDEN } });
    renderButton({ body: submission() }, viewer);

    expect(screen.queryByRole('button')).not.toBeInTheDocument();
  });
});
