import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { Sidebar } from '@/app/Sidebar';
import type { CurrentUser } from '@/session/useSession';
import { mockApi } from '@/testing/mockApi';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/users';
import { ModelsPage } from '../models/ModelsPage';
import { SettingsPage } from '../SettingsPage';
import { ACTIVITY_TYPES, MODELS } from '../testRecords';

const ALEX = {
  id: 'U-1001',
  name: 'Alex Morgan',
  initials: 'AM',
  jobTitle: 'Marketing administrator',
  active: true,
};

/** Sara Novak, a Viewer of her teams: she sees plans and nothing else (feature 014). */
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
    permissions: [{ key: 'plans.view', all: false, teams: [] }],
    administrators: [ALEX],
    noSelfApproval: true,
  },
});

const VIEW_ONLY =
  'View only — only administrators change Settings. Ask Alex Morgan if something needs to change.';

describe('Light access gating (feature 014, FR-011, FR-012)', () => {
  it('leaves out the sidebar links the person’s roles do not open', () => {
    renderWithProviders(<Sidebar />, { user: SARA });

    const links = within(screen.getByRole('navigation', { name: 'Main' })).getAllByRole('link');
    expect(links.map((link) => link.textContent?.trim())).toEqual([
      'Home',
      'Plans',
      'Approvals',
      'All work',
      'Settings',
      'Recycle bin',
      'Activity log',
    ]);
  });

  it('leaves out the Roles and access card for people who may not give roles', async () => {
    // The counts never answer here: only the cards matter.
    const never = () => new Promise<never>(() => {});
    mockApi({
      'GET /api/v1/settings/overview': never,
      'GET /api/v1/field-help': never,
      'GET /api/v1/people': never,
      'GET /api/v1/teams': never,
      'GET /api/v1/approval-workflows': never,
    });
    renderWithProviders(<SettingsPage />, { user: SARA });

    expect(await screen.findByRole('link', { name: 'Models' })).toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'Roles and access' })).not.toBeInTheDocument();
  });

  it('shows a Settings list read-only, with one View only note and no change actions', async () => {
    mockApi({
      'GET /api/v1/models': { body: MODELS },
      'GET /api/v1/activity-types': { body: ACTIVITY_TYPES },
    });
    const { user } = renderWithProviders(<ModelsPage />, {
      route: '/settings/models',
      path: '/settings/models',
      user: SARA,
    });

    await screen.findByRole('table');
    expect(screen.getAllByRole('note')).toHaveLength(1);
    expect(screen.getByRole('note')).toHaveTextContent(VIEW_ONLY);
    expect(screen.queryByRole('button', { name: 'New model' })).not.toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Actions for Lubricants 1.0' }));
    const items = await screen.findAllByRole('menuitem');
    expect(items.map((item) => item.textContent)).toEqual(['View']);
  });

  it('keeps every change action for people who may change Settings', async () => {
    mockApi({
      'GET /api/v1/models': { body: MODELS },
      'GET /api/v1/activity-types': { body: ACTIVITY_TYPES },
    });
    renderWithProviders(<ModelsPage />, {
      route: '/settings/models',
      path: '/settings/models',
      user: testUser(),
    });

    await screen.findByRole('table');
    expect(screen.getByRole('button', { name: 'New model' })).toBeInTheDocument();
    expect(screen.queryByRole('note')).not.toBeInTheDocument();
  });
});
