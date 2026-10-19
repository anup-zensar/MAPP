import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { Sidebar } from '@/app/Sidebar';
import { mockApi } from '@/test/mockApi';
import { renderWithProviders } from '@/test/render';
import { testUser } from '@/test/users';

const counts = (waitingForMe: number) => ({
  body: { waitingForMe, allOpen: 5, approvedThisMonth: 2, declinedThisMonth: 1 },
});

describe('ApprovalsNavBadge (feature 008, FR-027)', () => {
  it('shows the waiting count inside the Approvals link', async () => {
    const api = mockApi({ 'GET /api/approvals/summary': counts(3) });
    renderWithProviders(<Sidebar />, { user: testUser() });

    expect(
      // A busy test machine may need longer than the default 3 s for the first render.
      await screen.findByRole('link', { name: 'Approvals 3 waiting for you' }, { timeout: 10_000 }),
    ).toHaveAttribute('href', '/approvals');
    const asked = api.requests.find((request) => request.path === '/api/approvals/summary');
    expect(asked?.search).toMatch(/monthStart=.+&monthEnd=.+/);
  });

  it('shows nothing at 0', async () => {
    const api = mockApi({ 'GET /api/approvals/summary': counts(0) });
    renderWithProviders(<Sidebar />, { user: testUser() });

    await screen.findByRole('link', { name: 'Approvals' });
    expect(api.requests.some((request) => request.path === '/api/approvals/summary')).toBe(true);
    expect(screen.queryByText('waiting for you')).not.toBeInTheDocument();
  });
});
