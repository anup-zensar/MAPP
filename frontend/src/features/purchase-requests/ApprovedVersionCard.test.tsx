import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { expectNoAxeViolations } from '@/testing/axe';
import {
  DANIEL_REF,
  money,
  PRIYA_REF,
  prLine,
  routedPurchaseRequest,
} from '@/testing/fixtures/purchase-requests';
import { testUser } from '@/testing/fixtures/users';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { ApprovedVersionCard } from './ApprovedVersionCard';

const IN_REVIEW = routedPurchaseRequest({
  status: 'inReview',
  changePending: true,
  lines: [prLine({ amount: money('8000.00'), changed: true })],
  approvedVersion: {
    name: 'PR for Stand build',
    description: '',
    owner: PRIYA_REF,
    moaApprover: DANIEL_REF,
    attributes: {},
    amountDisplay: money('6521.74', 'USD'),
    approvedAt: '2026-10-27T10:15:00Z',
    lines: [
      {
        id: 'PRL-1002',
        lineItemId: 'L-1006',
        hxKey: 'HX1006',
        amount: money('6000.00'),
        deliveryDate: '2026-11-30',
        overallLimit: money('6000.00'),
        description: '',
      },
    ],
  },
});

describe('032 FR-036/FR-037: approved version while a change is in review', () => {
  it('keeps the approved values in view until the change is decided (US6 sc.1)', async () => {
    mockApi([]);
    renderWithProviders(<ApprovedVersionCard purchaseRequest={IN_REVIEW} />, { user: testUser() });

    const card = screen.getByRole('region', { name: 'Approved version' });
    expect(card).toHaveTextContent('The values approved on 27 Oct 2026');
    expect(within(card).getByRole('row', { name: /L-1006 · HX1006/ })).toHaveTextContent(
      '6,000.00 EUR',
    );
    expect(card).toHaveTextContent('6,521.74 USD');
    expect(card).toHaveTextContent('Daniel Okoro');
    await expectNoAxeViolations();
  });

  it('shows nothing when no change is in review', () => {
    mockApi([]);
    renderWithProviders(
      <ApprovedVersionCard purchaseRequest={{ ...IN_REVIEW, changePending: false }} />,
      { user: testUser() },
    );

    expect(screen.queryByRole('region', { name: 'Approved version' })).not.toBeInTheDocument();
  });
});
