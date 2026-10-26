import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { BRAND_OPTION, globalBudget, LUBRICANTS } from '@/testing/fixtures/global-budgets';
import { budgetRow, renderBudgetPage } from '@/testing/fixtures/global-budget-page';
import { reply } from '@/testing/replies';

describe('GlobalBudgetPage allocations (014 US6, FR-030, FR-034)', () => {
  it('changes an allocation with the allocation and plan versions (US6 scenario 2)', async () => {
    const { api, user } = renderBudgetPage([
      reply(
        'PATCH /api/v1/budget-allocations/{allocationId}',
        { body: globalBudget() },
        { allocationId: 'BA-1001' },
      ),
    ]);
    await screen.findByRole('table', { name: 'Allocations' });

    await user.click(
      within(budgetRow('FY2026 Lubricants Europe')).getByRole('button', {
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
    const { api, user } = renderBudgetPage([
      reply(
        'POST /api/v1/global-budgets/{globalBudgetId}/allocations',
        { status: 201, body: globalBudget() },
        { globalBudgetId: 'GB-1001' },
      ),
    ]);
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
    const { user } = renderBudgetPage([
      reply(
        'GET /api/v1/global-budgets/{globalBudgetId}/allocation-options',
        {
          body: { items: [BRAND_OPTION], total: 1 },
        },
        { globalBudgetId: 'GB-1001' },
      ),
      reply(
        'POST /api/v1/global-budgets/{globalBudgetId}/allocations',
        {
          status: 400,
          body: {
            error: {
              code: 'validation_failed',
              message: 'Check the highlighted fields.',
              fields: [{ field: 'planId', message: 'Choose a plan of 2026.' }],
            },
          },
        },
        { globalBudgetId: 'GB-1001' },
      ),
    ]);
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
    const { api, user } = renderBudgetPage([
      reply(
        'DELETE /api/v1/budget-allocations/{allocationId}',
        {
          body: globalBudget({ allocations: [LUBRICANTS] }),
        },
        { allocationId: 'BA-1002' },
      ),
    ]);
    await screen.findByRole('table', { name: 'Allocations' });

    await user.click(
      within(budgetRow('FY2026 Mobility China')).getByRole('button', {
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
    expect(api.requests.map((request) => `${request.method} ${request.path}`)).toContain(
      'DELETE /api/v1/budget-allocations/BA-1002',
    );
  });
});
