import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { expectNoAxeViolations } from '@/testing/axe';
import { PEOPLE } from '@/testing/fixtures/approval-workflows';
import { globalBudget } from '@/testing/fixtures/global-budgets';
import { budgetRow, renderBudgetPage } from '@/testing/fixtures/global-budget-page';
import { reply } from '@/testing/replies';

const OVERSPEND = 'When planned cost goes over this budget';

describe('GlobalBudgetPage — overspend control (037 US3)', () => {
  it('chooses Refuse for the plan budget in Change allocation, refusing a cut below its planned cost (FR-004, FR-015)', async () => {
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
    const group = within(dialog).getByRole('group', { name: OVERSPEND });
    expect(within(group).getByRole('radio', { name: /^Warn/ })).toBeChecked();

    await user.click(within(group).getByRole('radio', { name: /^Refuse/ }));
    const amount = within(dialog).getByRole('textbox', { name: /Amount/ });
    await user.clear(amount);
    await user.type(amount, '400000');
    const refused =
      "The planned cost is 460,000.00 USD; this budget refuses overspending, so it can't be lower than that.";
    expect(within(dialog).getByText(refused)).toBeVisible();
    await expectNoAxeViolations(dialog);
    await user.click(within(dialog).getByRole('button', { name: 'Save allocation' }));
    expect(api.bodiesOf('PATCH /api/v1/budget-allocations/BA-1001')).toHaveLength(0);

    await user.clear(amount);
    await user.type(amount, '500000');
    await user.click(within(dialog).getByRole('button', { name: 'Save allocation' }));
    expect(
      (await screen.findAllByText("Allocation to 'FY2026 Lubricants Europe' saved"))[0],
    ).toBeInTheDocument();
    expect(api.bodiesOf('PATCH /api/v1/budget-allocations/BA-1001')[0]).toMatchObject({
      overspend: 'refuse',
    });
  });

  it('offers the setting in Edit global budget, refusing an amount below the planned cost (FR-001, FR-015)', async () => {
    const { api, user } = renderBudgetPage([reply('GET /api/v1/people', PEOPLE)]);
    await user.click(await screen.findByRole('button', { name: 'Edit' }));
    const panel = await screen.findByRole('dialog', { name: 'Edit global budget' });
    await user.click(within(panel).getByRole('radio', { name: /^Refuse/ }));
    const amount = within(panel).getByRole('textbox', { name: /Amount/ });
    await user.clear(amount);
    await user.type(amount, '600000');

    expect(
      within(panel).getByText(
        "The planned cost is 670,000.00 USD; this budget refuses overspending, so it can't be lower than that.",
      ),
    ).toBeVisible();
    await user.click(within(panel).getByRole('button', { name: 'Save' }));
    expect(api.bodiesOf('PATCH /api/v1/global-budgets/GB-1001')).toHaveLength(0);
  });
});
