import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { CurrentUser } from '@/session/useSession';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { ViewOnlyNote } from './ViewOnlyNote';
import { viewOnlyText } from './view-only-text';

const ALEX = {
  id: 'U-1001',
  name: 'Alex Morgan',
  initials: 'AM',
  jobTitle: 'Marketing administrator',
  active: true,
};

/** Sara Novak, a Viewer of her teams. */
const SARA: CurrentUser = testUser({
  id: 'U-1006',
  name: 'Sara Novak',
  access: {
    roles: [
      {
        id: 'RA-1007',
        role: { id: 'RL-1005', name: 'Viewer', builtIn: 'viewer' },
        scope: { type: 'myTeams', teams: [], text: 'My teams' },
        scopeEmpty: false,
      },
    ],
    permissions: [],
    administrators: [ALEX],
    noSelfApproval: true,
  },
});

describe('ViewOnlyNote', () => {
  it('says the person’s roles let them see the plan, and whom to ask (feature 014, FR-011, message 33)', () => {
    renderWithProviders(<ViewOnlyNote area="plan" />, { user: SARA });

    expect(screen.getByRole('note')).toHaveTextContent(
      'View only — your role (Viewer) lets you see this plan but not change it. Ask an administrator (Alex Morgan) if you need to change it.',
    );
  });

  it('has a sentence for each area (messages 33–35)', () => {
    expect(viewOnlyText('plan', ['Approver', 'Planner'], ['Alex Morgan', 'Nina Berg'])).toBe(
      'View only — your roles (Approver, Planner) let you see this plan but not change it. Ask an administrator (Alex Morgan or Nina Berg) if you need to change it.',
    );
    expect(viewOnlyText('settings', ['Viewer'], ['Alex Morgan'])).toBe(
      'View only — only administrators change Settings. Ask Alex Morgan if something needs to change.',
    );
    expect(viewOnlyText('globalBudget', ['Planner'], ['Alex Morgan'])).toBe(
      'View only — you can see this global budget but not change it.',
    );
    expect(viewOnlyText('record', [], [])).toBe(
      'View only — you can see this record but not change it. Ask an administrator if you need to change it.',
    );
  });
});
