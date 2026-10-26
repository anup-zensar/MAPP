import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { PlanTable } from './PlanTable';
import type { PlanSummary } from './plansApi';
import { seedList } from '@/testing/fixtures/plans';

function renderTable(
  plans: PlanSummary[] = seedList().items,
  isPinned: (plan: PlanSummary) => boolean = () => false,
) {
  return renderWithProviders(
    <PlanTable plans={plans} actionsFor={() => []} isPinned={isPinned} onPinToggle={() => {}} />,
    {
      user: testUser(),
      route: '/plans',
      path: '/plans',
    },
  );
}

function row(name: string) {
  return screen.getByRole('link', { name }).closest('tr') as HTMLElement;
}

describe('PlanTable (FR-002, design system §5.4)', () => {
  it('shows one row per plan with its facts, never repeating a column label (UI audit P-02)', () => {
    renderTable();

    expect(screen.getAllByRole('columnheader').map((header) => header.textContent)).toEqual([
      'Pinned',
      'Name',
      'Status',
      'Owner',
      'Teams',
      'Plan budget',
      'Planned cost',
      'Approved',
      'Actions',
    ]);
    const europe = row('FY2026 Lubricants Europe');
    expect(europe).toHaveTextContent('P-1001 · 2026 · AOP');
    expect(europe).toHaveTextContent('Operational');
    expect(europe).toHaveTextContent('Priya Raman');
    expect(europe).not.toHaveTextContent('Owner:');
    expect(europe).toHaveTextContent('$1,200,000.00');
    expect(europe).toHaveTextContent('$960,000.0080% used');
    expect(europe).toHaveTextContent('4 of 7');
  });

  it('right-aligns money under right-aligned headers', () => {
    renderTable();

    const headers = screen.getAllByRole('columnheader');
    const budget = headers.find((header) => header.textContent === 'Plan budget') as HTMLElement;
    const cells = within(row('FY2026 Lubricants Europe')).getAllByRole('cell');
    expect(budget.className).toMatch(/number/);
    expect(cells[4].className).toMatch(/number/);
  });

  it('says over budget in words and "—" without a budget', () => {
    renderTable();

    expect(within(row('FY2025 Lubricants Europe')).getByText('108% used').className).toMatch(
      /problem/,
    );
    const china = within(row('FY2026 Mobility China')).getAllByRole('cell');
    expect(china[4]).toHaveTextContent('—');
  });

  it('marks pinned rows with the filled star, named for the plan (018 FR-001, FR-003)', () => {
    renderTable(seedList().items, (plan) => plan.id === 'P-1003');

    const pinned = within(row('FY2025 Lubricants Europe')).getByRole('button', {
      name: "Unpin plan 'FY2025 Lubricants Europe'",
    });
    expect(pinned.className).toMatch(/pinned/);
    const unpinned = within(row('FY2026 Mobility China')).getByRole('button', {
      name: "Pin plan 'FY2026 Mobility China'",
    });
    expect(unpinned.className).not.toMatch(/pinned/);
  });

  it('shows at most two teams, then "+n" with the other names', () => {
    const [plan] = seedList().items;
    const team = (id: string, name: string) => ({ ...plan.teams[0], id, name });
    renderTable([
      { ...plan, teams: [team('T-1', 'Alpha'), team('T-2', 'Beta'), team('T-3', 'Gamma')] },
    ]);

    const cells = within(row('FY2026 Lubricants Europe')).getAllByRole('cell');
    expect(cells[3]).toHaveTextContent('AlphaBeta+1');
    expect(within(cells[3]).getByRole('button', { name: '+1 more: Gamma' })).toBeInTheDocument();
  });

  it('links completeness to the field to fill in (FR-017)', async () => {
    const { user, router } = renderTable();

    await user.click(
      within(row('FY2026 Mobility China')).getByRole('button', { name: '1 field needed' }),
    );
    await user.click(await screen.findByRole('button', { name: 'Business unit' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/plans/P-1002/edit'));
    expect(router.state.location.search).toBe('?focus=attributes.SA-BU');
  });
});
