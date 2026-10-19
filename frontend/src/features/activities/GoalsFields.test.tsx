import { screen } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/test/render';
import { emptyGoals } from './activityFormState';
import type { GoalsState } from './activityFormState';
import { GoalsFields } from './GoalsFields';
import { MODEL } from './testData';

function Goals({ start = emptyGoals() }: { start?: GoalsState }) {
  const [goals, setGoals] = useState(start);
  return <GoalsFields value={goals} onChange={setGoals} model={MODEL} errors={{}} />;
}

function stage(label: string): HTMLElement {
  return screen.getByText(label, { selector: 'dt' }).nextElementSibling as HTMLElement;
}

describe('GoalsFields (feature 006, FR-018)', () => {
  it('calculates MQL and revenue from INQ with the model, labelled Calculated with the formula', async () => {
    const { user } = renderWithProviders(<Goals />);

    expect(stage('MQL')).toHaveTextContent('—');
    expect(stage('Revenue')).toHaveTextContent('—');

    await user.type(screen.getByLabelText('INQ'), '250,000');

    expect(stage('MQL')).toHaveTextContent('5,000Calculated');
    expect(stage('Revenue')).toHaveTextContent('$2,500,000.00Calculated');
    await user.click(screen.getByRole('button', { name: 'About MQL calculation' }));
    expect(
      await screen.findByText(
        "MQL = INQ × 2.00 % · deals = MQL × 5.00 % · revenue = deals × 10,000.00 USD — each rounded to a whole number (model 'Lubricants 1.0').",
      ),
    ).toBeVisible();
  });

  it.each(['-5', '12.5'])('says INQ "%s" must be a whole number of 0 or more', async (typed) => {
    const { user } = renderWithProviders(<Goals />);

    await user.type(screen.getByLabelText('INQ'), typed);

    expect(screen.getByLabelText('INQ')).toHaveAccessibleDescription(
      'Enter a whole number of 0 or more.',
    );
    expect(stage('MQL')).toHaveTextContent('—');
  });

  it('lets later stages be typed, keeping them when INQ changes', async () => {
    const { user } = renderWithProviders(<Goals />);

    await user.type(screen.getByLabelText('INQ'), '250000');
    await user.click(screen.getByLabelText('Set later stages myself'));
    expect(screen.getByLabelText('MQL')).toHaveValue('5,000');
    expect(screen.getByLabelText('Revenue')).toHaveValue('2,500,000.00');

    await user.clear(screen.getByLabelText('MQL'));
    await user.type(screen.getByLabelText('MQL'), '6,000');
    await user.clear(screen.getByLabelText('INQ'));
    await user.type(screen.getByLabelText('INQ'), '100');

    expect(screen.getByLabelText('MQL')).toHaveValue('6,000');
  });

  it('lists the typed values that saving will replace when typing is turned off', async () => {
    const { user } = renderWithProviders(
      <Goals
        start={{
          inq: '250,000',
          setLaterStagesMyself: true,
          mql: '6,000',
          revenue: '3,000,000.00',
        }}
      />,
    );

    await user.click(screen.getByLabelText('Set later stages myself'));

    expect(
      screen.getByText(
        'These typed values will be replaced when you save: MQL 6,000 → 5,000; Revenue $3,000,000.00 → $2,500,000.00.',
      ),
    ).toBeVisible();
  });
});
