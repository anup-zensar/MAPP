import { screen, within } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { HttpHandler } from 'msw';
import { expectNoAxeViolations } from '@/testing/axe';
import {
  purchaseRequest,
  ROUTING,
  routedPurchaseRequest,
} from '@/testing/fixtures/purchase-requests';
import { testUser } from '@/testing/fixtures/users';
import { mockApi } from '@/testing/mock-api';
import { findMessages, renderWithProviders } from '@/testing/render';
import { apiError, reply } from '@/testing/replies';
import { RoutingHistory } from './RoutingHistory';

const FORM = 'GET /api/v1/purchase-requests/{purchaseRequestId}/routings/{routingId}/form';
const RESEND = 'POST /api/v1/purchase-requests/{purchaseRequestId}/resend';
const FILE = 'SP&A One PR form – PR-1002 – Original – 27 Oct 2026.xlsx';

let saved: string[];

beforeEach(() => {
  saved = [];
  URL.createObjectURL = () => 'blob:pr-form';
  URL.revokeObjectURL = () => {};
  vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (
    this: HTMLAnchorElement,
  ) {
    saved.push(this.download);
  });
});

afterEach(() => vi.restoreAllMocks());

function renderHistory(pr: ReturnType<typeof purchaseRequest>, answers: HttpHandler[] = []) {
  const api = mockApi(answers);
  return {
    api,
    ...renderWithProviders(<RoutingHistory purchaseRequest={pr} />, { user: testUser() }),
  };
}

const RESENT = {
  ...ROUTING,
  id: 'PRR-1002',
  type: 'resend' as const,
  routedAt: '2026-10-29T08:00:00Z',
  routedBy: { id: 'U-1002', name: 'Priya Raman' },
};

describe('032 FR-031/FR-033/FR-035: routing history', () => {
  it('lists every routing newest first with its type, recipients, who and when', async () => {
    renderHistory(routedPurchaseRequest({ routings: [RESENT, ROUTING] }));

    const list = screen.getByRole('list', { name: 'Routing history' });
    const [first, second] = within(list).getAllByRole('listitem');
    expect(first).toHaveTextContent('Resend · routed to DE_LUBES');
    expect(first).toHaveTextContent('by Priya Raman · 1 line');
    expect(second).toHaveTextContent('Original · routed to DE_LUBES');
    expect(second).toHaveTextContent('27 Oct 2026');
    await expectNoAxeViolations();
  });

  it('downloads the PR form of a routing as XLSX or CSV', async () => {
    const { api, user } = renderHistory(routedPurchaseRequest(), [
      reply(
        FORM,
        () =>
          new Response('PR ID,HX key', {
            headers: {
              'Content-Type': 'text/csv',
              'Content-Disposition': `attachment; filename="form.xlsx"; filename*=UTF-8''${encodeURIComponent(FILE)}`,
            },
          }),
      ),
    ]);

    await user.click(screen.getByRole('button', { name: 'Download PR form PRR-1001 as XLSX' }));
    await user.click(screen.getByRole('button', { name: 'Download PR form PRR-1001 as CSV' }));

    await vi.waitFor(() => expect(saved).toEqual([FILE, FILE]));
    expect(
      api.requests
        .filter((request) => request.method === 'GET')
        .map((request) => `${request.path}${request.search}`),
    ).toEqual([
      '/api/v1/purchase-requests/PR-1002/routings/PRR-1001/form?format=xlsx',
      '/api/v1/purchase-requests/PR-1002/routings/PRR-1001/form?format=csv',
    ]);
  });

  it('says why a download failed', async () => {
    const { user } = renderHistory(routedPurchaseRequest(), [
      reply(FORM, { status: 404, body: apiError('not_found', 'This routing no longer exists.') }),
    ]);

    await user.click(screen.getByRole('button', { name: 'Download PR form PRR-1001 as CSV' }));

    expect(await screen.findByRole('alert')).toHaveTextContent('This routing no longer exists.');
  });

  it('routes the PR form again after confirmation, with the version seen (FR-033)', async () => {
    const { api, user } = renderHistory(routedPurchaseRequest(), [
      reply(RESEND, { body: routedPurchaseRequest({ routings: [RESENT, ROUTING] }) }),
    ]);

    await user.click(screen.getByRole('button', { name: 'Resend PR form' }));
    const dialog = await screen.findByRole('alertdialog', {
      name: "Resend the PR form of 'PR for Stand build'?",
    });
    expect(dialog).toHaveTextContent(
      "SP&A One records a new routing with today's values. Sent dates do not change.",
    );
    await user.click(within(dialog).getByRole('button', { name: 'Resend PR form' }));

    expect(await findMessages()).toHaveTextContent("PR form of 'PR for Stand build' routed again");
    expect(api.bodiesOf('POST /api/v1/purchase-requests/PR-1002/resend')).toEqual([{ version: 3 }]);
  });

  it('says the PR form comes with the approval, before any routing (US3 sc.5)', async () => {
    renderHistory(purchaseRequest());

    expect(
      screen.getByText('The PR form is created when the purchase request is approved.'),
    ).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Resend PR form' })).not.toBeInTheDocument();
    await expectNoAxeViolations();
  });
});
