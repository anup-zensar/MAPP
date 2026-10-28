import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { components } from '@/api/generated/schema';
import { expectNoAxeViolations } from '@/testing/axe';
import { attribute, option } from '@/testing/fixtures/attribute-forms';
import { PEOPLE } from '@/testing/fixtures/people';
import { BRIGHTSIDE_ROW, listPage, prRow } from '@/testing/fixtures/purchase-requests';
import { testUser } from '@/testing/fixtures/users';
import { mockApi } from '@/testing/mock-api';
import type { MockedApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { apiError, pending, reply } from '@/testing/replies';
import type { Answer } from '@/testing/replies';
import { PurchaseRequestsPage } from './PurchaseRequestsPage';

const LIST = 'GET /api/v1/purchase-requests';
const SWITCHED_OFF = 'Purchase requests are switched off for your organisation.';

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
          attribute({
            id: 'SA-1202',
            label: 'PR type',
            type: 'multipleChoice',
            options: [option('SO-1201', 'Standard PR/PO'), option('SO-1202', 'Manual PR/PO')],
          }),
        ],
      },
    ],
  };
}

function renderPage(answer: Answer<typeof LIST>, route = '/purchase-requests') {
  const api = mockApi([
    reply(LIST, answer),
    reply('GET /api/v1/shell-attribute-forms/{level}', { body: prForm() }),
    reply('GET /api/v1/people', { body: PEOPLE }),
    reply('GET /api/v1/me', { body: testUser() }),
  ]);
  return {
    api,
    ...renderWithProviders(<PurchaseRequestsPage />, {
      route,
      path: '/purchase-requests',
      user: testUser(),
    }),
  };
}

function lastQuery(api: MockedApi): URLSearchParams {
  const requests = api.requests.filter((request) => request.path === '/api/v1/purchase-requests');
  return new URLSearchParams(requests.at(-1)?.search ?? '');
}

describe('032 FR-044–FR-048: Purchase requests page', () => {
  it('groups the purchase requests by vendor with the totals the server counts (US5 scenario 1)', async () => {
    const { user } = renderPage({ body: listPage() });

    const table = await screen.findByRole('table', { name: 'Purchase requests by vendor' });
    const [brightside, northlight] = within(table).getAllByRole('rowgroup').slice(1);
    expect(brightside).toHaveTextContent('Brightside Media BV · 1 purchase request');
    expect(brightside).toHaveTextContent('$30,000.00');
    expect(northlight).toHaveTextContent('Northlight Events GmbH · 1 purchase request');
    expect(
      within(northlight ?? table).getByRole('link', { name: 'PR for Stand build' }),
    ).toHaveAttribute('href', '/purchase-requests/PR-1002');
    expect(within(northlight ?? table).getByText('Waiting for GSAP')).toBeInTheDocument();
    expect(screen.getByText('2 purchase requests')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Show the lines of PR for Stand build' }));
    const lines = screen.getByRole('table', { name: '1 line of PR for Stand build' });
    expect(within(lines).getByRole('link', { name: 'Stand build (L-1006)' })).toHaveAttribute(
      'href',
      '/line-items/L-1006',
    );
    expect(lines).toHaveTextContent('HX1006');
    expect(lines).toHaveTextContent('9,200.00 EUR');
    expect(
      screen.getByRole('button', { name: 'Hide the lines of PR for Stand build' }),
    ).toHaveAttribute('aria-expanded', 'true');
    await expectNoAxeViolations();
  });

  it('searches after a pause, keeping the search in the address (US5 scenario 2)', async () => {
    const { api, router, user } = renderPage({ body: listPage() });
    await screen.findByRole('table');

    await user.type(screen.getByRole('searchbox', { name: 'Search purchase requests' }), 'HX1006');

    await waitFor(() => expect(router.state.location.search).toBe('?q=HX1006'));
    await waitFor(() => expect(lastQuery(api).get('q')).toBe('HX1006'));
    expect(router.state.historyAction).toBe('REPLACE');
  });

  it('filters by status and PR type, each chip removable on its own (US5 scenario 3)', async () => {
    const { api, router, user } = renderPage(
      { body: listPage() },
      '/purchase-requests?status=approved&type=SO-1201&typeNotCompleted=true',
    );

    expect(
      await screen.findByRole('button', { name: 'Remove filter: Status: Approved' }),
    ).toBeVisible();
    expect(
      await screen.findByRole('button', { name: 'Remove filter: PR type: Standard PR/PO' }),
    ).toBeVisible();
    const query = lastQuery(api);
    expect(query.getAll('status')).toEqual(['approved']);
    expect(query.getAll('type')).toEqual(['SO-1201']);
    expect(query.get('typeNotCompleted')).toBe('true');

    await user.click(screen.getByRole('button', { name: 'Remove filter: PR type: Not completed' }));
    await waitFor(() => expect(router.state.location.search).toBe('?status=approved&type=SO-1201'));
    await user.click(screen.getByRole('button', { name: 'Remove filter: Status: Approved' }));
    await waitFor(() => expect(router.state.location.search).toBe('?type=SO-1201'));
  });

  it('chooses a status from the Status menu', async () => {
    const { router, user } = renderPage({ body: listPage() });
    await screen.findByRole('table');

    await user.click(screen.getByRole('button', { name: 'Status: Any status' }));
    await user.click(await screen.findByRole('menuitemcheckbox', { name: 'Declined' }));

    await waitFor(() => expect(router.state.location.search).toBe('?status=declined'));
  });

  it('shows the Waiting for GSAP view, and keeps the address when coming Back (US5 scenario 4)', async () => {
    const { api, router, user } = renderPage({ body: listPage({ items: [prRow()], total: 1 }) });
    await screen.findByRole('table');

    await user.click(screen.getByRole('button', { name: 'Waiting for GSAP' }));
    await waitFor(() => expect(router.state.location.search).toBe('?waiting=true'));
    await waitFor(() => expect(lastQuery(api).get('waitingForGsap')).toBe('true'));

    await user.click(await screen.findByRole('link', { name: 'PR for Stand build' }));
    expect(await screen.findByText('Another page')).toBeInTheDocument();
    await router.navigate(-1);
    expect(await screen.findByRole('button', { name: 'Waiting for GSAP' })).toHaveAttribute(
      'aria-pressed',
      'true',
    );
    expect(router.state.location.search).toBe('?waiting=true');
  });

  it('sorts and pages through the address (FR-046)', async () => {
    const { api, router, user } = renderPage({ body: listPage({ total: 120 }) });
    await screen.findByRole('table');

    await user.selectOptions(
      screen.getByRole('combobox', { name: 'Sort' }),
      'Amount, largest first',
    );
    await waitFor(() => expect(router.state.location.search).toBe('?sort=-amount'));
    await waitFor(() => expect(lastQuery(api).get('sort')).toBe('-amount'));

    expect(await screen.findByText('Page 1 of 3')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Next' }));
    await waitFor(() => expect(router.state.location.search).toBe('?sort=-amount&page=2'));
    expect(lastQuery(api).get('pageSize')).toBe('50');
  });

  it('shows a placeholder while the purchase requests load', async () => {
    renderPage(() => pending());

    expect(await screen.findByText('Loading…')).toBeInTheDocument();
    expect(screen.queryByRole('table')).not.toBeInTheDocument();
  });

  it('explains how purchase requests are raised when there are none (FR-048)', async () => {
    renderPage({ body: listPage({ items: [], total: 0, vendors: [] }) });

    expect(
      await screen.findByText(
        "No purchase requests yet — purchase requests are raised from a line item's Purchase requests section.",
      ),
    ).toBeInTheDocument();
    await expectNoAxeViolations();
  });

  it('offers Clear filters when nothing matches', async () => {
    const { router, user } = renderPage(
      { body: listPage({ items: [], total: 0, vendors: [] }) },
      '/purchase-requests?vendor=Brightside%20Media%20BV&waiting=true',
    );

    expect(await screen.findByText('Nothing matches these filters')).toBeInTheDocument();
    await user.click(
      screen.getAllByRole('button', { name: 'Clear filters' }).at(-1) ?? document.body,
    );
    await waitFor(() => expect(router.state.location.search).toBe('?waiting=true'));
    expect(await screen.findByText('No purchase request is waiting for GSAP.')).toBeInTheDocument();
  });

  it('shows a failure with Try again', async () => {
    renderPage({ status: 500, body: apiError('internal_error', 'Something went wrong.') });

    expect(await screen.findByRole('button', { name: 'Try again' })).toBeInTheDocument();
  });

  it('shows the no-access page for a refused list (FR-049)', async () => {
    renderPage({
      status: 403,
      body: apiError('forbidden', "Seeing the Purchase requests page isn't part of your role."),
    });

    expect(await screen.findByRole('heading', { name: 'No access' })).toBeInTheDocument();
    expect(
      screen.getByText("Seeing the Purchase requests page isn't part of your role."),
    ).toBeInTheDocument();
  });

  it('says when purchase requests are switched off (FR-004)', async () => {
    renderPage({ status: 409, body: apiError('not_allowed', SWITCHED_OFF) });

    expect(await screen.findByText(SWITCHED_OFF)).toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'New purchase request' })).not.toBeInTheDocument();
  });

  it('lists a purchase request without a vendor under Not completed', async () => {
    renderPage({
      body: listPage({
        items: [{ ...BRIGHTSIDE_ROW, vendor: '' }],
        total: 1,
        vendors: [
          {
            vendor: '',
            label: '',
            purchaseRequests: 1,
            amountDisplay: { amount: '30000.00', currency: 'USD' },
            invoicedDisplay: { amount: '0.00', currency: 'USD' },
          },
        ],
      }),
    });

    expect(await screen.findByText('Not completed')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'PR for Media booking' })).toBeInTheDocument();
  });
});
