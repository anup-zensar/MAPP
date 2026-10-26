import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { RequirePermission } from './RequirePermission';
import type { CurrentUser } from './useSession';

/** Priya Raman with the Planner role's permissions for Lubricants Europe only. */
const PRIYA: CurrentUser = testUser({
  access: {
    roles: [],
    permissions: [
      { key: 'plans.view', all: false, teams: [] },
      { key: 'plans.create', all: false, teams: [] },
    ],
    administrators: [],
    noSelfApproval: true,
  },
});

describe('RequirePermission', () => {
  it('opens the page for a person whose role includes the permission (feature 014, FR-012)', () => {
    renderWithProviders(
      <RequirePermission permission="plans.create">
        <h1>New plan</h1>
      </RequirePermission>,
      { user: PRIYA },
    );

    expect(screen.getByRole('heading', { name: 'New plan' })).toBeInTheDocument();
  });

  it('shows the no-access page to anyone else (FR-012)', () => {
    renderWithProviders(
      <RequirePermission permission="access.manage">
        <h1>Roles and access</h1>
      </RequirePermission>,
      { user: PRIYA },
    );

    expect(screen.queryByRole('heading', { name: 'Roles and access' })).not.toBeInTheDocument();
    expect(screen.getByRole('heading', { level: 1, name: 'No access' })).toBeInTheDocument();
  });

  it('opens every page while the API sends no access (before feature 014)', () => {
    renderWithProviders(
      <RequirePermission permission="access.manage">
        <h1>Roles and access</h1>
      </RequirePermission>,
      { user: testUser() },
    );

    expect(screen.getByRole('heading', { name: 'Roles and access' })).toBeInTheDocument();
  });
});
