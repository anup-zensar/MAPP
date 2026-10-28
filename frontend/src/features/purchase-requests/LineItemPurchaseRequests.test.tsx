import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { expectNoAxeViolations } from '@/testing/axe';
import { HIDDEN, lineItemSection, money, no } from '@/testing/fixtures/purchase-requests';
import { testUser } from '@/testing/fixtures/users';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { apiError, pending, reply } from '@/testing/replies';
import type { Answer } from '@/testing/replies';
import { LineItemPurchaseRequests } from './LineItemPurchaseRequests';

const SECTION = 'GET /api/v1/line-items/{lineItemId}/purchase-requests';
const SWITCHED_OFF = 'Purchase requests are switched off for your organisation.';

function renderSection(answer: Answer<typeof SECTION>) {
  mockApi([
    reply(SECTION, answer, { lineItemId: 'L-1006' }),
    reply('GET /api/v1/me', { body: testUser() }),
  ]);
  return renderWithProviders(<LineItemPurchaseRequests lineItemId="L-1006" />, {
    route: '/line-items/L-1006',
    user: testUser(),
  });
}

function section() {
  return screen.getByRole('region', { name: 'Purchase requests' });
}

describe('032 FR-001/FR-018: line item Purchase requests section', () => {
  it('shows what is requested and available as the server sends them, and its purchase requests (US1 scenario 5)', async () => {
    renderSection({ body: lineItemSection() });

    expect(await screen.findByText('6,000.00 EUR', { selector: 'dd' })).toBeInTheDocument();
    expect(within(section()).getByText('Total requested')).toBeInTheDocument();
    expect(within(section()).getByText('3,200.00 EUR', { selector: 'dd' })).toBeInTheDocument();
    const row = within(section()).getByRole('row', { name: /PR for Stand build/ });
    expect(within(row).getByRole('link', { name: 'PR for Stand build' })).toHaveAttribute(
      'href',
      '/purchase-requests/PR-1002',
    );
    expect(row).toHaveTextContent('Created');
    expect(row).toHaveTextContent('Priya Raman');
    expect(screen.getByRole('link', { name: 'New purchase request' })).toHaveAttribute(
      'href',
      '/purchase-requests/new?lineItemId=L-1006',
    );
    await expectNoAxeViolations();
  });

  it('shows a line in another currency with its amount in the line item currency', async () => {
    const base = lineItemSection();
    renderSection({
      body: {
        ...base,
        purchaseRequests: base.purchaseRequests.map((row) => ({
          ...row,
          amount: money('3700.00', 'USD'),
          amountInLineItemCurrency: money('3404.00'),
        })),
      },
    });

    const row = await screen.findByRole('row', { name: /PR for Stand build/ });
    expect(row).toHaveTextContent('$3,700.00');
    expect(row).toHaveTextContent("3,404.00 EUR in the line item's currency");
  });

  it('says when rates moved the requests above the planned cost (edge case)', async () => {
    renderSection({ body: lineItemSection({ availableToRequest: money('-120.00') }) });

    expect(
      await screen.findByText('Requested above planned cost by 120.00 EUR'),
    ).toBeInTheDocument();
  });

  it('explains why New purchase request is not offered, or leaves it out (FR-049, US1 scenario 8)', async () => {
    renderSection({
      body: lineItemSection({ create: no('Nothing left to request'), purchaseRequests: [] }),
    });

    expect(await screen.findByText('Nothing left to request')).toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'New purchase request' })).not.toBeInTheDocument();
    expect(screen.getByText('No purchase requests on this line item yet.')).toBeInTheDocument();
  });

  it('shows no create action to a person who may not raise purchase requests', async () => {
    renderSection({ body: lineItemSection({ create: HIDDEN }) });

    expect(await screen.findByRole('row', { name: /PR for Stand build/ })).toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'New purchase request' })).not.toBeInTheDocument();
  });

  it('shows the switched-off notice (FR-004, US1 scenario 7)', async () => {
    renderSection({ body: lineItemSection({ create: HIDDEN, notice: SWITCHED_OFF }) });

    expect(await screen.findByText(SWITCHED_OFF)).toBeInTheDocument();
  });

  it('shows a placeholder while it loads', async () => {
    renderSection(() => pending());

    expect(await within(section()).findByText('Loading…')).toBeInTheDocument();
  });

  it('shows a failure with Try again', async () => {
    renderSection({ status: 500, body: apiError('internal_error', 'Something went wrong.') });

    expect(await screen.findByRole('button', { name: 'Try again' })).toBeInTheDocument();
  });

  it('says in place when the person may not see them', async () => {
    renderSection({ status: 403, body: apiError('forbidden', "That isn't part of your role.") });

    expect(
      await screen.findByText("You can't see the purchase requests here."),
    ).toBeInTheDocument();
  });
});
