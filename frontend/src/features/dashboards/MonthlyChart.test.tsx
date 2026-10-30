import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { dashboard } from '@/testing/fixtures/dashboards';
import { testUser } from '@/testing/fixtures/users';
import { renderWithProviders } from '@/testing/render';
import { MonthlyChart } from './MonthlyChart';

function renderChart(negativeBalanceCount: number) {
  const { months } = dashboard();
  renderWithProviders(<MonthlyChart months={{ ...months, negativeBalanceCount }} />, {
    user: testUser(),
  });
}

describe('MonthlyChart (036 FR-014)', () => {
  it('says why the months add up to more than planned cost (038 FR-024, L-2)', () => {
    renderChart(1);

    expect(
      screen.getByText(
        '1 activity has line items above its planned cost, so its months add up to more than its planned cost.',
      ),
    ).toBeInTheDocument();
  });

  it('says nothing of it when no activity is above its planned cost (038 FR-024)', () => {
    renderChart(0);

    expect(screen.queryByText(/months add up to more/)).not.toBeInTheDocument();
  });
});
