import { screen } from '@testing-library/react';
import { describe, it } from 'vitest';
import { expectNoAxeViolations } from '@/testing/axe';
import { mockApi } from '@/testing/mock-api';
import { renderWithProviders } from '@/testing/render';
import { testUser } from '@/testing/fixtures/users';
import { AccessPage } from './AccessPage';
import { RoleFormPage } from './RoleFormPage';
import { RolePage } from './RolePage';
import { ACCESS_ANSWERS, ALEX, PLANNER, SARA_ROLES } from '@/testing/fixtures/access';

const ALEX_USER = testUser({ id: ALEX.id, name: ALEX.name, firstName: 'Alex', initials: 'AM' });

function answers() {
  return mockApi({
    'GET /api/v1/me': { body: ALEX_USER },
    ...ACCESS_ANSWERS,
    'GET /api/v1/people/U-1006/roles': { body: SARA_ROLES },
    [`GET /api/v1/roles/${PLANNER.id}`]: { body: PLANNER },
  });
}

// The screens of feature 014 whose API is not served on this branch yet: WCAG 2.2 A and AA.
describe('Roles and access: no axe violations (WCAG 2.2 AA)', () => {
  it.each([
    [
      'the People tab with a person’s roles open',
      '/settings/access?person=U-1006',
      'Roles of Sara Novak',
    ],
    ['the Roles tab', '/settings/access?tab=roles', 'Planner'],
    ['the Rules tab', '/settings/access?tab=rules', 'On — someone else must approve.'],
  ])('%s', async (_state, route, waitFor) => {
    answers();
    renderWithProviders(<AccessPage />, { user: ALEX_USER, route, path: '/settings/access' });

    await screen.findAllByText(waitFor);
    await expectNoAxeViolations();
  });

  it('the role page', async () => {
    answers();
    renderWithProviders(<RolePage />, {
      user: ALEX_USER,
      route: `/settings/access/roles/${PLANNER.id}`,
      path: '/settings/access/roles/:roleId',
    });

    await screen.findByRole('region', { name: 'Administration' });
    await expectNoAxeViolations();
  });

  it('the role form', async () => {
    answers();
    renderWithProviders(<RoleFormPage />, {
      user: ALEX_USER,
      route: `/settings/access/roles/${PLANNER.id}/edit`,
      path: '/settings/access/roles/:roleId/edit',
    });

    await screen.findByRole('textbox', { name: /^Name/ });
    await expectNoAxeViolations();
  });
});
