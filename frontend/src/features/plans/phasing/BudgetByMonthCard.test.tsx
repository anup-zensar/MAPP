import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { mockApi } from '@/testing/mock-api';
import type { MockHandler } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { BudgetByMonthCard } from './BudgetByMonthCard';
import type { BudgetPhasing } from './api';
import { budgetByMonth, gbp, periods, phasing, twelve } from '@/testing/fixtures/phasing';

const PLAN = {
  id: 'P-1004',
  name: 'FY2027 Lubricants UK & Ireland',
  version: 4,
  budget: gbp('600000.00'),
};

function renderCard(
  record: typeof PLAN & { budgetPhasing?: BudgetPhasing } = { ...PLAN, budgetPhasing: phasing() },
  answers: Record<string, MockHandler> = {},
  level: 'plan' | 'campaign' = 'plan',
) {
  const api = mockApi({
    'GET /api/v1/plans/P-1004/budget-by-month': { body: budgetByMonth() },
    ...answers,
  });
  const result = renderWithProviders(
    <BudgetByMonthCard
      level={level}
      planId="P-1004"
      record={record}
      year={2027}
      currency="GBP"
      variant={level === 'plan' ? 'page' : 'panel'}
    />,
    { user: testUser(), route: '/plans/P-1004', path: '/plans/:planId' },
  );
  return { ...result, api };
}

describe('BudgetByMonthCard (016 FR-024)', () => {
  it('shows nothing while the API does not phase budgets yet', () => {
    const api = mockApi({});
    renderWithProviders(
      <BudgetByMonthCard level="plan" planId="P-1004" record={PLAN} year={2027} currency="GBP" />,
      { user: testUser() },
    );

    expect(screen.queryByRole('heading', { name: 'Budget by month' })).not.toBeInTheDocument();
    expect(api.requests).toHaveLength(0);
  });

  it('shows budget, planned cost, difference and allocated by month in the plan currency', async () => {
    renderCard();

    const card = screen.getByRole('region', { name: 'Budget by month' });
    expect(within(card).getByText('Amounts in GBP, the plan currency · 2027')).toBeInTheDocument();
    const table = await within(card).findByRole('table');
    const headers = within(table)
      .getAllByRole('columnheader')
      .map((header) => header.textContent);
    expect(headers).toEqual([
      'Row',
      'Jan',
      'Feb',
      'Mar',
      'Apr',
      'May',
      'Jun',
      'Jul',
      'Aug',
      'Sep',
      'Oct',
      'Nov',
      'Dec',
      'Total',
    ]);
    const rows = within(table).getAllByRole('row').slice(1);
    expect(rows.map((row) => within(row).getByRole('rowheader').textContent)).toEqual([
      'Budget',
      'Planned cost',
      'Difference',
      'Allocated to campaigns',
    ]);
    expect(rows[0]).toHaveTextContent('50,000.00');
    expect(rows[0]).toHaveTextContent('600,000.00');
    expect(rows[1]).toHaveTextContent('480,000.00');
    expect(within(card).getByText('1 campaign is not phased')).toBeInTheDocument();
  });

  it('switches to quarters', async () => {
    const { user } = renderCard();

    await screen.findByRole('table');
    await user.click(screen.getByRole('button', { name: 'Quarterly' }));

    const headers = screen.getAllByRole('columnheader').map((header) => header.textContent);
    expect(headers).toEqual(['Row', 'Q1', 'Q2', 'Q3', 'Q4', 'Total']);
    expect(screen.getAllByRole('row')[1]).toHaveTextContent('150,000.00');
  });

  it('marks a period over its budget in red with words, and shows Other years (FR-024)', async () => {
    const over = Array(12).fill(false);
    over[2] = true;
    const plannedCost = twelve('40000.00') as { amount: string; currency: string }[];
    plannedCost[2] = gbp('65000.00');
    renderCard(undefined, {
      'GET /api/v1/plans/P-1004/budget-by-month': {
        body: budgetByMonth({
          rows: [
            {
              ...budgetByMonth().rows[0],
              months: periods({ plannedCost, over }),
              otherYears: gbp('5000.00'),
            },
          ],
        }),
      },
    });

    const cell = await screen.findByText('Over by 15,000.00');
    expect(cell.className).toMatch(/over/);
    expect(screen.getByRole('columnheader', { name: 'Other years' })).toBeInTheDocument();
  });

  it('says Not phased, and the needed-to-publish problem of a Custom phasing (FR-022)', async () => {
    renderCard(
      {
        ...PLAN,
        id: 'C-1005',
        name: 'Helix UK',
        budgetPhasing: phasing({
          method: 'custom',
          problem: 'Adds up to 590,000.00 GBP, not the budget 600,000.00 GBP',
        }),
      },
      {},
      'campaign',
    );

    expect(
      screen.getByText(
        'Budget by month: Adds up to 590,000.00 GBP, not the budget 600,000.00 GBP. Needed to publish.',
      ),
    ).toBeInTheDocument();
    expect(await screen.findByText('Not phased')).toBeInTheDocument();
  });

  it('saves a custom phasing typed by quarter through the level’s PUT (FR-020, FR-021, FR-023)', async () => {
    const { user, api } = renderCard(
      { ...PLAN, budgetPhasing: phasing({ method: 'none', monthly: [], total: null }) },
      {
        'PUT /api/v1/plans/P-1004/budget-phasing': { body: {} },
      },
    );

    await screen.findByRole('table');
    await user.click(screen.getByRole('button', { name: 'Edit budget by month' }));
    const dialog = await screen.findByRole('dialog', {
      name: 'Budget by month — FY2027 Lubricants UK & Ireland',
    });
    await user.click(within(dialog).getByRole('radio', { name: /Custom/ }));
    await user.click(within(dialog).getByRole('button', { name: 'Split evenly' }));
    expect(
      within(dialog).getByText('600,000.00 of 600,000.00 GBP phased — all of it'),
    ).toBeInTheDocument();

    await user.click(within(dialog).getByRole('button', { name: 'Quarterly' }));
    const q1 = within(dialog).getByRole('textbox', { name: 'Q1 2027' });
    await user.clear(q1);
    await user.type(q1, '140000');
    expect(
      within(dialog).getByText('590,000.00 of 600,000.00 GBP phased — 10,000.00 left'),
    ).toBeInTheDocument();
    await user.click(within(dialog).getByRole('button', { name: 'Save budget by month' }));

    await waitFor(() =>
      expect(api.bodiesOf('PUT /api/v1/plans/P-1004/budget-phasing')).toHaveLength(1),
    );
    const [body] = api.bodiesOf('PUT /api/v1/plans/P-1004/budget-phasing') as {
      method: string;
      months: { month: string; amount: string }[];
      version: number;
    }[];
    expect(body.method).toBe('custom');
    expect(body.version).toBe(4);
    expect(body.months.slice(0, 3)).toEqual([
      { month: '2027-01', amount: '46666.66' },
      { month: '2027-02', amount: '46666.66' },
      { month: '2027-03', amount: '46666.68' },
    ]);
    expect(body.months).toHaveLength(12);
    expect(await screen.findByText(/Budget by month of .* saved/)).toBeInTheDocument();
  });

  it('copies the planned cost by month and names what it leaves out (FR-021)', async () => {
    const { user } = renderCard(
      { ...PLAN, id: 'C-1005', name: 'Helix UK', budgetPhasing: phasing({ method: 'none' }) },
      {},
      'campaign',
    );

    await screen.findByRole('table');
    await user.click(screen.getByRole('button', { name: 'Edit budget by month' }));
    const dialog = await screen.findByRole('dialog');
    await user.click(within(dialog).getByRole('radio', { name: /Custom/ }));
    await user.click(within(dialog).getByRole('button', { name: 'Copy from planned cost' }));

    expect(within(dialog).getByRole('textbox', { name: 'Jan 2027' })).toHaveValue('40,000.00');
    expect(within(dialog).getByText('Left out: 5,000.00 GBP in other years.')).toBeInTheDocument();
  });

  it('comes into view with the focus when a completeness link leads to it (T053)', async () => {
    mockApi({ 'GET /api/v1/plans/P-1004/budget-by-month': { body: budgetByMonth() } });
    renderWithProviders(
      <BudgetByMonthCard
        level="plan"
        planId="P-1004"
        record={{ ...PLAN, budgetPhasing: phasing() }}
        year={2027}
        currency="GBP"
      />,
      { user: testUser(), route: '/plans/P-1004#budget-by-month-P-1004', path: '/plans/:planId' },
    );

    const card = screen.getByRole('region', { name: 'Budget by month' });
    expect(card).toHaveAttribute('id', 'budget-by-month-P-1004');
    await waitFor(() => expect(card).toHaveFocus());
  });
});
