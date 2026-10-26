import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import type { HttpHandler } from 'msw';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { FINANCE_BUDGETS, FORM_ANSWERS, plan, planDefaults } from '@/testing/fixtures/plans';
import { reply } from '@/testing/replies';
import { PlanForm } from './PlanForm';
import type { PlanFormProps } from './PlanForm';
import { valuesFromDefaults, valuesFromPlan } from './plan-form-values';

function renderForm(props: Partial<PlanFormProps> = {}, answers: HttpHandler[] = []) {
  const api = mockApi([...FORM_ANSWERS, ...answers]);
  const defaults = planDefaults();
  const result = renderWithProviders(
    <PlanForm
      mode="create"
      initial={valuesFromDefaults(defaults)}
      defaults={defaults}
      failure={null}
      onSave={async () => null}
      cancelTo="/plans"
      {...props}
    />,
    { user: testUser(), route: '/plans/new' },
  );
  return { ...result, api };
}

/** The headings of the form's sections, in order. */
function sectionTitles() {
  return screen.getAllByRole('heading', { level: 2 }).map((heading) => heading.textContent);
}

describe('PlanForm (US1, FR-005 – FR-010)', () => {
  it('shows the sections in order with markings as words and the defaults labelled', async () => {
    renderForm();

    expect(await screen.findByRole('textbox', { name: /Plan name/ })).toBeInTheDocument();
    expect(sectionTitles()).toEqual(['Basics', 'Budget', 'Shell attributes', 'Goals']);
    const basics = screen.getByRole('region', { name: 'Basics' });
    expect(within(basics).getByText('Plan name').parentElement).toHaveTextContent('Required');
    expect(screen.getByRole('combobox', { name: /Year/ })).toHaveValue('2026');
    expect(screen.getByText('Default: the current year')).toBeInTheDocument();
    expect(await screen.findByText('Default: your teams')).toBeInTheDocument();
    expect(await screen.findByText('Default: you')).toBeInTheDocument();
    expect(screen.getByText('Default: Evenly')).toBeInTheDocument();
    expect(screen.getByText('Default: INQ')).toBeInTheDocument();
    expect(screen.getAllByText('Default: No')).toHaveLength(2);
    expect(screen.getAllByText('Needed to publish').length).toBeGreaterThanOrEqual(3);
    // A model is not defaulted when several are active.
    expect(screen.queryByText('Default: the only active model')).not.toBeInTheDocument();
  });

  it('drops a default hint once the person changes the field', async () => {
    const { user } = renderForm();

    await user.selectOptions(await screen.findByRole('combobox', { name: /Year/ }), '2027');

    expect(screen.queryByText('Default: the current year')).not.toBeInTheDocument();
  });

  it('labels the only active model as the default and shows its rates', async () => {
    renderForm({
      initial: valuesFromDefaults(planDefaults({ modelId: 'MOD-1001' })),
      defaults: planDefaults({ modelId: 'MOD-1001' }),
    });

    expect(await screen.findByText('Default: the only active model')).toBeInTheDocument();
    expect(
      screen.getByText('INQ→MQL 2.00 % · MQL→deal 5.00 % · average deal size 10,000.00 USD'),
    ).toBeInTheDocument();
  });

  it('locks Owner with the reason while no team is chosen (US1#4, FR-007)', async () => {
    renderForm({
      initial: valuesFromDefaults(planDefaults({ teamIds: [], ownerId: null })),
      defaults: planDefaults({ teamIds: [], ownerId: null }),
    });

    expect(
      await screen.findByText(
        "Choose teams first — the owner must belong to one of the plan's teams.",
      ),
    ).toBeInTheDocument();
    expect(screen.getByRole('combobox', { name: /Owner/ })).toHaveAttribute(
      'aria-disabled',
      'true',
    );
  });

  it('names the owner who is in none of the chosen teams (FR-021)', async () => {
    renderForm({
      initial: { ...valuesFromDefaults(planDefaults()), teamIds: ['T-1002'] },
    });

    expect(
      await screen.findByText(
        "Priya Raman isn't in any of the chosen teams — choose another owner or clear the field",
      ),
    ).toBeInTheDocument();
  });

  it('calculates the goals live from the chosen model (US1#5, FR-009)', async () => {
    const { user } = renderForm();

    expect(await screen.findAllByText('Choose a model to calculate')).toHaveLength(3);
    await user.selectOptions(screen.getByRole('combobox', { name: /Model/ }), 'MOD-1001');
    await user.type(screen.getByRole('textbox', { name: 'Goal amount (INQ)' }), '250000');

    expect(screen.getByText('5,000')).toBeInTheDocument();
    expect(screen.getByText('2,500,000.00 USD')).toBeInTheDocument();
    expect(screen.getAllByText('Calculated')).toHaveLength(3);
    expect(screen.getByText('MQL = INQ × 2.00 %')).toBeInTheDocument();
  });

  it('shows business-unit attributes with the reason (FR-008)', async () => {
    const { user } = renderForm();

    await user.click(await screen.findByRole('checkbox', { name: 'Mobility' }));

    expect(screen.getByRole('textbox', { name: /Fleet size/ })).toBeInTheDocument();
    expect(
      screen.getByText("Shown because the plan's business unit includes Mobility."),
    ).toBeInTheDocument();
  });

  it('clears a finance budget of another year with a note (FR-007)', async () => {
    const { user } = renderForm({
      initial: { ...valuesFromDefaults(planDefaults()), defaultFinanceBudgetId: 'FB-1001' },
    });

    const budget = await screen.findByRole('combobox', { name: 'Default finance budget' });
    await waitFor(() => expect(budget).toHaveValue('FB-1001'));
    await user.selectOptions(screen.getByRole('combobox', { name: /Year/ }), '2027');

    expect(
      await screen.findByText(
        "Default finance budget cleared — '2026 Lubricants Budget' belongs to 2026.",
      ),
    ).toBeInTheDocument();
    expect(screen.getByRole('combobox', { name: 'Default finance budget' })).toHaveValue('none');
  });

  it('says when a year has no finance budgets yet', async () => {
    renderForm({}, [
      reply('GET /api/v1/finance-budgets', { body: { ...FINANCE_BUDGETS, items: [], total: 0 } }),
    ]);

    expect(
      await screen.findByText(
        'No finance budgets for 2026 yet — an administrator adds them in Settings.',
      ),
    ).toBeInTheDocument();
  });

  it('shows every problem at once, in the summary and under the fields (US1#3, FR-010)', async () => {
    const { user } = renderForm({
      onSave: async () => ({
        problems: [
          {
            field: 'name',
            message:
              "There is already a plan called 'FY2026 Lubricants Europe'. Choose another name.",
          },
          { field: 'modelId', message: 'Choose a model.' },
          { field: 'attributes.SA-BU', message: 'Choose a business unit.' },
        ],
      }),
    });
    await user.click(await screen.findByRole('button', { name: 'Save' }));

    const summary = await screen.findByRole('alert');
    expect(within(summary).getByText('3 problems need your attention')).toBeInTheDocument();
    await waitFor(() => expect(summary.parentElement).toHaveFocus());
    expect(screen.getAllByText('Choose a model.')).toHaveLength(2);
  });

  it('sends the typed values to onSave', async () => {
    const onSave = vi.fn(async () => null);
    const { user } = renderForm({ onSave });

    await user.type(await screen.findByRole('textbox', { name: /Plan name/ }), 'FY2026 Test');
    await user.click(screen.getByRole('button', { name: 'Save' }));

    await waitFor(() => expect(onSave).toHaveBeenCalledTimes(1));
    expect(onSave.mock.calls[0]).toEqual([
      expect.objectContaining({ name: 'FY2026 Test', year: '2026', teamIds: ['T-1001'] }),
    ]);
  });
});

describe('PlanForm on the edit page (US4, FR-018, FR-022)', () => {
  it('shows the year locked with the reason', async () => {
    renderForm({
      mode: 'edit',
      plan: plan(),
      initial: valuesFromPlan(plan()),
      defaults: undefined,
    });

    expect(
      await screen.findByText(
        "Year can't change once the plan is saved — copy the plan into the next year or create a new plan.",
      ),
    ).toBeInTheDocument();
    expect(screen.queryByRole('combobox', { name: /Year/ })).not.toBeInTheDocument();
  });

  it('says that changed defaults apply to line items created from now on', async () => {
    const { user } = renderForm({
      mode: 'edit',
      plan: plan(),
      initial: valuesFromPlan(plan()),
      defaults: undefined,
    });

    await user.selectOptions(
      await screen.findByRole('combobox', { name: 'Default line-item spread' }),
      'firstMonth',
    );

    expect(screen.getByText('Applies to line items created from now on.')).toBeInTheDocument();
  });
});
