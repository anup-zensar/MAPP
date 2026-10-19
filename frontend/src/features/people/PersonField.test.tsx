import { screen } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it } from 'vitest';
import { mockApi } from '@/test/mockApi';
import { renderWithProviders } from '@/test/render';
import { testUser } from '@/test/users';
import { PersonField } from './PersonField';
import { TeamField } from './TeamField';

const EUROPE = { id: 'T-1001', name: 'Lubricants Europe', colour: 'blue', active: true } as const;
const META = {
  version: 1,
  createdAt: '2026-03-01T09:00:00Z',
  createdBy: { id: 'U-1001', name: 'Alex Morgan' },
  updatedAt: '2026-03-01T09:00:00Z',
  updatedBy: { id: 'U-1001', name: 'Alex Morgan' },
};
const PEOPLE = {
  total: 2,
  items: [
    {
      id: 'U-1001',
      name: 'Alex Morgan',
      email: 'alex.morgan@northstar.demo',
      jobTitle: 'Head of marketing',
      initials: 'AM',
      active: true,
      teams: [EUROPE],
      ...META,
    },
    {
      id: 'U-1002',
      name: 'Priya Raman',
      email: 'priya.raman@northstar.demo',
      jobTitle: 'Marketer',
      initials: 'PR',
      active: true,
      teams: [EUROPE],
      ...META,
    },
  ],
};
const TEAMS = {
  total: 1,
  items: [{ ...EUROPE, description: '', members: [], plans: [], ...META }],
};

function Owner() {
  const [owner, setOwner] = useState('');
  return <PersonField label="Owner" name="ownerId" value={owner} onChange={setOwner} />;
}

function Teams() {
  const [teams, setTeams] = useState<string[]>([]);
  return <TeamField label="Teams" name="teamIds" value={teams} onChange={setTeams} />;
}

describe('PersonField and TeamField (feature 003, US3)', () => {
  it('loads the people and offers them, the signed-in person first', async () => {
    mockApi({ 'GET /api/people': { body: PEOPLE } });
    const { user } = renderWithProviders(<Owner />, { user: testUser() });

    expect(screen.getByRole('status')).toHaveTextContent('Loading…');
    await user.click(await screen.findByRole('combobox', { name: 'Owner' }));

    expect(screen.getAllByRole('option')[0]).toHaveTextContent('Priya Raman (you)');
  });

  it('says when the people could not be loaded, with Try again', async () => {
    let calls = 0;
    mockApi({
      'GET /api/people': () => {
        calls += 1;
        return calls === 1 ? { status: 500 } : { body: PEOPLE };
      },
    });
    const { user } = renderWithProviders(<Owner />, { user: testUser() });

    expect(await screen.findByText("People couldn't be loaded.")).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Try again' }));

    expect(await screen.findByRole('combobox', { name: 'Owner' })).toBeInTheDocument();
  });

  it('loads the teams for a teams field', async () => {
    mockApi({ 'GET /api/teams': { body: TEAMS } });
    const { user } = renderWithProviders(<Teams />, { user: testUser() });

    await user.click(await screen.findByRole('combobox', { name: 'Teams' }));

    expect(screen.getByRole('option', { name: 'Lubricants Europe' })).toBeInTheDocument();
  });
});
