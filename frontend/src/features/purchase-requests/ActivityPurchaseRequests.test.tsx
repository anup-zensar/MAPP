import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { expectNoAxeViolations } from '@/testing/axe';
import { activitySection } from '@/testing/fixtures/purchase-requests';
import { testUser } from '@/testing/fixtures/users';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { apiError, pending, reply } from '@/testing/replies';
import type { Answer } from '@/testing/replies';
import { ActivityPurchaseRequests } from './ActivityPurchaseRequests';

const SECTION = 'GET /api/v1/activities/{activityId}/purchase-requests';

function renderSection(answer: Answer<typeof SECTION>) {
  mockApi([
    reply(SECTION, answer, { activityId: 'A-1003' }),
    reply('GET /api/v1/me', { body: testUser() }),
  ]);
  return renderWithProviders(<ActivityPurchaseRequests activityId="A-1003" />, {
    route: '/plans/P-1001/activities/A-1003',
    user: testUser(),
  });
}

describe('032 FR-018: activity Purchase requests section', () => {
  it("shows the activity's total in the plan currency and a row per line item (US1 scenario 5)", async () => {
    renderSection({ body: activitySection() });

    expect(await screen.findByText('6,521.74 USD', { selector: 'dd' })).toBeInTheDocument();
    const row = screen.getByRole('row', { name: /Stand build/ });
    expect(within(row).getByRole('link', { name: 'Stand build' })).toHaveAttribute(
      'href',
      '/line-items/L-1006#purchase-requests',
    );
    expect(row).toHaveTextContent('6,000.00 EUR');
    expect(row).toHaveTextContent('3,200.00 EUR');
    expect(row).toHaveTextContent('1');
    await expectNoAxeViolations();
  });

  it('says when no line item has a purchase request', async () => {
    renderSection({
      body: activitySection({ requested: { amount: '0.00', currency: 'USD' }, lineItems: [] }),
    });

    expect(
      await screen.findByText("No purchase requests on this activity's line items yet."),
    ).toBeInTheDocument();
  });

  it('shows a placeholder while it loads', async () => {
    renderSection(() => pending());

    expect(await screen.findByText('Loading…')).toBeInTheDocument();
  });

  it('shows a failure with Try again', async () => {
    renderSection({ status: 500, body: apiError('internal_error', 'Something went wrong.') });

    expect(await screen.findByRole('button', { name: 'Try again' })).toBeInTheDocument();
  });

  it('says in place when the person may not see them', async () => {
    renderSection({ status: 404, body: apiError('not_found', 'Activity not found.') });

    expect(
      await screen.findByText("You can't see the purchase requests here."),
    ).toBeInTheDocument();
  });
});
