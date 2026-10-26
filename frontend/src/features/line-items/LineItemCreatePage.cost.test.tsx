import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { formOptions } from '@/testing/fixtures/line-item-forms';
import { findNameField, renderCreatePage } from '@/testing/fixtures/line-item-pages';
import { reply } from '@/testing/replies';

describe('LineItemCreatePage, cost and currency (US2)', () => {
  it('converts the balance when the currency changes and shows the USD amount live (scenario 2, FR-011, FR-012)', async () => {
    const { user } = renderCreatePage();
    await findNameField();

    await user.selectOptions(screen.getByLabelText('Currency'), 'EUR');
    const cost = screen.getByLabelText('Planned cost');
    expect(cost).toHaveValue('18,400.00');
    expect(screen.getByText('Changed to 18,400.00 EUR = 20,000.00 USD')).toBeInTheDocument();

    await user.clear(cost);
    await user.type(cost, '10000');
    expect(
      await screen.findByText(/^= 10,869\.57 USD at 0\.92 EUR per USD \(rate of 0?1 Mar 2026\)$/),
    ).toBeInTheDocument();
    expect(screen.getByText('Calculated')).toBeInTheDocument();
    expect(screen.getByLabelText('Activity summary')).toHaveTextContent('After saving9,130.43 USD');
  });

  it('calculates in the plan currency of a GBP plan, with both rates and their date (016 FR-009, FR-015)', async () => {
    const gbp = (amount: string) => ({ amount, currency: 'GBP' });
    const { user } = renderCreatePage([
      reply(
        'GET /api/v1/activities/{activityId}/line-items/form-options',
        {
          body: formOptions({
            balance: {
              plannedCost: gbp('40000.00'),
              assigned: gbp('15000.00'),
              balance: gbp('25000.00'),
              state: 'left',
              roundingHint: false,
            },
            defaults: {
              ...formOptions().defaults,
              currency: 'GBP',
              plannedCost: gbp('25000.00'),
            },
          }),
        },
        { activityId: 'A-1001' },
      ),
    ]);
    await findNameField();

    // The pre-filled balance keeps its plan-currency value in another currency.
    await user.selectOptions(screen.getByLabelText('Currency'), 'EUR');
    const cost = screen.getByLabelText('Planned cost');
    expect(cost).toHaveValue('29,113.92');
    expect(screen.getByText('Changed to 29,113.92 EUR = 25,000.00 GBP')).toBeInTheDocument();

    await user.clear(cost);
    await user.type(cost, '23000');
    expect(
      await screen.findByText(
        /^= 19,750\.00 GBP at 0\.92 EUR and 0\.79 GBP per USD \(rates of 0?1 Mar 2026\)$/,
      ),
    ).toBeInTheDocument();
    expect(screen.getByLabelText('Activity summary')).toHaveTextContent('After saving5,250.00 GBP');
  });
});
