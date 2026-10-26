import { screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { mockApi } from '@/testing/mockApi';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { DelegationBanner } from './DelegationBanner';
import { CARLOS, DANIEL, delegationOf, NO_DELEGATIONS, PRIYA } from '@/testing/fixtures/approvals';

const LIST = 'GET /api/v1/approval-delegations';

describe('DelegationBanner (feature 017, US4)', () => {
  it('names each current delegation that concerns the person, one line each (FR-024)', async () => {
    mockApi({
      [LIST]: {
        body: {
          items: [
            delegationOf({ person: PRIYA, delegate: CARLOS, state: 'current' }),
            delegationOf({ id: 'DLG-1002', person: DANIEL, delegate: PRIYA, state: 'current' }),
            delegationOf({ id: 'DLG-1003', person: PRIYA, delegate: DANIEL, state: 'planned' }),
          ],
          total: 3,
          canManageOthers: false,
        },
      },
    });
    renderWithProviders(<DelegationBanner />, { user: testUser() });

    const banner = await screen.findByRole('status');
    expect(banner).toHaveTextContent('Carlos Mendes approves for you until 14 Nov 2026.');
    expect(banner).toHaveTextContent('You are approving for Daniel Okoro until 14 Nov 2026.');
    expect(banner).not.toHaveTextContent('Daniel Okoro approves for you');
  });

  it('shows nothing without a current delegation', async () => {
    const api = mockApi({ [LIST]: NO_DELEGATIONS });
    renderWithProviders(<DelegationBanner />, { user: testUser() });

    await vi.waitFor(() => expect(api.requests).toHaveLength(1));
    expect(screen.queryByRole('status')).not.toBeInTheDocument();
  });
});
