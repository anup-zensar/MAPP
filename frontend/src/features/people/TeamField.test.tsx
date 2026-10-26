import { screen } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it } from 'vitest';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { TeamField } from './TeamField';

const META = {
  version: 1,
  createdAt: '2026-03-01T09:00:00Z',
  createdBy: { id: 'U-1001', name: 'Alex Morgan' },
  updatedAt: '2026-03-01T09:00:00Z',
  updatedBy: { id: 'U-1001', name: 'Alex Morgan' },
};

function team(id: string, name: string, active = true) {
  return { id, name, colour: 'blue', active, description: '', members: [], plans: [], ...META };
}

const TEAMS = {
  total: 2,
  items: [team('T-1001', 'Lubricants Europe'), team('T-1003', 'Old Markets', false)],
};

function Teams({ start = [] }: { start?: string[] }) {
  const [teams, setTeams] = useState<string[]>(start);
  return <TeamField label="Teams" name="teamIds" value={teams} onChange={setTeams} />;
}

describe('TeamField (feature 003, US3)', () => {
  it('loads the teams and offers only the active ones', async () => {
    mockApi({ 'GET /api/v1/teams': { body: TEAMS } });
    const { user } = renderWithProviders(<Teams />, { user: testUser() });

    expect(screen.getByRole('status')).toHaveTextContent('Loading…');
    await user.click(await screen.findByRole('combobox', { name: 'Teams' }));

    expect(screen.getAllByRole('option').map((option) => option.textContent)).toEqual([
      'Lubricants Europe',
    ]);
  });

  it('keeps an inactive team already chosen, labelled', async () => {
    mockApi({ 'GET /api/v1/teams': { body: TEAMS } });
    renderWithProviders(<Teams start={['T-1003']} />, { user: testUser() });

    expect(await screen.findByText('Old Markets (inactive)')).toBeInTheDocument();
  });

  it('says when the teams could not be loaded, with Try again', async () => {
    let calls = 0;
    mockApi({
      'GET /api/v1/teams': () => {
        calls += 1;
        return calls === 1 ? { status: 500 } : { body: TEAMS };
      },
    });
    const { user } = renderWithProviders(<Teams />, { user: testUser() });

    expect(await screen.findByText("Teams couldn't be loaded.")).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Try again' }));

    expect(await screen.findByRole('combobox', { name: 'Teams' })).toBeInTheDocument();
  });
});
