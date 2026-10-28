import { screen, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import type { HttpHandler } from 'msw';
import type { components } from '@/api/generated/schema';
import { expectNoAxeViolations } from '@/testing/axe';
import { HISTORY_ANSWERS } from '@/testing/fixtures/activity-log';
import { attribute } from '@/testing/fixtures/attribute-forms';
import {
  no,
  orderedPurchaseRequest,
  purchaseRequest,
  purchaseRequestApproval,
  routedPurchaseRequest,
} from '@/testing/fixtures/purchase-requests';
import { testUser } from '@/testing/fixtures/users';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { apiError, pending, reply } from '@/testing/replies';
import type { Answer } from '@/testing/replies';
import { PurchaseRequestPage } from './PurchaseRequestPage';

const GET = 'GET /api/v1/purchase-requests/{purchaseRequestId}';

function prForm(): components['schemas']['AttributeForm'] {
  return {
    level: 'purchaseRequest',
    businessUnitAttributeId: null,
    businessUnitOptions: [],
    sections: [
      {
        name: 'Purchase request',
        conditional: false,
        attributes: [
          attribute({ id: 'SA-1201', label: 'Vendor', type: 'text', marking: 'neededToSubmit' }),
        ],
      },
    ],
  };
}

function renderPage(answer: Answer<typeof GET>, extra: HttpHandler[] = []) {
  const api = mockApi([
    ...HISTORY_ANSWERS,
    reply('GET /api/v1/shell-attribute-forms/{level}', { body: prForm() }),
    reply('GET /api/v1/purchase-requests/{purchaseRequestId}/approval', {
      body: purchaseRequestApproval(),
    }),
    reply(GET, answer, { purchaseRequestId: 'PR-1002' }),
    ...extra,
  ]);
  return {
    api,
    ...renderWithProviders(<PurchaseRequestPage />, {
      route: '/purchase-requests/PR-1002',
      path: '/purchase-requests/:purchaseRequestId',
      user: testUser(),
    }),
  };
}

function detail(label: string): HTMLElement {
  const value = screen.getByText(label, { selector: 'dt' }).nextElementSibling;
  if (!(value instanceof HTMLElement)) {
    throw new Error(`No value for ${label}`);
  }
  return value;
}

describe('032 FR-008/FR-009/FR-020/FR-052: purchase request page', () => {
  it('shows the header, its derived fields, the lines and their HX keys (US1)', async () => {
    renderPage({
      body: purchaseRequest({ attributes: { 'SA-1201': 'Northlight Events GmbH' } }),
    });

    expect(
      await screen.findByRole('heading', { level: 1, name: 'PR for Stand build' }),
    ).toBeInTheDocument();
    const breadcrumb = screen.getByRole('navigation', { name: 'Breadcrumb' });
    expect(within(breadcrumb).getByRole('link', { name: 'Billboard winter wave' })).toHaveAttribute(
      'href',
      '/plans/P-1001/activities/A-1003',
    );
    expect(screen.getByText('Created')).toBeInTheDocument();
    expect(screen.getByText('PURCHASE REQUEST')).toBeInTheDocument();
    expect(detail('MoA approver')).toHaveTextContent('Daniel Okoro');
    expect(detail('Total amount')).toHaveTextContent('$10,000.00');
    expect(detail('Company code')).toHaveTextContent('GB01');
    expect(detail('Cost centre')).toHaveTextContent('GB01000105');
    expect(detail('HX keys')).toHaveTextContent('HX1006');
    expect(detail('Approval workflow')).toHaveTextContent('Purchase request approval (MoA)');
    expect(await screen.findByText('Northlight Events GmbH')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Copy HX key HX1006' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'About Total invoiced' })).toBeInTheDocument();
    expect(
      screen.getByText('The PR form is created when the purchase request is approved.'),
    ).toBeInTheDocument();
    await expectNoAxeViolations();
  });

  it('offers Edit as the one primary action when the person may edit', async () => {
    renderPage({ body: purchaseRequest() });

    expect(await screen.findByRole('link', { name: 'Edit' })).toHaveAttribute(
      'href',
      '/purchase-requests/PR-1002/edit',
    );
  });

  it('lists what submitting still needs; choosing one opens its field on the edit page', async () => {
    const { user, router } = renderPage({
      body: purchaseRequest({
        readiness: {
          ready: false,
          problems: [
            {
              field: 'moaApproverId',
              group: 'purchaseRequest',
              recordId: 'PR-1002',
              message: 'Choose the MoA approver.',
            },
            {
              field: '',
              group: 'lineItem',
              recordId: 'L-1006',
              message:
                "Line item L-1006 'Stand build' has no SWBS code yet (Ready to send to GSAP).",
            },
          ],
        },
      }),
    });

    await user.click(await screen.findByRole('button', { name: /2 fields needed/ }));
    expect(
      screen.getByRole('button', {
        name: "Line item L-1006 'Stand build' has no SWBS code yet (Ready to send to GSAP).",
      }),
    ).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Choose the MoA approver.' }));

    await vi.waitFor(() =>
      expect(`${router.state.location.pathname}${router.state.location.search}`).toBe(
        '/purchase-requests/PR-1002/edit?field=moaApproverId',
      ),
    );
  });

  it('shows the server notice under the header (FR-023)', async () => {
    renderPage({
      body: purchaseRequest({
        status: 'submitted',
        notice: 'Awaiting approval — saving changes will restart approval.',
      }),
    });

    expect(
      await screen.findByText('Awaiting approval — saving changes will restart approval.'),
    ).toBeInTheDocument();
  });

  it('explains why an approved purchase request cannot be deleted (US5 sc.8)', async () => {
    const { user } = renderPage({ body: routedPurchaseRequest() });

    await user.click(await screen.findByRole('button', { name: 'More' }));
    const remove = await screen.findByRole('menuitem', { name: /Delete purchase request/ });
    expect(remove).toHaveAttribute('aria-disabled', 'true');
    expect(remove).toHaveTextContent(
      'Approved purchase requests cannot be deleted — close them instead.',
    );
    expect(screen.getByRole('menuitem', { name: 'Close purchase request' })).toBeInTheDocument();
  });

  it('hides Edit when the person may not change the purchase request', async () => {
    renderPage({
      body: purchaseRequest({
        access: { ...purchaseRequest().access, edit: no(''), delete: no('') },
      }),
    });

    await screen.findByRole('heading', { level: 1, name: 'PR for Stand build' });
    expect(screen.queryByRole('link', { name: 'Edit' })).not.toBeInTheDocument();
  });

  it('shows the simulated GSAP answer and the PO on the line (033 US1 sc.2)', async () => {
    renderPage({ body: orderedPurchaseRequest() });

    expect(await screen.findByText('SIM-0010001001')).toBeInTheDocument();
    expect(detail('Source')).toHaveTextContent('Simulated GSAP');
    expect(screen.getByRole('row', { name: /Stand build \(L-1006\)/ })).toHaveTextContent(
      'SIM-4500001001',
    );
    expect(
      screen.getByRole('button', { name: 'Create GSAP PR and PO (simulated)' }),
    ).toBeDisabled();
  });

  it('keeps the header while the purchase request loads', async () => {
    renderPage(() => pending());

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Purchase request' }),
    ).toBeInTheDocument();
    expect(screen.getByRole('status')).toHaveTextContent('Loading…');
  });

  it('answers a missing or hidden purchase request like a missing one', async () => {
    const message = "This purchase request doesn't exist or you don't have access to it.";
    renderPage({ status: 404, body: apiError('not_found', message) });

    expect(await screen.findByText(message)).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Go to Purchase requests' })).toBeInTheDocument();
  });

  it('shows the refusal of a person whose roles do not open it', async () => {
    const message = "You don't have access to this purchase request.";
    renderPage({ status: 403, body: apiError('forbidden', message) }, [
      reply('GET /api/v1/me', { body: testUser() }),
    ]);

    expect(await screen.findByRole('heading', { level: 1, name: 'No access' })).toBeInTheDocument();
    expect(screen.getByText(message)).toBeInTheDocument();
  });

  it('offers Try again when the purchase request could not be loaded', async () => {
    renderPage({ status: 500, body: apiError('internal_error', 'Something went wrong.') });

    expect(await screen.findByRole('button', { name: 'Try again' })).toBeInTheDocument();
  });
});
