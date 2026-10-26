import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/testing/render';
import { GoalFields } from './GoalFields';
import { goalRows } from './goalWaterfall';
import { model } from '@/testing/fixtures/plans';

const LUBRICANTS_1 = model();

function rowsText() {
  const list = document.querySelector('dl') as HTMLElement;
  return within(list)
    .getAllByRole('term')
    .map((term) => `${term.textContent}: ${term.nextElementSibling?.textContent}`);
}

describe('GoalFields (FR-009)', () => {
  it('works forwards from INQ 250,000: MQL 5,000, deals 250 and revenue 2,500,000.00 USD, calculated', () => {
    renderWithProviders(
      <GoalFields
        stage="inq"
        amount="250,000"
        onStageChange={() => {}}
        onAmountChange={() => {}}
        model={LUBRICANTS_1}
      />,
    );

    expect(screen.getByRole('textbox', { name: 'Goal amount (INQ)' })).toHaveValue('250,000');
    expect(rowsText()).toEqual([
      'MQL: 5,000Calculated MQL = INQ × 2.00 %',
      'Deals: 250Calculated Deals = MQL × 5.00 %',
      'Revenue: 2,500,000.00 USDCalculated Revenue = deals × 10,000.00 USD',
    ]);
  });

  it('works backwards from Revenue', () => {
    expect(goalRows('revenue', '2,500,000.00', LUBRICANTS_1)).toEqual([
      {
        stage: 'inq',
        value: '250,000',
        calculated: true,
        formula: 'INQ = MQL ÷ 2.00 %',
        reason: undefined,
      },
      {
        stage: 'mql',
        value: '5,000',
        calculated: true,
        formula: 'MQL = deals ÷ 5.00 %',
        reason: undefined,
      },
      {
        stage: 'deals',
        value: '250',
        calculated: true,
        formula: 'Deals = revenue ÷ 10,000.00 USD',
        reason: undefined,
      },
      { stage: 'revenue', value: '2,500,000.00 USD', calculated: false },
    ]);
  });

  it('works partly backwards from MQL, rounding half up', () => {
    const rows = goalRows('mql', '5,001', LUBRICANTS_1);
    expect(rows.map((row) => row.value)).toEqual(['250,050', '5,001', '250', '2,500,000.00 USD']);
  });

  it('says why a stage cannot be worked out when a rate is 0', () => {
    const rows = goalRows('mql', '5000', model({ inqToMqlRate: '0.00' }));
    expect(rows[0]).toMatchObject({
      value: null,
      reason: "The model's INQ→MQL rate is 0, so this can't be worked out.",
    });
    const fromRevenue = goalRows(
      'revenue',
      '1000',
      model({ averageDealSize: { amount: '0.00', currency: 'USD' } }),
    );
    expect(fromRevenue[2].reason).toBe(
      "The model's average deal size is 0, so this can't be worked out.",
    );
  });

  it('asks for a model before calculating', () => {
    renderWithProviders(
      <GoalFields
        stage="revenue"
        amount="1000"
        onStageChange={() => {}}
        onAmountChange={() => {}}
        model={null}
      />,
    );

    expect(screen.getByRole('textbox', { name: 'Goal amount (Revenue, USD)' })).toBeInTheDocument();
    expect(screen.getAllByText('Choose a model to calculate')).toHaveLength(3);
  });
});
