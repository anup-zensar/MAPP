import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { mockApi } from '@/testing/mockApi';
import type { MockHandler } from '@/testing/mockApi';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/users';
import type { WaitingItem } from './homeApi';
import { waitingItemOf } from './testData';
import { WaitingCard } from './WaitingCard';

const daysAgo = (days: number) => new Date(Date.now() - (days * 24 + 1) * 3_600_000).toISOString();

const page = (items: WaitingItem[], total = items.length) => ({ body: { items, total } });

function renderCard(answer: MockHandler = page([waitingItemOf({ waitingSince: daysAgo(3) })])) {
  mockApi({ 'GET /api/v1/home/waiting': answer });
  return renderWithProviders(<WaitingCard />, { user: testUser(), path: '/' });
}

function card() {
  return screen.getByRole('region', { name: 'Waiting for my approval' });
}

describe('WaitingCard', () => {
  it('lists each waiting activity with its plan, owner, cost, level and time waiting (FR-013, US3 scenario 5)', async () => {
    renderCard();

    const name = await within(card()).findByRole('link', { name: 'Dealer summit' });
    expect(name).toHaveAttribute('href', '/plans/P-1001/activities/A-1045');
    const row = within(name.closest('li') as HTMLElement);
    expect(row.getByText('A-1045')).toBeInTheDocument();
    expect(
      row.getByText('FY2026 Lubricants Europe · Priya Raman · $48,000.00'),
    ).toBeInTheDocument();
    expect(row.getByText('Level 1 of 1 · waiting 3 days')).toBeInTheDocument();
  });

  it("opens the Approvals page with the activity's preview on Review, in one step (FR-013, SC-003)", async () => {
    const { user, router } = renderCard();

    const review = await within(card()).findByRole('link', { name: 'Review Dealer summit' });
    expect(review).toHaveTextContent('Review');
    await user.click(review);

    expect(router.state.location.pathname).toBe('/approvals');
    expect(router.state.location.search).toBe('?tab=waiting&activity=A-1045');
  });

  it('leaves out an owner that is not set (FR-013)', async () => {
    renderCard(page([waitingItemOf({ owner: null, plannedCost: null })]));

    const name = await within(card()).findByRole('link', { name: 'Dealer summit' });
    expect(
      within(name.closest('li') as HTMLElement).getByText('FY2026 Lubricants Europe · —'),
    ).toBeInTheDocument();
  });

  it('offers "View all <n>" on the Waiting for me tab when more than 5 wait (FR-013)', async () => {
    const items = Array.from({ length: 5 }, (_, index) =>
      waitingItemOf({ id: `A-30${index}`, name: `Activity ${index + 1}` }),
    );
    renderCard(page(items, 7));

    expect(await within(card()).findByRole('link', { name: 'View all (7)' })).toHaveAttribute(
      'href',
      '/approvals?tab=waiting',
    );
  });

  it('has no "View all" when every waiting activity is shown', async () => {
    renderCard();

    await within(card()).findByRole('link', { name: 'Dealer summit' });
    expect(within(card()).queryByRole('link', { name: /View all/ })).not.toBeInTheDocument();
  });

  it('says so in one sentence when nothing waits (US3 scenario 6)', async () => {
    renderCard(page([]));

    expect(
      await within(card()).findByText('Nothing is waiting for your approval right now.'),
    ).toBeInTheDocument();
  });

  it('shows a placeholder while loading, and Try again when it fails (FR-022)', async () => {
    let calls = 0;
    const { user } = renderCard(() => (++calls === 1 ? { status: 500 } : page([waitingItemOf()])));

    expect(within(card()).getByRole('status')).toHaveTextContent('Loading…');
    expect(await within(card()).findByRole('alert')).toHaveTextContent(
      "This section couldn't load",
    );
    await user.click(within(card()).getByRole('button', { name: 'Try again' }));

    expect(
      await within(card()).findByRole('link', { name: 'Review Dealer summit' }),
    ).toBeInTheDocument();
  });
});
