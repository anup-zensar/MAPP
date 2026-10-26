import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { closestTo } from '@/testing/dom';
import { renderPlanPage } from '@/testing/fixtures/plan-page';
import { plan } from '@/testing/fixtures/plans';

describe('PlanPage, Details tab (US3#5, FR-015, FR-016)', () => {
  it('shows the goals beside what the activities plan (US3#5, FR-016)', async () => {
    renderPlanPage(
      {
        body: plan({
          goals: {
            ...plan().goals,
            activities: {
              inq: 150000,
              mql: 3000,
              revenue: { amount: '1500000.00', currency: 'USD' },
            },
            plannedPercent: 60,
          },
        }),
      },
      [],
      '/plans/P-1001?tab=details',
    );

    const goals = closestTo(await screen.findByRole('heading', { name: 'Goals' }), 'section');
    const rows = within(goals)
      .getAllByRole('row')
      .slice(1)
      .map((row) => row.textContent);
    expect(rows).toEqual([
      'INQ250,000150,000',
      'MQLCalculated5,0003,000',
      'RevenueCalculated$2,500,000.00$1,500,000.00',
    ]);
    expect(within(goals).getByRole('button', { name: 'About MQL goal' })).toBeInTheDocument();
    expect(within(goals).getByText('60% of the INQ goal planned')).toBeInTheDocument();
  });

  it('says when no goal is set yet', async () => {
    renderPlanPage(
      {
        body: plan({
          goalAmount: null,
          goals: {
            ...plan().goals,
            plan: { inq: null, mql: null, deals: null, revenue: null, reasons: {} },
            plannedPercent: null,
          },
        }),
      },
      [],
      '/plans/P-1001?tab=details',
    );

    expect(
      await screen.findByText('No goal set yet — set a goal amount in Edit.'),
    ).toBeInTheDocument();
  });

  it('shows every detail read-only, with "—" and "(inactive)" (FR-015)', async () => {
    renderPlanPage(
      { body: plan({ description: '', model: { ...plan().model, active: false } }) },
      [],
      '/plans/P-1001?tab=details',
    );

    const details = closestTo(await screen.findByRole('heading', { name: 'Details' }), 'section');
    expect(within(details).getByText('Amounts in USD')).toBeInTheDocument();
    expect(within(details).getByText('Description').nextElementSibling).toHaveTextContent('—');
    expect(within(details).getByText('Model').nextElementSibling).toHaveTextContent(
      'Lubricants 1.0 (inactive)INQ→MQL 2.00 % · MQL→deal 5.00 % · average deal size 10,000.00 USD',
    );
    expect(
      within(details).getByText('Default line-item spread').nextElementSibling,
    ).toHaveTextContent('Evenly');
    expect(await within(details).findByText('Lubricants')).toBeInTheDocument();
  });
});
