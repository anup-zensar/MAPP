import { screen } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/testing/render';
import { TeamPicker } from './TeamPicker';

const TEAMS = [
  { id: 'T-1001', name: 'Lubricants Europe', colour: 'blue', active: true },
  { id: 'T-1002', name: 'Mobility China', colour: 'teal', active: true },
  { id: 'T-1003', name: 'Fleet Americas', colour: 'gold', active: false },
] as const;

function Example({ start = [], offered }: { start?: string[]; offered?: string[] }) {
  const [value, setValue] = useState(start);
  return (
    <>
      <TeamPicker
        label="Teams"
        name="teamIds"
        teams={[...TEAMS]}
        value={value}
        onChange={setValue}
        offered={offered}
      />
      <p>chosen: {value.join(',')}</p>
    </>
  );
}

describe('TeamPicker', () => {
  it('offers active teams and chooses several (feature 003)', async () => {
    const { user } = renderWithProviders(<Example />);
    const field = screen.getByRole('combobox', { name: 'Teams' });

    await user.click(field);
    expect(screen.getAllByRole('option').map((option) => option.textContent)).toEqual([
      'Lubricants Europe',
      'Mobility China',
    ]);
    expect(screen.getByRole('status')).toHaveTextContent('2 teams');

    await user.click(screen.getByRole('option', { name: 'Mobility China' }));
    await user.click(screen.getByRole('option', { name: 'Lubricants Europe' }));
    expect(screen.getByText('chosen: T-1002,T-1001')).toBeInTheDocument();
  });

  it('keeps an inactive team already chosen, explained', () => {
    renderWithProviders(<Example start={['T-1003']} />);

    expect(screen.getByText('Fleet Americas (inactive)')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'About Fleet Americas' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Remove Fleet Americas' })).toBeInTheDocument();
  });

  it('says when no team matches', async () => {
    const { user } = renderWithProviders(<Example />);

    await user.type(screen.getByRole('combobox', { name: 'Teams' }), 'zzz');

    expect(screen.getByText("No team matches 'zzz'.")).toBeInTheDocument();
  });

  it('offers only the person’s own teams and keeps the plan’s other teams (014 FR-013)', async () => {
    const { user } = renderWithProviders(
      <Example start={['T-1002', 'T-1001']} offered={['T-1001']} />,
    );

    expect(screen.getByText('Kept, outside your teams: Mobility China.')).toBeVisible();
    expect(screen.queryByRole('button', { name: 'Remove Mobility China' })).not.toBeInTheDocument();
    await user.click(screen.getByRole('combobox', { name: 'Teams' }));
    expect(screen.getAllByRole('option').map((option) => option.textContent)).toEqual([
      'Lubricants Europe',
    ]);

    await user.click(screen.getByRole('button', { name: 'Remove Lubricants Europe' }));
    expect(screen.getByText('chosen: T-1002')).toBeInTheDocument();
  });
});
