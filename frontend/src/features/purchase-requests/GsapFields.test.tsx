import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { expectNoAxeViolations } from '@/testing/axe';
import {
  money,
  orderedPurchaseRequest,
  prLine,
  purchaseRequest,
} from '@/testing/fixtures/purchase-requests';
import { testUser } from '@/testing/fixtures/users';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { GsapFields } from './GsapFields';
import { PurchaseRequestLinesTable } from './PurchaseRequestLinesTable';

function detail(label: string): HTMLElement {
  const term = screen.getByText(label, { selector: 'dt' });
  const value = term.nextElementSibling;
  if (!(value instanceof HTMLElement)) {
    throw new Error(`No value for ${label}`);
  }
  return value;
}

describe('033 FR-004/FR-011: GSAP answer labelled Simulated GSAP', () => {
  it('shows the GSAP PR number and date with the source Simulated GSAP, read-only', async () => {
    mockApi([]);
    const { user } = renderWithProviders(
      <GsapFields purchaseRequest={orderedPurchaseRequest()} />,
      {
        user: testUser(),
      },
    );

    expect(detail('GSAP PR number')).toHaveTextContent('SIM-0010001001');
    expect(detail('GSAP PR date')).toHaveTextContent('27 Oct 2026');
    expect(detail('Source')).toHaveTextContent('Simulated GSAP');
    expect(screen.queryByRole('textbox')).not.toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'About GSAP PR number' }));
    expect(await screen.findByText('Filled by GSAP')).toBeInTheDocument();
    await expectNoAxeViolations();
  });

  it('032 FR-009: says the fields stay empty until GSAP answers', () => {
    mockApi([]);
    renderWithProviders(<GsapFields purchaseRequest={purchaseRequest()} />, { user: testUser() });

    expect(screen.getByText('Empty until GSAP answers this purchase request.')).toBeInTheDocument();
    expect(detail('GSAP PR number')).toHaveTextContent('—');
  });

  it('shows each line PO fields and says the simulated GSAP filled them', async () => {
    mockApi([]);
    renderWithProviders(<PurchaseRequestLinesTable purchaseRequest={orderedPurchaseRequest()} />, {
      user: testUser(),
    });

    const lines = screen.getByRole('region', { name: 'Lines' });
    expect(lines).toHaveTextContent(
      'PO number, line, status, date and value: filled by Simulated GSAP',
    );
    const row = within(lines).getByRole('row', { name: /Stand build \(L-1006\)/ });
    expect(row).toHaveTextContent('SIM-4500001001');
    expect(row).toHaveTextContent('Released');
    expect(row).toHaveTextContent('9,200.00 EUR');
    expect(row).toHaveTextContent('SIM-GB01-0001');
    expect(within(lines).getByRole('button', { name: 'About PO number' })).toBeInTheDocument();
    await expectNoAxeViolations();
  });

  it('032 FR-036: marks the lines a change in review touches', () => {
    mockApi([]);
    const pr = purchaseRequest({
      status: 'inReview',
      changePending: true,
      lines: [prLine({ amount: money('9500.00'), changed: true })],
    });
    renderWithProviders(<PurchaseRequestLinesTable purchaseRequest={pr} />, { user: testUser() });

    expect(screen.getByRole('row', { name: /Stand build/ })).toHaveTextContent('Changed');
  });
});
