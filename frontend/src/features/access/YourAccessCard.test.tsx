import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/test/render';
import { testUser } from '@/test/users';
import { YourAccessCard } from './YourAccessCard';
import { ALEX, assignment, EUROPE, MY_TEAMS_EUROPE, PLANNER, PRIYA, SARA } from './testData';

const PRIYA_ACCESS = {
  roles: [assignment('RA-1004', PLANNER, MY_TEAMS_EUROPE)],
  permissions: [
    { key: 'plans.view' as const, all: false, teams: [EUROPE] },
    { key: 'plans.create' as const, all: false, teams: [EUROPE] },
  ],
  administrators: [ALEX],
  noSelfApproval: true,
};

describe('YourAccessCard (feature 014, FR-024)', () => {
  it('shows your roles, what they allow and whom to ask', () => {
    renderWithProviders(<YourAccessCard />, { user: testUser({ access: PRIYA_ACCESS }) });

    const card = screen.getByRole('region', { name: 'Your access' });
    expect(within(card).getByRole('list', { name: 'Your roles' })).toHaveTextContent(
      'Planner · My teams',
    );
    expect(card).toHaveTextContent(
      'In Lubricants Europe: view plans and everything in them and create and copy plans.',
    );
    expect(card).toHaveTextContent('To change your access, ask Alex Morgan.');
  });

  it('says what to do without a role', () => {
    renderWithProviders(<YourAccessCard />, {
      user: testUser({
        id: SARA.id,
        access: { ...PRIYA_ACCESS, roles: [], permissions: [] },
      }),
    });

    expect(
      screen.getByText(
        "You don't have access to any plans yet. Ask an administrator (Alex Morgan) to give you a role.",
      ),
    ).toBeInTheDocument();
  });

  it('points an administrator to Roles and access', () => {
    renderWithProviders(<YourAccessCard />, {
      user: testUser({
        id: ALEX.id,
        access: {
          ...PRIYA_ACCESS,
          permissions: [{ key: 'access.manage', all: true, teams: [] }],
          administrators: [ALEX],
        },
      }),
    });

    expect(screen.getByRole('link', { name: 'Settings › Roles and access' })).toHaveAttribute(
      'href',
      '/settings/access',
    );
    expect(screen.queryByText(/ask Alex Morgan/)).not.toBeInTheDocument();
  });

  it('is left out while the session has no access (older answers)', () => {
    renderWithProviders(<YourAccessCard />, { user: testUser({ id: PRIYA.id }) });

    expect(screen.queryByRole('region', { name: 'Your access' })).not.toBeInTheDocument();
  });
});
