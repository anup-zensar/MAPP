import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { globalBudget, LUBRICANTS } from '@/testing/fixtures/global-budgets';
import { budgetRow, renderBudgetPage } from '@/testing/fixtures/global-budget-page';
import { reply } from '@/testing/replies';

const ONE = 'GET /api/v1/global-budgets/GB-1001';

describe('GlobalBudgetPage (014 US6, FR-032 – FR-035)', () => {
  it('shows the figures and one row per allocated plan (US6 scenario 1)', async () => {
    renderBudgetPage();

    expect(
      await screen.findByRole('heading', { level: 1, name: 'FY2026 Global marketing budget' }),
    ).toBeInTheDocument();
    expect(document.title).toBe('FY2026 Global marketing budget · SP&A One');
    expect(screen.getByRole('link', { name: 'Global budgets' })).toHaveAttribute(
      'href',
      '/budgets',
    );
    expect(screen.getByText('Owner: Carlos Mendes')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '$2.5M ($2,500,000.00)' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '$2M ($2,000,000.00)' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '$500K ($500,000.00)' })).toBeInTheDocument();

    const allocations = screen.getByRole('table', { name: 'Allocations' });
    expect(within(allocations).getAllByRole('row')).toHaveLength(3);
    const lubricants = budgetRow('FY2026 Lubricants Europe');
    expect(
      within(lubricants).getByRole('link', { name: 'FY2026 Lubricants Europe' }),
    ).toHaveAttribute('href', '/plans/P-1001');
    expect(lubricants).toHaveTextContent('Lubricants Europe');
    expect(lubricants).toHaveTextContent('Priya Raman');
    expect(lubricants).toHaveTextContent('Published');
    expect(lubricants).toHaveTextContent('$1,200,000.00');
    expect(lubricants).toHaveTextContent('$460,000.00');
    expect(lubricants).toHaveTextContent('$740,000.00');
    expect(lubricants).toHaveTextContent('38.33 %');
    expect(
      within(lubricants).getByRole('progressbar', { name: 'Budget used' }),
    ).toBeInTheDocument();
    expect(budgetRow('FY2026 Mobility China')).toHaveTextContent('$800,000.00');

    const without = screen.getByRole('table', { name: 'Plans of 2026 without an allocation' });
    expect(within(without).getByRole('row', { name: /FY2026 Brand Europe/ })).toHaveTextContent(
      '$300,000.00',
    );
  });

  it('adds up the plans the person can not see in "Other markets" (FR-033)', async () => {
    renderBudgetPage([
      reply(
        'GET /api/v1/global-budgets/{globalBudgetId}',
        {
          body: globalBudget({
            allocations: [LUBRICANTS],
            otherMarkets: {
              planCount: 1,
              amount: { amount: '800000.00', currency: 'USD' },
              plannedCost: { amount: '210000.00', currency: 'USD' },
            },
          }),
        },
        { globalBudgetId: 'GB-1001' },
      ),
    ]);

    const other = await screen.findByRole('row', { name: /Other markets \(1 plan\)/ });
    expect(other).toHaveTextContent('$800,000.00');
    expect(other).toHaveTextContent('$210,000.00');
    expect(screen.queryByRole('link', { name: 'FY2026 Mobility China' })).toBeNull();
  });

  it('shows Unallocated in red with words when over-allocated (US6 scenario 5)', async () => {
    renderBudgetPage([
      reply(
        'GET /api/v1/global-budgets/{globalBudgetId}',
        {
          body: globalBudget({
            figures: {
              ...globalBudget().figures,
              allocated: { amount: '2600000.00', currency: 'USD' },
              unallocated: { amount: '-100000.00', currency: 'USD' },
              overAllocated: true,
            },
          }),
        },
        { globalBudgetId: 'GB-1001' },
      ),
    ]);

    expect(await screen.findByText('Over-allocated by $100,000.00')).toBeInTheDocument();
  });

  it("says why a global budget with allocations can't be deleted (FR-035)", async () => {
    const reason =
      "'FY2026 Global marketing budget' has allocations to 2 plans. Remove them first.";
    const { user } = renderBudgetPage([
      reply(
        'GET /api/v1/global-budgets/{globalBudgetId}',
        {
          body: globalBudget({
            access: {
              edit: { available: true, reason: '' },
              delete: { available: false, reason },
              allocate: { available: true, reason: '' },
            },
          }),
        },
        { globalBudgetId: 'GB-1001' },
      ),
    ]);
    await screen.findByRole('table', { name: 'Allocations' });

    await user.click(screen.getByRole('button', { name: 'More' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Delete' }));
    const dialog = await screen.findByRole('alertdialog', {
      name: "Can't delete 'FY2026 Global marketing budget'",
    });
    expect(dialog).toHaveTextContent(reason);
  });

  it('deletes a global budget without allocations and goes back to the list', async () => {
    const { api, router, user } = renderBudgetPage([
      reply(
        'GET /api/v1/global-budgets/{globalBudgetId}',
        { body: globalBudget({ allocations: [], plansWithoutAllocation: [] }) },
        { globalBudgetId: 'GB-1001' },
      ),
      reply(
        'DELETE /api/v1/global-budgets/{globalBudgetId}',
        {
          body: {
            recycleBinEntry: {
              id: 'RB-1009',
              recordType: 'globalBudget',
              recordTypeLabel: 'global budget',
              recordId: 'GB-1001',
              recordName: 'FY2026 Global marketing budget',
              path: [],
              parent: null,
              contents: [],
              deletedAt: '2026-10-20T09:00:00Z',
              deletedBy: { id: 'U-1005', name: 'Carlos Mendes' },
            },
          },
        },
        { globalBudgetId: 'GB-1001' },
      ),
    ]);
    expect(await screen.findByText('No plan has an allocation yet')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'More' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Delete' }));
    const confirm = await screen.findByRole('alertdialog', {
      name: "Delete global budget 'FY2026 Global marketing budget'?",
    });
    await user.click(within(confirm).getByRole('button', { name: 'Delete global budget' }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/budgets'));
    // The deleted global budget is not asked for again while its page closes (no "not found").
    const asked = api.requests.filter((r) => `${r.method} ${r.path}` === ONE);
    expect(asked).toHaveLength(1);
  });

  it('says when the global budget no longer exists', async () => {
    renderBudgetPage([
      reply(
        'GET /api/v1/global-budgets/{globalBudgetId}',
        {
          status: 404,
          body: {
            error: {
              code: 'not_found',
              message: 'This global budget no longer exists.',
              fields: [],
            },
          },
        },
        { globalBudgetId: 'GB-1001' },
      ),
    ]);

    expect(
      await screen.findByText('This global budget no longer exists — someone may have deleted it.'),
    ).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Go to Global budgets' })).toHaveAttribute(
      'href',
      '/budgets',
    );
  });
});
