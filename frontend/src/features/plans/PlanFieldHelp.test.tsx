import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { fieldHelpReply, helpOf } from '@/testing/fixtures/field-help-texts';
import { FORM_ANSWERS, plan, planDefaults } from '@/testing/fixtures/plans';
import { testUser } from '@/testing/fixtures/users';
import { PlanDetails } from './PlanDetails';
import { PlanForm } from './PlanForm';
import { valuesFromDefaults } from './plan-form-values';
import { PlanHeader } from './PlanHeader';
import { PlanSummaryCards } from './PlanSummaryCards';

const FORM_FIELDS: [string, string][] = [
  ['Plan name', 'plan.name'],
  ['Year', 'plan.year'],
  ['Model', 'plan.model'],
  ['Teams', 'plan.teams'],
  ['Owner', 'plan.owner'],
  ['Strategic / AOP plan', 'plan.aop'],
  ['Description', 'plan.description'],
  ['Plan budget', 'plan.budget'],
  ['Default finance budget', 'plan.defaultFinanceBudget'],
  ['Default line-item spread', 'plan.defaultSpread'],
  ['Goal stage', 'plan.goalStage'],
  ['Goal amount (INQ)', 'plan.goalAmount'],
];

const DETAILS: [string, string][] = [
  ['Plan name', 'plan.name'],
  ['Year', 'plan.year'],
  ['Model', 'plan.model'],
  ['Teams', 'plan.teams'],
  ['Owner', 'plan.owner'],
  ['Strategic / AOP plan', 'plan.aop'],
  ['Description', 'plan.description'],
  ['Plan budget', 'plan.budget'],
  ['Default finance budget', 'plan.defaultFinanceBudget'],
  ['Default line-item spread', 'plan.defaultSpread'],
];

const CARDS: [string, string][] = [
  ['Plan budget', 'plan.budget'],
  ['Planned cost', 'plan.plannedCost'],
  ['Budget used', 'plan.budgetUsed'],
  ['Activities approved', 'plan.activitiesApproved'],
];

const keysOf = (rows: [string, string][]) => rows.map(([, key]) => key);

async function expectTips(scope: HTMLElement, rows: [string, string][]) {
  for (const [label] of rows) {
    const tips = await within(scope).findAllByRole('button', { name: `About ${label}` });
    expect(tips, label).toHaveLength(1);
  }
}

describe('016 FR-047: the plan screens show the Field help of every labelled field', () => {
  it('gives each field of the plan form one ⓘ, and none for an emptied text', async () => {
    mockApi([
      ...FORM_ANSWERS,
      fieldHelpReply({ ...helpOf(keysOf(FORM_FIELDS)), 'plan.bcpir': '' }),
    ]);
    const defaults = planDefaults();
    const { user } = renderWithProviders(
      <PlanForm
        mode="create"
        initial={valuesFromDefaults(defaults)}
        defaults={defaults}
        failure={null}
        onSave={async () => null}
        cancelTo="/plans"
      />,
      { user: testUser(), route: '/plans/new' },
    );

    await expectTips(document.body, FORM_FIELDS);
    expect(screen.queryByRole('button', { name: 'About BCPIR plan' })).not.toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'About Plan budget' }));
    expect(await screen.findByRole('dialog', { name: 'About Plan budget' })).toHaveTextContent(
      'Help of plan.budget.',
    );
  });

  it('gives each value of the Details card one ⓘ', async () => {
    mockApi([...FORM_ANSWERS, fieldHelpReply(helpOf(keysOf(DETAILS)))]);
    renderWithProviders(<PlanDetails plan={plan()} />, { user: testUser() });

    await expectTips(screen.getByRole('region', { name: 'Details' }), DETAILS);
  });

  it('explains the summary cards and the header’s status and completeness', async () => {
    mockApi([fieldHelpReply(helpOf([...keysOf(CARDS), 'plan.status', 'common.completeness']))]);
    renderWithProviders(
      <>
        <PlanHeader plan={plan()} moreItems={[]} />
        <PlanSummaryCards plan={plan()} />
      </>,
      { user: testUser() },
    );

    await expectTips(screen.getByRole('region', { name: 'Summary' }), CARDS);
    const facts = screen.getByRole('group', { name: 'Plan facts' });
    expect(await within(facts).findByRole('button', { name: 'About Status' })).toBeVisible();
    expect(within(facts).getByRole('button', { name: 'About Completeness' })).toBeVisible();
  });
});
