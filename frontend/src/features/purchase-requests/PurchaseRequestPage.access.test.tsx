import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { components } from '@/api/generated/schema';
import { expectNoAxeViolations } from '@/testing/axe';
import { HISTORY_ANSWERS } from '@/testing/fixtures/activity-log';
import {
  HIDDEN,
  approvalSection,
  prLine,
  purchaseRequestApproval,
  routedPurchaseRequest,
} from '@/testing/fixtures/purchase-requests';
import { testUser } from '@/testing/fixtures/users';
import { VIEW_ONLY_NOTE, VIEWER, refused } from '@/testing/fixtures/viewer';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { apiError, reply } from '@/testing/replies';
import type { Answer } from '@/testing/replies';
import type { CurrentUser } from '@/session/useSession';
import { PurchaseRequestPage } from './PurchaseRequestPage';

const GET = 'GET /api/v1/purchase-requests/{purchaseRequestId}';
const CHANGE = refused('Changing purchase requests');

const VIEW_ONLY_ACCESS: components['schemas']['PurchaseRequestAccess'] = {
  edit: CHANGE,
  delete: refused('Deleting purchase requests'),
  submit: CHANGE,
  resubmit: HIDDEN,
  submitChanges: HIDDEN,
  recall: HIDDEN,
  close: CHANGE,
  resend: CHANGE,
  addLine: CHANGE,
  gsapOrder: refused('Importing financial data'),
};

function renderPage(answer: Answer<typeof GET>, user: CurrentUser) {
  mockApi([
    ...HISTORY_ANSWERS,
    reply('GET /api/v1/shell-attribute-forms/{level}', {
      body: {
        level: 'purchaseRequest',
        businessUnitAttributeId: null,
        businessUnitOptions: [],
        sections: [],
      },
    }),
    reply('GET /api/v1/purchase-requests/{purchaseRequestId}/approval', {
      body: purchaseRequestApproval({
        sections: [
          approvalSection({
            actions: {
              submit: HIDDEN,
              resubmit: HIDDEN,
              approve: HIDDEN,
              decline: HIDDEN,
              recall: HIDDEN,
            },
          }),
        ],
      }),
    }),
    reply(GET, answer, { purchaseRequestId: 'PR-1002' }),
  ]);
  return renderWithProviders(<PurchaseRequestPage />, {
    route: '/purchase-requests/PR-1002',
    path: '/purchase-requests/:purchaseRequestId',
    user,
  });
}

function disabledControls(): HTMLElement[] {
  return [...screen.queryAllByRole('button'), ...screen.queryAllByRole('menuitem')].filter(
    (control) =>
      control.hasAttribute('disabled') || control.getAttribute('aria-disabled') === 'true',
  );
}

describe('038 FR-025 (M-7): a person who may change nothing on a purchase request', () => {
  it('sees one View only note and no disabled change action', async () => {
    const pr = routedPurchaseRequest({
      access: VIEW_ONLY_ACCESS,
      lines: [prLine({ access: { remove: CHANGE } })],
    });
    const { user } = renderPage({ body: pr }, VIEWER);

    await screen.findByRole('heading', { level: 1, name: 'PR for Stand build' });
    expect(
      screen.getAllByRole('note').filter((n) => n.textContent === VIEW_ONLY_NOTE),
    ).toHaveLength(1);
    expect(disabledControls()).toEqual([]);
    await user.click(screen.getByRole('button', { name: 'More' }));
    await screen.findByRole('menuitem', { name: 'View history' });
    expect(disabledControls()).toEqual([]);
    await user.keyboard('{Escape}');
    await expectNoAxeViolations();
  });
});

describe('038 FR-050 (L-24): breadcrumb of a purchase request', () => {
  it('starts with Purchase requests, then where the request was raised', async () => {
    renderPage({ body: routedPurchaseRequest() }, testUser());

    await screen.findByRole('heading', { level: 1, name: 'PR for Stand build' });
    const links = within(screen.getByRole('navigation', { name: 'Breadcrumb' })).getAllByRole(
      'link',
    );
    expect(links[0]).toHaveTextContent('Purchase requests');
    expect(links[0]).toHaveAttribute('href', '/purchase-requests');
  });

  it('starts with Purchase requests when the request is missing', async () => {
    const message = "This purchase request doesn't exist or you don't have access to it.";
    renderPage({ status: 404, body: apiError('not_found', message) }, testUser());

    await screen.findByText(message);
    const breadcrumb = screen.getByRole('navigation', { name: 'Breadcrumb' });
    expect(within(breadcrumb).getByRole('link', { name: 'Purchase requests' })).toHaveAttribute(
      'href',
      '/purchase-requests',
    );
  });

  it('starts with Plans for a person who may not see Purchase requests', async () => {
    const message = "This purchase request doesn't exist or you don't have access to it.";
    renderPage({ status: 404, body: apiError('not_found', message) }, VIEWER);

    await screen.findByText(message);
    const breadcrumb = screen.getByRole('navigation', { name: 'Breadcrumb' });
    expect(within(breadcrumb).getByRole('link', { name: 'Plans' })).toHaveAttribute(
      'href',
      '/plans',
    );
  });
});
