import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { dashboard, figures } from '@/testing/fixtures/dashboards';
import { testUser } from '@/testing/fixtures/users';
import { renderWithProviders } from '@/testing/render';
import { ApprovalsPanel } from './ApprovalsPanel';
import { FigureCards } from './FigureCards';
import { GlobalBudgetsPanel } from './GlobalBudgetsPanel';
import { MonthlyChart } from './MonthlyChart';
import { PurchaseRequestsPanel } from './PurchaseRequestsPanel';

const user = testUser();
const usd = (amount: string) => ({ amount, currency: 'USD' });

describe('Dashboard panels (036 FR-010, FR-014, FR-016, FR-017, edge cases)', () => {
  it('writes over budget and a planned cost above the budget in words, not only in red (FR-010)', () => {
    renderWithProviders(
      <FigureCards figures={figures({ overBudget: true, leftToPlan: usd('-100.00') })} />,
      { user },
    );

    expect(screen.getByText('Over budget')).toBeInTheDocument();
    expect(screen.getByText('Planned cost is over the budget')).toBeInTheDocument();
  });

  it('says why a figure is missing: no budget, no financial data (FR-003, edge cases)', () => {
    renderWithProviders(
      <FigureCards
        figures={figures({ budget: null, committed: null, actual: null, budgetUsedPercent: null })}
      />,
      { user },
    );

    expect(screen.getByText('No budget set')).toBeInTheDocument();
    expect(screen.getAllByText("You can't see financial data")).toHaveLength(2);
  });

  it('says when nothing was submitted for approval yet (FR-016)', () => {
    const none = {
      ...dashboard().approvals,
      waiting: 0,
      approved: 0,
      declined: 0,
      oldestWaitingSince: null,
      averageDecisionDays: null,
    };
    renderWithProviders(<ApprovalsPanel approvals={none} />, { user });

    expect(screen.getByText('No approvals yet.')).toBeInTheDocument();
    expect(screen.getAllByText('—')).toHaveLength(2);
  });

  it('says when purchase requests are switched off instead of showing zeros (FR-017)', () => {
    const off = { ...dashboard().purchaseRequests, available: false };
    renderWithProviders(<PurchaseRequestsPanel purchaseRequests={off} />, { user });

    expect(screen.getByText('Purchase requests are switched off in Settings.')).toBeInTheDocument();
    expect(screen.queryByRole('table')).not.toBeInTheDocument();
  });

  it('marks an over-allocated global budget and says when the year has none (FR-004)', () => {
    const [budget] = dashboard().globalBudgets ?? [];
    if (!budget) {
      throw new Error('the fixture has a global budget');
    }
    renderWithProviders(<GlobalBudgetsPanel budgets={[{ ...budget, overAllocated: true }]} />, {
      user,
    });
    expect(screen.getByText('· Over-allocated')).toBeInTheDocument();
  });

  it('says when the year has no global budgets (FR-004)', () => {
    renderWithProviders(<GlobalBudgetsPanel budgets={[]} />, { user });
    expect(screen.getByText('No global budgets for this year.')).toBeInTheDocument();
  });

  it('notes amounts outside the months and hidden actuals under the chart (FR-014, FR-003)', () => {
    const months = dashboard().months;
    renderWithProviders(
      <MonthlyChart
        months={{
          ...months,
          otherYears: usd('700.00'),
          paidOutsideYear: usd('50.00'),
          items: months.items.map((m) => ({ ...m, actual: null, actualToDate: null })),
        }}
      />,
      { user },
    );

    expect(screen.getByText('Spread into other years: $700.00')).toBeInTheDocument();
    expect(screen.getByText('Paid outside the year: $50.00')).toBeInTheDocument();
    expect(screen.getByText("Actual: You can't see financial data")).toBeInTheDocument();
    expect(within(screen.getByRole('img')).queryAllByText('Mar')).toHaveLength(1);
  });
});
