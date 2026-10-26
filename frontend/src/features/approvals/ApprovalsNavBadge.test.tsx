import { act, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { Sidebar } from '@/app/Sidebar';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';

const counts = (waitingForMe: number) => ({
  body: { waitingForMe, allOpen: 5, approvedThisMonth: 2, declinedThisMonth: 1 },
});

describe('ApprovalsNavBadge (feature 008, FR-027)', () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it('shows the waiting count inside the Approvals link', async () => {
    const api = mockApi({ 'GET /api/v1/approvals/summary': counts(3) });
    renderWithProviders(<Sidebar />, { user: testUser() });

    expect(
      // A busy test machine may need longer than the default 3 s for the first render.
      await screen.findByRole('link', { name: 'Approvals 3 waiting for you' }, { timeout: 10_000 }),
    ).toHaveAttribute('href', '/approvals');
    const asked = api.requests.find((request) => request.path === '/api/v1/approvals/summary');
    expect(asked?.search).toMatch(/monthStart=.+&monthEnd=.+/);
  });

  it('shows nothing at 0', async () => {
    const api = mockApi({ 'GET /api/v1/approvals/summary': counts(0) });
    renderWithProviders(<Sidebar />, { user: testUser() });

    await screen.findByRole('link', { name: 'Approvals' });
    expect(api.requests.some((request) => request.path === '/api/v1/approvals/summary')).toBe(true);
    expect(screen.queryByText('waiting for you')).not.toBeInTheDocument();
  });

  it('shows a new count within a minute, without reloading the page (FR-036)', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    let calls = 0;
    mockApi({
      'GET /api/v1/approvals/summary': () => {
        calls += 1;
        return counts(calls === 1 ? 0 : 2);
      },
    });
    renderWithProviders(<Sidebar />, { user: testUser() });
    await screen.findByRole('link', { name: 'Approvals' }, { timeout: 10_000 });
    await waitFor(() => expect(calls).toBe(1));

    await act(async () => {
      await vi.advanceTimersByTimeAsync(30_000);
    });

    expect(
      await screen.findByRole('link', { name: 'Approvals 2 waiting for you' }),
    ).toBeInTheDocument();
  });
});
