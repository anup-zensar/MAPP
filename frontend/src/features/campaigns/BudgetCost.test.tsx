import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/test/render';
import { budgetInput, remainingWhileTyping } from './budget';
import { BudgetCostSection, GoalsSection, RemainingBudgetLine } from './BudgetCost';
import { budgetAndCost, usd } from './fixtures';

describe('budget arithmetic (feature 005, FR-010)', () => {
  it('takes the typed amount instead of the saved one, exact to the cent', () => {
    expect(remainingWhileTyping(usd('200000.00'), null, '300,000')).toEqual(usd('-100000.00'));
    expect(remainingWhileTyping(usd('0.10'), usd('0.20'), '0.3')).toEqual(usd('0.00'));
    expect(remainingWhileTyping(usd('100.00'), usd('50.00'), '')).toEqual(usd('150.00'));
    expect(remainingWhileTyping(usd('100.00'), null, 'abc')).toEqual(usd('100.00'));
    expect(remainingWhileTyping(null, null, '5')).toBeNull();
  });

  it('sends the budget rounded to the cent, as typed when it is not an amount, null when empty', () => {
    expect(budgetInput('12,500.505')).toEqual({ amount: '12500.51', currency: 'USD' });
    expect(budgetInput('12.5k')).toEqual({ amount: '12500.00', currency: 'USD' });
    expect(budgetInput(' abc ')).toEqual({ amount: 'abc', currency: 'USD' });
    expect(budgetInput('  ')).toBeNull();
  });
});

describe('RemainingBudgetLine (feature 005, FR-010)', () => {
  it('follows the typed budget and turns red below zero, with the reason', () => {
    renderWithProviders(
      <RemainingBudgetLine
        label="Remaining plan budget"
        tip="The plan's budget minus the budgets of all its campaigns, including this one."
        parentRemaining={usd('200000.00')}
        savedOwn={usd('100000.00')}
        typed="400,000"
      />,
    );

    expect(screen.getByText('-$100,000.00').className).toContain('negative');
    expect(screen.getByText('Over-allocated by $100,000.00')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'About Remaining plan budget' })).toBeInTheDocument();
  });

  it('says "—" with the reason when the parent has no budget', () => {
    renderWithProviders(
      <RemainingBudgetLine
        label="Remaining campaign budget"
        tip="The campaign's budget minus the budgets of all its initiatives, including this one."
        parentRemaining={null}
        savedOwn={null}
        typed="150000"
      />,
    );

    expect(screen.getByText('—')).toBeInTheDocument();
    expect(screen.getByText('No budget to allocate from')).toBeInTheDocument();
  });
});

describe('BudgetCostSection and GoalsSection (feature 005, FR-011, FR-016)', () => {
  it('shows planned cost as Calculated with its formula, and budget used in red over 100 %', async () => {
    const { user } = renderWithProviders(
      <BudgetCostSection
        level="campaign"
        budget={usd('120000.00')}
        parentRemainingBudget={usd('80000.00')}
        budgetAndCost={budgetAndCost({
          allocatedToChildren: usd('100000.00'),
          remainingForChildren: usd('20000.00'),
          plannedCost: usd('150000.00'),
          budgetUsedPercent: 125,
          overBudget: true,
          leftToPlan: usd('-30000.00'),
        })}
      />,
    );
    const section = screen.getByRole('region', { name: 'Budget and cost' });

    expect(within(section).getByText('$150,000.00')).toBeInTheDocument();
    expect(within(section).getByText('125%').className).toContain('negative');
    expect(within(section).getByRole('progressbar', { name: 'Budget used' })).toHaveAttribute(
      'aria-valuetext',
      '125 %',
    );
    expect(within(section).getByText('-$30,000.00')).toBeInTheDocument();
    await user.click(within(section).getByRole('button', { name: 'About Planned cost' }));
    expect(
      await screen.findByText('Sum of the planned costs of its initiatives.'),
    ).toBeInTheDocument();
  });

  it('shows "—" without a planned cost or a budget', () => {
    renderWithProviders(
      <BudgetCostSection
        level="initiative"
        budget={null}
        parentRemainingBudget={null}
        budgetAndCost={budgetAndCost()}
      />,
    );
    const section = screen.getByRole('region', { name: 'Budget and cost' });

    expect(within(section).getByText('No budget set')).toBeInTheDocument();
    expect(within(section).getByText('No budget to allocate from')).toBeInTheDocument();
    expect(within(section).getByText('No planned cost below yet')).toBeInTheDocument();
    expect(within(section).queryByRole('progressbar')).not.toBeInTheDocument();
  });

  it('sums the goals below, "—" for each without a value', () => {
    renderWithProviders(
      <GoalsSection goals={{ inq: 1200, mql: null, revenue: usd('25000.00') }} />,
    );
    const section = screen.getByRole('region', { name: /Goals/ });

    expect(within(section).getByText('1,200')).toBeInTheDocument();
    expect(within(section).getByText('—')).toBeInTheDocument();
    expect(within(section).getByText('$25,000.00')).toBeInTheDocument();
  });
});
