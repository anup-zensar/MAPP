import type { CurrentUser } from '@/app/useSession';

/**
 * A complete signed-in person for tests (feature 003, T022): Priya Raman, a member of Lubricants
 * Europe, with every preference set to its default. Pass only what a test needs to differ:
 *
 *   testUser({ preferences: { ...testUser().preferences, sidebarCollapsed: true } })
 */
export function testUser(overrides: Partial<CurrentUser> = {}): CurrentUser {
  return {
    id: 'U-1002',
    name: 'Priya Raman',
    firstName: 'Priya',
    email: 'priya.raman@shell.com',
    jobTitle: 'Marketer',
    initials: 'PR',
    version: 1,
    teams: [{ id: 'T-1001', name: 'Lubricants Europe', colour: 'blue', active: true }],
    preferences: {
      theme: 'system',
      sidebarCollapsed: false,
      displayCurrency: 'USD',
      dateFormat: 'DD MMM YYYY',
      pinnedPlanIds: [],
    },
    ...overrides,
  };
}
