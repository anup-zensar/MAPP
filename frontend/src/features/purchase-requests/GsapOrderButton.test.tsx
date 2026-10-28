import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { expectNoAxeViolations } from '@/testing/axe';
import {
  HIDDEN,
  no,
  orderedPurchaseRequest,
  purchaseRequest,
  routedPurchaseRequest,
} from '@/testing/fixtures/purchase-requests';
import { testUser } from '@/testing/fixtures/users';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { GsapOrderButton } from './GsapOrderButton';

const ACTION = 'Create GSAP PR and PO (simulated)';
const ALREADY =
  'The simulated GSAP already created the PO for this purchase request. Run it again after an approved change of its amounts or lines.';

function renderButton(pr: Parameters<typeof GsapOrderButton>[0]['purchaseRequest']) {
  mockApi([]);
  return renderWithProviders(<GsapOrderButton purchaseRequest={pr} />, { user: testUser() });
}

describe('033 FR-001/FR-006/FR-010: Create GSAP PR and PO (simulated)', () => {
  it('offers the run when the PR waits for GSAP and the person may import GSAP extracts', async () => {
    renderButton(routedPurchaseRequest());

    expect(screen.getByRole('button', { name: ACTION })).toBeEnabled();
    await expectNoAxeViolations();
  });

  it('explains why the run is unavailable once the PO exists (FR-006)', async () => {
    renderButton(orderedPurchaseRequest());

    const button = screen.getByRole('button', { name: ACTION });
    expect(button).toBeDisabled();
    expect(button).toHaveAccessibleDescription(ALREADY);
    await expectNoAxeViolations();
  });

  it('explains a PR that is not approved and routed (FR-010)', () => {
    const reason =
      'The simulated GSAP processes approved purchase requests that were routed to the requisitioner.';
    renderButton(
      purchaseRequest({ access: { ...purchaseRequest().access, gsapOrder: no(reason) } }),
    );

    expect(screen.getByRole('button', { name: ACTION })).toHaveAccessibleDescription(reason);
  });

  it('is not shown to a person without the permission to import GSAP extracts', () => {
    renderButton(
      routedPurchaseRequest({ access: { ...routedPurchaseRequest().access, gsapOrder: HIDDEN } }),
    );

    expect(screen.queryByRole('button', { name: ACTION })).not.toBeInTheDocument();
  });
});
