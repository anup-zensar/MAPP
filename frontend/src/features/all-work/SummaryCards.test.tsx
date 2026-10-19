import { cleanup, screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/test/render';
import { testUser } from '@/test/users';
import { SummaryCards } from './SummaryCards';
import { gridResult } from './testData';

function card(label: string): HTMLElement {
  const summary = screen.getByRole('region', { name: 'Summary' });
  return within(summary).getByText(label, { selector: 'p' }).parentElement as HTMLElement;
}

describe('SummaryCards (FR-003)', () => {
  it('shows the six cards: counts, and money in compact form with the full value', () => {
    renderWithProviders(<SummaryCards summary={gridResult().summary} />, { user: testUser() });

    expect(card('Campaigns')).toHaveTextContent('3');
    expect(card('Initiatives')).toHaveTextContent('2');
    expect(card('Activities')).toHaveTextContent('3');
    expect(
      within(card('Planned cost')).getByRole('button', { name: '$17K ($17,000.00)' }),
    ).toBeInTheDocument();
    expect(
      within(card('Plan budget')).getByRole('button', { name: '$2M ($2,000,000.00)' }),
    ).toBeInTheDocument();
    expect(
      within(card('Unassigned planned cost')).getByRole('button', { name: '$3K ($3,000.00)' }),
    ).toBeInTheDocument();
  });

  it('explains each card in a tip', async () => {
    const { user } = renderWithProviders(<SummaryCards summary={gridResult().summary} />, {
      user: testUser(),
    });

    await user.click(screen.getByRole('button', { name: 'About Planned cost' }));

    expect(await screen.findByRole('dialog', { name: 'About Planned cost' })).toHaveTextContent(
      "The sum of the matching activities' planned cost. An activity without a planned cost counts as zero.",
    );
    for (const label of ['Campaigns', 'Initiatives', 'Activities', 'Plan budget']) {
      expect(screen.getByRole('button', { name: `About ${label}` })).toBeInTheDocument();
    }
  });

  it('says in red how many activities have line items above their planned cost', () => {
    const { summary } = gridResult();
    renderWithProviders(<SummaryCards summary={summary} />, { user: testUser() });
    expect(
      within(card('Unassigned planned cost')).getByText(
        '1 activity has line items above its planned cost',
      ).className,
    ).toMatch(/problem/);
    cleanup();

    renderWithProviders(<SummaryCards summary={{ ...summary, negativeBalanceCount: 4 }} />, {
      user: testUser(),
    });
    expect(
      screen.getByText('4 activities have line items above their planned cost'),
    ).toBeInTheDocument();
  });

  it('shows "—" for money when nothing matches, and nothing in red', () => {
    renderWithProviders(
      <SummaryCards
        summary={{
          campaigns: 0,
          initiatives: 0,
          activities: 0,
          plannedCost: null,
          planBudget: null,
          unassignedPlannedCost: null,
          negativeBalanceCount: 0,
        }}
      />,
      { user: testUser() },
    );

    expect(card('Planned cost')).toHaveTextContent('—');
    expect(screen.queryByText(/above/)).not.toBeInTheDocument();
  });

  it('shows money in the display currency as the API answers it (FR-017)', () => {
    const summary = {
      ...gridResult().summary,
      plannedCost: { amount: '15640.00', currency: 'EUR' },
    };
    renderWithProviders(<SummaryCards summary={summary} />, { user: testUser() });

    expect(
      within(card('Planned cost')).getByRole('button', { name: '15.64K EUR (15,640.00 EUR)' }),
    ).toBeInTheDocument();
  });

  it('shows grey shapes while the first answer loads', () => {
    renderWithProviders(<SummaryCards summary={null} />, { user: testUser() });

    expect(screen.getAllByText('Loading…')).toHaveLength(12);
  });
});
