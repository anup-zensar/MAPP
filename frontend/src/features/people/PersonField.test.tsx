import { screen } from '@testing-library/react';
import { useState } from 'react';
import { describe, expect, it } from 'vitest';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { reply } from '@/testing/replies';
import { PersonField } from './PersonField';

const EUROPE = { id: 'T-1001', name: 'Lubricants Europe', colour: 'blue', active: true } as const;
const CHINA = { id: 'T-1002', name: 'Mobility China', colour: 'teal', active: true } as const;
const META = {
  version: 1,
  createdAt: '2026-03-01T09:00:00Z',
  createdBy: { id: 'U-1001', name: 'Alex Morgan' },
  updatedAt: '2026-03-01T09:00:00Z',
  updatedBy: { id: 'U-1001', name: 'Alex Morgan' },
};

function person(id: string, name: string, jobTitle: string, teams = [EUROPE], active = true) {
  const email = `${name.toLowerCase().replace(' ', '.')}@shell.com`;
  const initials = name
    .split(' ')
    .map((part) => part[0])
    .join('');
  return { id, name, email, jobTitle, initials, active, teams, ...META };
}

const PEOPLE = {
  total: 2,
  items: [
    person('U-1001', 'Alex Morgan', 'Head of marketing'),
    person('U-1002', 'Priya Raman', 'Marketer'),
  ],
};

function Owner({ teamIds }: { teamIds?: string[] }) {
  const [owner, setOwner] = useState('');
  return (
    <PersonField label="Owner" name="ownerId" teamIds={teamIds} value={owner} onChange={setOwner} />
  );
}

function Members({ start = [] }: { start?: string[] }) {
  const [members, setMembers] = useState<string[]>(start);
  return (
    <>
      <PersonField
        label="Members"
        name="memberIds"
        multiple
        value={members}
        onChange={setMembers}
      />
      <p>chosen: {members.join(', ')}</p>
    </>
  );
}

describe('PersonField (feature 003, US3)', () => {
  it('loads the people and offers them, the signed-in person first', async () => {
    mockApi([reply('GET /api/v1/people', { body: PEOPLE })]);
    const { user } = renderWithProviders(<Owner />, { user: testUser() });

    expect(screen.getByRole('status')).toHaveTextContent('Loading…');
    await user.click(await screen.findByRole('combobox', { name: 'Owner' }));

    expect(screen.getAllByRole('option')[0]).toHaveTextContent('Priya Raman (you)');
  });

  it('says when the people could not be loaded, with Try again', async () => {
    let calls = 0;
    mockApi([
      reply('GET /api/v1/people', () => {
        calls += 1;
        return calls === 1 ? { status: 500 } : { body: PEOPLE };
      }),
    ]);
    const { user } = renderWithProviders(<Owner />, { user: testUser() });

    expect(await screen.findByText("People couldn't be loaded.")).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Try again' }));

    expect(await screen.findByRole('combobox', { name: 'Owner' })).toBeInTheDocument();
  });

  it('lists 300 people, every one reachable, and narrows as the user types (FR-020, SC-005)', async () => {
    const many = Array.from({ length: 300 }, (_, index) =>
      person(`U-${2000 + index}`, `Person ${String(index).padStart(3, '0')}`, 'Marketer'),
    );
    mockApi([reply('GET /api/v1/people', { body: { items: many, total: many.length } })]);
    const { user } = renderWithProviders(<Owner />, { user: testUser() });

    await user.click(await screen.findByRole('combobox', { name: 'Owner' }));

    expect(screen.getByRole('status')).toHaveTextContent('300 people');
    expect(screen.getAllByRole('option')).toHaveLength(300);
    await user.type(screen.getByRole('combobox', { name: 'Owner' }), 'Person 29');
    expect(screen.getByRole('status')).toHaveTextContent('10 people');
  });

  it('in team mode says when no member of the chosen teams matches (FR-019)', async () => {
    mockApi([reply('GET /api/v1/people', { body: PEOPLE })]);
    const { user } = renderWithProviders(<Owner teamIds={[CHINA.id]} />, { user: testUser() });

    await user.click(await screen.findByRole('combobox', { name: 'Owner' }));

    expect(
      screen.getByText(
        'No members of the selected teams match. Add people to these teams in Settings › Teams.',
      ),
    ).toBeInTheDocument();
  });

  it('chooses several people with the keyboard and removes the last with Backspace (US3-7)', async () => {
    mockApi([reply('GET /api/v1/people', { body: PEOPLE })]);
    const { user } = renderWithProviders(<Members />, { user: testUser() });

    await user.click(await screen.findByRole('combobox', { name: 'Members' }));
    await user.keyboard('{Enter}{ArrowDown}{Enter}{Escape}');

    expect(screen.getByText('chosen: U-1002, U-1001')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Remove Alex Morgan' })).toBeInTheDocument();

    await user.keyboard('{Backspace}');
    expect(screen.getByText('chosen: U-1002')).toBeInTheDocument();
  });

  it('keeps an inactive person already chosen, labelled (FR-018, SC-003)', async () => {
    const inactive = {
      ...PEOPLE,
      items: [...PEOPLE.items, person('U-1009', 'Old Timer', '', [EUROPE], false)],
    };
    mockApi([reply('GET /api/v1/people', { body: inactive })]);
    const { user } = renderWithProviders(<Members start={['U-1009']} />, { user: testUser() });

    expect(await screen.findByText('Old Timer (inactive)')).toBeInTheDocument();
    await user.click(screen.getByRole('combobox', { name: 'Members' }));
    expect(screen.queryByRole('option', { name: /Old Timer/ })).not.toBeInTheDocument();
  });
});
