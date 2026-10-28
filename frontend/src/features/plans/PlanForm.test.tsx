import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { HttpHandler } from 'msw';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { FINANCE_BUDGETS, FORM_ANSWERS, planDefaults } from '@/testing/fixtures/plans';
import { reply } from '@/testing/replies';
import { PlanForm } from './PlanForm';
import type { PlanFormProps } from './PlanForm';
import { valuesFromDefaults } from './plan-form-values';

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

/** Chooses `year` in the Year field: 11 years (034 FR-014) make it searchable (design system §5). */
async function chooseYear(user: ReturnType<typeof renderForm>['user'], year: string) {
  await user.click(screen.getByRole('combobox', { name: /Year/ }));
  await user.click(screen.getByRole('option', { name: year }));
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
    expect(screen.getByRole('button', { name: 'Remove 2026' })).toBeInTheDocument();
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

    await screen.findByRole('combobox', { name: /Year/ });
    await chooseYear(user, '2027');

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
    await chooseYear(user, '2027');

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
});
