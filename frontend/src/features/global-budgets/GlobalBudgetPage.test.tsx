import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/testing/render';
import { mockApi } from '@/testing/mockApi';
import type { MockHandler } from '@/testing/mockApi';
import { testUser } from '@/testing/users';
import { GlobalBudgetPage } from './GlobalBudgetPage';
import { BRAND_OPTION, globalBudget, LUBRICANTS, MOBILITY } from './testData';

const ONE = 'GET /api/v1/global-budgets/GB-1001';

function renderPage(answers: Record<string, MockHandler> = {}) {
  const api = mockApi({ [ONE]: { body: globalBudget() }, ...answers });
  return {
    ...renderWithProviders(<GlobalBudgetPage />, {
      route: '/budgets/GB-1001',
      path: '/budgets/:globalBudgetId',
      user: testUser(),
    }),
    api,
  };
}

const rowOf = (name: string) => screen.getByRole('row', { name: new RegExp(name) });

describe('GlobalBudgetPage (014 US6, FR-032 – FR-035)', () => {
  it('shows the figures and one row per allocated plan (US6 scenario 1)', async () => {
    renderPage();

    expect(
      await screen.findByRole('heading', { level: 1, name: 'FY2026 Global marketing budget' }),
    ).toBeInTheDocument();
    expect(document.title).toBe('FY2026 Global marketing budget · MAPP');
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
    const lubricants = rowOf('FY2026 Lubricants Europe');
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
    expect(rowOf('FY2026 Mobility China')).toHaveTextContent('$800,000.00');

    const without = screen.getByRole('table', { name: 'Plans of 2026 without an allocation' });
    expect(within(without).getByRole('row', { name: /FY2026 Brand Europe/ })).toHaveTextContent(
      '$300,000.00',
    );
  });

  it('adds up the plans the person can not see in "Other markets" (FR-033)', async () => {
    renderPage({
      [ONE]: {
        body: globalBudget({
          allocations: [LUBRICANTS],
          otherMarkets: {
            planCount: 1,
            amount: { amount: '800000.00', currency: 'USD' },
            plannedCost: { amount: '210000.00', currency: 'USD' },
          },
        }),
      },
    });

    const other = await screen.findByRole('row', { name: /Other markets \(1 plan\)/ });
    expect(other).toHaveTextContent('$800,000.00');
    expect(other).toHaveTextContent('$210,000.00');
    expect(screen.queryByRole('link', { name: 'FY2026 Mobility China' })).toBeNull();
  });

  it('shows Unallocated in red with words when over-allocated (US6 scenario 5)', async () => {
    renderPage({
      [ONE]: {
        body: globalBudget({
          figures: {
            ...globalBudget().figures,
            allocated: { amount: '2600000.00', currency: 'USD' },
            unallocated: { amount: '-100000.00', currency: 'USD' },
            overAllocated: true,
          },
        }),
      },
    });

    expect(await screen.findByText('Over-allocated by $100,000.00')).toBeInTheDocument();
  });

  it('changes an allocation with the allocation and plan versions (US6 scenario 2)', async () => {
    const { api, user } = renderPage({
      'PATCH /api/v1/budget-allocations/BA-1001': { body: globalBudget() },
    });
    await screen.findByRole('table', { name: 'Allocations' });

    await user.click(
      within(rowOf('FY2026 Lubricants Europe')).getByRole('button', {
        name: 'Actions for FY2026 Lubricants Europe',
      }),
    );
    await user.click(await screen.findByRole('menuitem', { name: 'Change allocation' }));
    const dialog = await screen.findByRole('dialog', {
      name: "Change the allocation to 'FY2026 Lubricants Europe'",
    });
    const amount = within(dialog).getByRole('textbox', { name: /Amount/ });
    expect(amount).toHaveValue('1,200,000.00');
    await user.clear(amount);
    await user.type(amount, '1300000');
    await user.click(within(dialog).getByRole('button', { name: 'Save allocation' }));

    expect(
      (await screen.findAllByText("Allocation to 'FY2026 Lubricants Europe' saved"))[0],
    ).toBeInTheDocument();
    expect(api.bodiesOf('PATCH /api/v1/budget-allocations/BA-1001')).toEqual([
      { amount: { amount: '1300000.00', currency: 'USD' }, version: 1, planVersion: 4 },
    ]);
  });

  it('allocates to a plan without an allocation', async () => {
    const { api, user } = renderPage({
      'POST /api/v1/global-budgets/GB-1001/allocations': { status: 201, body: globalBudget() },
    });
    await screen.findByRole('table', { name: 'Allocations' });

    await user.click(screen.getByRole('button', { name: 'Allocate to FY2026 Brand Europe' }));
    const dialog = await screen.findByRole('dialog', { name: "Allocate to 'FY2026 Brand Europe'" });
    expect(within(dialog).getByText('Its budget now: $300,000.00')).toBeInTheDocument();
    await user.type(within(dialog).getByRole('textbox', { name: /Amount/ }), '250000');
    await user.click(within(dialog).getByRole('button', { name: 'Allocate' }));

    expect(
      (await screen.findAllByText("Allocation to 'FY2026 Brand Europe' saved"))[0],
    ).toBeInTheDocument();
    expect(api.bodiesOf('POST /api/v1/global-budgets/GB-1001/allocations')).toEqual([
      { planId: 'P-1005', amount: { amount: '250000.00', currency: 'USD' }, planVersion: 1 },
    ]);
  });

  it('chooses the plan from the allocation options and shows refusals at the fields', async () => {
    const { user } = renderPage({
      'GET /api/v1/global-budgets/GB-1001/allocation-options': {
        body: { items: [BRAND_OPTION], total: 1 },
      },
      'POST /api/v1/global-budgets/GB-1001/allocations': {
        status: 400,
        body: {
          error: {
            code: 'validation_failed',
            message: 'Check the highlighted fields.',
            fields: [{ field: 'planId', message: 'Choose a plan of 2026.' }],
          },
        },
      },
    });
    await screen.findByRole('table', { name: 'Allocations' });

    await user.click(screen.getByRole('button', { name: 'Allocate to a plan' }));
    const dialog = await screen.findByRole('dialog', {
      name: "Allocate from 'FY2026 Global marketing budget'",
    });
    await user.click(within(dialog).getByRole('button', { name: 'Allocate' }));
    expect(within(dialog).getByText('Choose a plan.')).toBeInTheDocument();

    await user.selectOptions(
      await within(dialog).findByRole('combobox', { name: /Plan/ }),
      'FY2026 Brand Europe',
    );
    await user.type(within(dialog).getByRole('textbox', { name: /Amount/ }), '100');
    await user.click(within(dialog).getByRole('button', { name: 'Allocate' }));
    expect(await within(dialog).findByText('Choose a plan of 2026.')).toBeInTheDocument();
  });

  it('removes an allocation after asking; the plan keeps its budget (US6 scenario 6)', async () => {
    const { api, user } = renderPage({
      'DELETE /api/v1/budget-allocations/BA-1002': {
        body: globalBudget({ allocations: [LUBRICANTS] }),
      },
    });
    await screen.findByRole('table', { name: 'Allocations' });

    await user.click(
      within(rowOf('FY2026 Mobility China')).getByRole('button', {
        name: 'Actions for FY2026 Mobility China',
      }),
    );
    await user.click(await screen.findByRole('menuitem', { name: 'Remove allocation' }));
    const confirm = await screen.findByRole('alertdialog', {
      name: "Remove the allocation to 'FY2026 Mobility China'?",
    });
    expect(confirm).toHaveTextContent(
      "'FY2026 Mobility China' keeps its budget of $800,000.00, and its owner can change the budget again.",
    );
    await user.click(within(confirm).getByRole('button', { name: 'Remove allocation' }));

    expect(
      (await screen.findAllByText("Allocation to 'FY2026 Mobility China' removed"))[0],
    ).toBeInTheDocument();
    expect(api.requests.some((request) => request.method === 'DELETE')).toBe(true);
    expect(MOBILITY.id).toBe('BA-1002');
  });

  it("says why a global budget with allocations can't be deleted (FR-035)", async () => {
    const reason =
      "'FY2026 Global marketing budget' has allocations to 2 plans. Remove them first.";
    const { user } = renderPage({
      [ONE]: {
        body: globalBudget({
          access: {
            edit: { available: true, reason: '' },
            delete: { available: false, reason },
            allocate: { available: true, reason: '' },
          },
        }),
      },
    });
    await screen.findByRole('table', { name: 'Allocations' });

    await user.click(screen.getByRole('button', { name: 'More' }));
    await user.click(await screen.findByRole('menuitem', { name: 'Delete' }));
    const dialog = await screen.findByRole('alertdialog', {
      name: "Can't delete 'FY2026 Global marketing budget'",
    });
    expect(dialog).toHaveTextContent(reason);
  });

  it('deletes a global budget without allocations and goes back to the list', async () => {
    const { api, router, user } = renderPage({
      [ONE]: { body: globalBudget({ allocations: [], plansWithoutAllocation: [] }) },
      'DELETE /api/v1/global-budgets/GB-1001': {
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
    });
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
    renderPage({
      [ONE]: {
        status: 404,
        body: { error: { code: 'not_found', message: 'This global budget no longer exists.' } },
      },
    });

    expect(
      await screen.findByText('This global budget no longer exists — someone may have deleted it.'),
    ).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Go to Global budgets' })).toHaveAttribute(
      'href',
      '/budgets',
    );
  });
});
