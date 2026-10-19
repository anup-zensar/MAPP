import { screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/test/render';
import { testUser } from '@/test/users';
import { PlanCard } from './PlanCard';
import { seedList } from './testData';

const [europe, china, lastYear] = seedList().items;

function renderCard(plan = europe) {
  return renderWithProviders(<PlanCard plan={plan} actions={[]} />, {
    user: testUser(),
    route: '/plans',
    path: '/plans',
  });
}

describe('PlanCard (FR-002)', () => {
  it('shows every fact of the plan', () => {
    renderCard();

    expect(screen.getByRole('link', { name: 'FY2026 Lubricants Europe' })).toHaveAttribute(
      'href',
      '/plans/P-1001',
    );
    expect(screen.getByText('P-1001 · 2026')).toBeInTheDocument();
    expect(screen.getByText('Draft')).toBeInTheDocument();
    expect(screen.getByText('All filled in')).toBeInTheDocument();
    expect(screen.getByText('AOP')).toBeInTheDocument();
    expect(screen.getByText('Owner: Priya Raman')).toBeInTheDocument();
    expect(screen.getByText('Lubricants Europe')).toBeInTheDocument();
    expect(screen.getByText('Business unit: Lubricants')).toBeInTheDocument();
    expect(
      screen.getByText('Planned cost $960,000.00 · Budget $1,200,000.00 · 80 % used'),
    ).toBeInTheDocument();
    expect(screen.getByRole('progressbar', { name: 'Budget used' })).toHaveAttribute(
      'aria-valuenow',
      '80',
    );
    expect(screen.getByText('4 of 7 activities approved')).toBeInTheDocument();
  });

  it('turns the budget bar red above 100 %', () => {
    renderCard(lastYear);

    expect(screen.getByRole('progressbar', { name: 'Budget used' })).toHaveAttribute(
      'aria-valuetext',
      '108 %',
    );
    expect(
      screen.getByText('Planned cost $1,300,000.00 · Budget $1,200,000.00 · 108 % used'),
    ).toBeInTheDocument();
  });

  it('says what is missing: no budget, no owner, no business unit', () => {
    renderCard(china);

    expect(screen.getByText('Planned cost — · No budget set')).toBeInTheDocument();
    expect(screen.queryByRole('progressbar')).not.toBeInTheDocument();
    expect(screen.getByText('No owner yet')).toBeInTheDocument();
    expect(screen.getByText('No business unit yet')).toBeInTheDocument();
  });

  it('opens the edit form on a missing field from the completeness chip (FR-017)', async () => {
    const { user, router } = renderCard(china);

    await user.click(screen.getByRole('button', { name: '1 field needed' }));
    await user.click(await screen.findByRole('button', { name: 'Business unit' }));

    await waitFor(() =>
      expect(`${router.state.location.pathname}${router.state.location.search}`).toBe(
        '/plans/P-1002/edit?focus=attributes.SA-BU',
      ),
    );
  });
});
