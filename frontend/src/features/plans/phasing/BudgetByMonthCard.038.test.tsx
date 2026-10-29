import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { Availability } from '@/session/access';
import { budgetByMonth, gbp, phasing } from '@/testing/fixtures/phasing';
import { testUser } from '@/testing/fixtures/users';
import { ALLOWED, refused, VIEWER } from '@/testing/fixtures/viewer';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { reply } from '@/testing/replies';
import { BudgetByMonthCard } from './BudgetByMonthCard';

const PLAN = {
  id: 'P-1004',
  name: 'FY2027 Lubricants UK & Ireland',
  version: 4,
  budget: gbp('600000.00'),
  budgetPhasing: phasing(),
};

function renderCard(negativeBalanceCount = 0, access: Availability = ALLOWED, user = testUser()) {
  const answer = budgetByMonth();
  mockApi([
    reply(
      'GET /api/v1/plans/{planId}/budget-by-month',
      {
        body: {
          ...answer,
          rows: answer.rows.map((row, i) => (i === 0 ? { ...row, negativeBalanceCount } : row)),
        },
      },
      { planId: 'P-1004' },
    ),
  ]);
  return renderWithProviders(
    <BudgetByMonthCard
      level="plan"
      planId="P-1004"
      record={PLAN}
      access={access}
      year={2027}
      currency="GBP"
    />,
    { user, route: '/plans/P-1004', path: '/plans/:planId' },
  );
}

describe('038 FR-024 (L-2): Budget by month says why its months add up to more', () => {
  it('names the activities whose line items are above their planned cost', async () => {
    renderCard(1);

    await screen.findByRole('table');
    expect(
      screen.getByText(
        '1 activity has line items above its planned cost, so its months add up to more than its planned cost.',
      ),
    ).toBeInTheDocument();
  });

  it('counts several', async () => {
    renderCard(3);
    await screen.findByRole('table');
    expect(
      screen.getByText(
        '3 activities have line items above their planned cost, so their months add up to more than their planned cost.',
      ),
    ).toBeInTheDocument();
  });

  it('says nothing when there are none', async () => {
    renderCard(0);
    await screen.findByRole('table');
    expect(screen.queryByText(/above (its|their) planned cost/)).not.toBeInTheDocument();
  });
});

describe('038 FR-029 (M-6): Edit budget by month follows the reader’s access', () => {
  it('offers no Edit to a Viewer', async () => {
    renderCard(0, refused('Editing plans'), VIEWER);

    await screen.findByRole('table');
    expect(screen.queryByRole('button', { name: 'Edit budget by month' })).not.toBeInTheDocument();
  });

  it('offers Edit to a person who may edit the plan', async () => {
    renderCard();

    await screen.findByRole('table');
    expect(screen.getByRole('button', { name: 'Edit budget by month' })).toBeInTheDocument();
  });
});

describe('038 FR-040 (L-4): a month problem names the month', () => {
  it('says which months are not amounts', async () => {
    const { user } = renderCard();

    await screen.findByRole('table');
    await user.click(screen.getByRole('button', { name: 'Edit budget by month' }));
    const dialog = await screen.findByRole('dialog');
    await user.click(within(dialog).getByRole('radio', { name: /Custom/ }));
    const march = within(dialog).getByRole('textbox', { name: 'Mar 2027' });
    await user.clear(march);
    await user.type(march, 'abc');
    await user.click(within(dialog).getByRole('button', { name: 'Save budget by month' }));

    expect(
      await within(dialog).findByText('Mar 2027: Type an amount of 0.00 or more.'),
    ).toBeInTheDocument();
  });
});
