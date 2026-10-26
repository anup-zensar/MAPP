import { describe, expect, it } from 'vitest';
import { ApiError } from '@/api/errors';
import {
  accessOf,
  availabilityProps,
  forbiddenMessage,
  isShown,
  itemsFor,
  joinWith,
  summariseAccess,
} from './access';
import type { MyAccess, MyPermission } from './access';

const LUBRICANTS = {
  id: 'T-1001',
  name: 'Lubricants Europe',
  colour: 'blue' as const,
  active: true,
};
const MOBILITY = { id: 'T-1002', name: 'Mobility China', colour: 'violet' as const, active: true };
const ALEX = {
  id: 'U-1001',
  name: 'Alex Morgan',
  initials: 'AM',
  jobTitle: 'Marketing administrator',
  active: true,
};

/** Sara Novak as the seed has her: Viewer for her two teams. */
const SARA: MyAccess = {
  roles: [
    {
      id: 'RA-1007',
      role: { id: 'RL-1005', name: 'Viewer', builtIn: 'viewer' },
      scope: { type: 'myTeams', teams: [LUBRICANTS, MOBILITY], text: 'My teams' },
      scopeEmpty: false,
    },
  ],
  permissions: [
    { key: 'plans.view', all: false, teams: [LUBRICANTS, MOBILITY] },
    { key: 'data.export', all: false, teams: [LUBRICANTS, MOBILITY] },
    { key: 'finance.view', all: false, teams: [LUBRICANTS, MOBILITY] },
  ],
  administrators: [ALEX],
  noSelfApproval: true,
};

describe('accessOf (useAccess)', () => {
  it('answers from the person’s merged permissions (FR-012, FR-024)', () => {
    const access = accessOf(SARA);

    expect(access.known).toBe(true);
    expect(access.has('plans.view')).toBe(true);
    expect(access.has('activities.edit')).toBe(false);
    expect(access.scopeOf('plans.view')).toEqual({ all: false, teams: [LUBRICANTS, MOBILITY] });
    expect(access.scopeOf('settings.manage')).toBeNull();
    expect(access.roles.map((r) => r.role.name)).toEqual(['Viewer']);
    expect(access.adminNames()).toBe('Alex Morgan');
    expect(access.rulesNoSelfApproval).toBe(true);
  });

  it('allows everything while GET /api/v1/me sends no access (an API from before feature 014, D-298)', () => {
    const access = accessOf(undefined);

    expect(access.known).toBe(false);
    expect(access.has('access.manage')).toBe(true);
    expect(access.scopeOf('plans.edit')).toEqual({ all: true, teams: [] });
    expect(access.adminNames()).toBe('');
  });

  it('names several administrators with "or" (FR-011)', () => {
    const access = accessOf({
      ...SARA,
      administrators: [ALEX, { ...ALEX, id: 'U-1009', name: 'Nina Berg', initials: 'NB' }],
    });

    expect(access.adminNames()).toBe('Alex Morgan or Nina Berg');
  });
});

describe('availabilityProps and isShown', () => {
  it('disables an unavailable action and gives its reason as the hint (FR-011)', () => {
    const reason =
      "Deleting plans isn't part of your role (Planner). Ask an administrator if you need it.";

    expect(availabilityProps({ available: false, reason })).toEqual({
      disabled: true,
      hint: reason,
    });
    expect(availabilityProps({ available: true, reason: '' })).toEqual({
      disabled: false,
      hint: undefined,
    });
    expect(availabilityProps(undefined)).toEqual({ disabled: false, hint: undefined });
  });

  it('hides an unavailable action without a reason and shows the others (FR-011, 008)', () => {
    expect(isShown({ available: false, reason: '' })).toBe(false);
    expect(
      isShown({
        available: false,
        reason: 'You own this activity, so someone else must approve it.',
      }),
    ).toBe(true);
    expect(isShown({ available: true, reason: '' })).toBe(true);
    expect(isShown(undefined)).toBe(true);
  });
});

describe('summariseAccess', () => {
  it('says in one line per place what the scoped permissions allow, in Table A order (FR-017, FR-024)', () => {
    expect(summariseAccess(SARA.permissions)).toEqual([
      'In Lubricants Europe and Mobility China: view plans and everything in them, export from All work and see financial data.',
    ]);
  });

  it('groups by place, then lists what applies everywhere (FR-017)', () => {
    const permissions: MyPermission[] = [
      { key: 'plans.view', all: false, teams: [LUBRICANTS, MOBILITY] },
      { key: 'plans.create', all: false, teams: [MOBILITY] },
      { key: 'activities.edit', all: false, teams: [MOBILITY] },
      { key: 'globalBudgets.view', all: true, teams: [] },
      { key: 'approvals.decide', all: true, teams: [] },
      { key: 'people.manage', all: true, teams: [] },
    ];

    expect(summariseAccess(permissions)).toEqual([
      'In Lubricants Europe and Mobility China: view plans and everything in them.',
      'In Mobility China: create and copy plans and add, edit, move and copy activities and line items.',
      'In all markets: see global budgets and their allocations.',
      'Everywhere: approve or decline when named as an approver and add and change people and teams.',
    ]);
  });

  it('says "No access yet." for a person without a permission (FR-024)', () => {
    expect(summariseAccess([])).toEqual(['No access yet.']);
  });
});

describe('helpers', () => {
  it('joins names as sentences do', () => {
    expect(joinWith([], 'and')).toBe('');
    expect(joinWith(['A'], 'or')).toBe('A');
    expect(joinWith(['A', 'B', 'C'], 'and')).toBe('A, B and C');
  });

  it('finds the server’s sentence of a refusal by role (FR-008)', () => {
    const refusal = new ApiError('forbidden', {
      status: 403,
      message: "You don't have access to this plan. Ask an administrator if you need it.",
    });

    expect(forbiddenMessage(refusal)).toBe(
      "You don't have access to this plan. Ask an administrator if you need it.",
    );
    expect(forbiddenMessage(new ApiError('notFound'))).toBeNull();
    expect(forbiddenMessage(new Error('boom'))).toBeNull();
  });
});

describe('itemsFor', () => {
  it('keeps the sidebar entries and Settings cards the person’s roles open (FR-012)', () => {
    const items = [
      { label: 'Plans' },
      { label: 'Global budgets', permission: 'globalBudgets.view' as const },
      { label: 'Roles and access', permission: 'access.manage' as const },
    ];

    const shown = itemsFor(items, (p) => p === 'globalBudgets.view');

    expect(shown.map((item) => item.label)).toEqual(['Plans', 'Global budgets']);
  });
});
