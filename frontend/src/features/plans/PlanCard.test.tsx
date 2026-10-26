import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/users';
import { PlanCard } from './PlanCard';
import { seedList } from './testData';

const [europe, china, lastYear] = seedList().items;

function renderCard(plan = europe, pinned = false, onPinToggle = () => {}) {
  return renderWithProviders(
    <PlanCard plan={plan} actions={[]} pinned={pinned} onPinToggle={onPinToggle} />,
    {
      user: testUser(),
      route: '/plans',
      path: '/plans',
    },
  );
}

/** The value beside a label of the card's money list ("Planned cost", "Plan budget"). */
function moneyLine(label: string) {
  return screen.getByText(label).nextElementSibling as HTMLElement;
}

describe('PlanCard (FR-002)', () => {
  it('shows every fact of the plan', () => {
    renderCard();

    expect(screen.getByRole('link', { name: 'FY2026 Lubricants Europe' })).toHaveAttribute(
      'href',
      '/plans/P-1001',
    );
    expect(screen.getByText('P-1001 · 2026 · AOP')).toBeInTheDocument();
    // The plan type (016 FR-032): Operational until the API sends another.
    expect(screen.getByText('Operational')).toBeInTheDocument();
    expect(screen.getByText('Draft')).toBeInTheDocument();
    expect(screen.getByText('All filled in')).toBeInTheDocument();
    expect(screen.getByText('Priya Raman')).toBeInTheDocument();
    expect(screen.getByText('Lubricants Europe')).toBeInTheDocument();
    expect(screen.getByText('Business unit: Lubricants')).toBeInTheDocument();
    expect(moneyLine('Planned cost')).toHaveTextContent('$960,000.00');
    expect(moneyLine('Plan budget')).toHaveTextContent('$1,200,000.00 80% used');
    expect(screen.getByRole('progressbar', { name: 'Budget used' })).toHaveAttribute(
      'aria-valuenow',
      '80',
    );
    expect(screen.getByText('4 of 7 activities approved')).toBeInTheDocument();
  });

  it('turns the budget bar and its words red above 100 %', () => {
    renderCard(lastYear);

    expect(screen.getByRole('progressbar', { name: 'Budget used' })).toHaveAttribute(
      'aria-valuetext',
      '108 %',
    );
    expect(moneyLine('Planned cost')).toHaveTextContent('$1,300,000.00');
    expect(within(moneyLine('Plan budget')).getByText('108% used').className).toMatch(/problem/);
  });

  it('says what is missing: no budget, no owner, no business unit', () => {
    renderCard(china);

    expect(screen.getByText('No plan budget set')).toBeInTheDocument();
    expect(moneyLine('Plan budget')).toHaveTextContent('—');
    expect(screen.queryByRole('progressbar')).not.toBeInTheDocument();
    expect(screen.getByText('No owner yet')).toBeInTheDocument();
    expect(screen.getByText('No business unit yet')).toBeInTheDocument();
  });

  it('shows a plan in its own currency with the display currency beside (016 FR-014)', () => {
    renderCard({
      ...europe,
      planType: 'crossMarket',
      currency: 'GBP',
      planBudget: { amount: '600000.00', currency: 'GBP' },
      planPlannedCost: { amount: '480000.00', currency: 'GBP' },
    });

    expect(screen.getByText('Cross-market')).toBeInTheDocument();
    expect(moneyLine('Plan budget')).toHaveTextContent('600,000.00 GBP');
    expect(moneyLine('Plan budget')).toHaveTextContent('1,200,000.00 USD');
    expect(moneyLine('Planned cost')).toHaveTextContent('480,000.00 GBP≈ about 960,000.00 USD');
  });

  it('says "No activities yet" rather than "0 of 0" (functional review X-08)', () => {
    renderCard({ ...china, figures: { ...china.figures, activities: 0, activitiesApproved: 0 } });

    expect(screen.getByText('No activities yet')).toBeInTheDocument();
  });

  it('offers the pin star, named for the plan, filled while pinned (018 FR-001, FR-003)', async () => {
    const onPinToggle = vi.fn();
    const { user } = renderCard(europe, false, onPinToggle);

    const pin = screen.getByRole('button', { name: "Pin plan 'FY2026 Lubricants Europe'" });
    expect(pin.className).not.toMatch(/pinned/);
    await user.hover(pin);
    expect(await screen.findByRole('tooltip')).toHaveTextContent(
      "Pin plan 'FY2026 Lubricants Europe'",
    );
    await user.click(pin);

    expect(onPinToggle).toHaveBeenCalledTimes(1);
  });

  it('keeps the filled star visible on a pinned card, with the unpin tooltip (018 FR-003)', async () => {
    const { user } = renderCard(europe, true);

    const star = screen.getByRole('button', { name: "Unpin plan 'FY2026 Lubricants Europe'" });
    expect(star.className).toMatch(/pinned/);
    await user.hover(star);

    expect(await screen.findByRole('tooltip')).toHaveTextContent(
      "Unpin plan 'FY2026 Lubricants Europe'",
    );
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
