/**
 * The demo roles and people of feature 014 for tests (spec "User Scenarios": Alex Administrator,
 * Daniel Planner and Approver, Sara Viewer …), shaped as the API answers them.
 */
import type {
  AccessRules,
  PermissionCatalogue,
  PermissionKey,
  PersonRoles,
  Role,
  RoleAssignment,
} from '@/features/people';
import { PERMISSIONS } from '@/features/people';
import { reply } from '../replies';

const META = {
  version: 1,
  createdAt: '2026-10-01T09:00:00Z',
  createdBy: { id: 'system', name: 'MAPP' },
  updatedAt: '2026-10-01T09:00:00Z',
  updatedBy: { id: 'system', name: 'MAPP' },
};

export const EUROPE = {
  id: 'T-1001',
  name: 'Lubricants Europe',
  colour: 'blue' as const,
  active: true,
};
export const CHINA = {
  id: 'T-1002',
  name: 'Mobility China',
  colour: 'teal' as const,
  active: true,
};
export const GLOBAL = {
  id: 'T-1003',
  name: 'Global Marketing Ops',
  colour: 'gold' as const,
  active: true,
};
export const TEAMS = [EUROPE, CHINA, GLOBAL];

export const ALEX = {
  id: 'U-1001',
  name: 'Alex Morgan',
  initials: 'AM',
  jobTitle: 'Marketing administrator',
  active: true,
};
export const DANIEL = {
  id: 'U-1003',
  name: 'Daniel Okoro',
  initials: 'DO',
  jobTitle: 'Marketing team leader',
  active: true,
};
export const PRIYA = {
  id: 'U-1002',
  name: 'Priya Raman',
  initials: 'PR',
  jobTitle: 'Marketer',
  active: true,
};
export const SARA = {
  id: 'U-1006',
  name: 'Sara Novak',
  initials: 'SN',
  jobTitle: 'Local finance advisor',
  active: true,
};

const ALL_KEYS = PERMISSIONS.map((permission) => permission.key);
const PLANNER_KEYS: PermissionKey[] = [
  'plans.view',
  'plans.create',
  'plans.edit',
  'plans.publish',
  'campaigns.edit',
  'activities.edit',
  'activities.submit',
  'data.export',
  'finance.view',
];

function role(overrides: Partial<Role> & Pick<Role, 'id' | 'name'>): Role {
  return {
    description: '',
    builtIn: '',
    permissions: ['plans.view'],
    defaultScope: 'myTeams',
    people: [],
    locks: [],
    ...META,
    ...overrides,
  };
}

export const ADMINISTRATOR = role({
  id: 'RL-1001',
  name: 'Administrator',
  description: 'Changes everything, including Settings and access.',
  builtIn: 'administrator',
  permissions: ALL_KEYS,
  defaultScope: 'all',
  people: [ALEX],
  locks: [
    {
      field: 'delete',
      reason: "The Administrator role always has every permission and can't be changed.",
    },
  ],
});
export const PLANNER = role({
  id: 'RL-1003',
  name: 'Planner',
  description: 'Plans for their own markets.',
  builtIn: 'planner',
  permissions: PLANNER_KEYS,
  people: [DANIEL, PRIYA],
  locks: [
    { field: 'delete', reason: "'Planner' is given to 2 people. Take it away from them first." },
  ],
});
export const APPROVER = role({
  id: 'RL-1004',
  name: 'Approver',
  description: 'Approves activities they are named on.',
  builtIn: 'approver',
  permissions: ['plans.view', 'approvals.decide'],
  people: [DANIEL],
  locks: [
    { field: 'delete', reason: "'Approver' is given to 1 person. Take it away from them first." },
  ],
});
export const VIEWER = role({
  id: 'RL-1005',
  name: 'Viewer',
  description: 'Sees plans of their markets without changing them.',
  builtIn: 'viewer',
  permissions: ['plans.view', 'finance.view'],
  people: [SARA],
  locks: [
    { field: 'delete', reason: "'Viewer' is given to 1 person. Take it away from them first." },
  ],
});
export const FINANCE_REVIEWER = role({
  id: 'RL-1006',
  name: 'Finance reviewer',
  permissions: ['plans.view', 'finance.view'],
});
export const ROLES = [ADMINISTRATOR, APPROVER, FINANCE_REVIEWER, PLANNER, VIEWER];

/** A role with its scope, as a person's roles show it. */
export function assignment(
  id: string,
  of: Role,
  scope: RoleAssignment['scope'],
  scopeEmpty = false,
): RoleAssignment {
  return {
    id,
    role: { id: of.id, name: of.name, builtIn: of.builtIn },
    scope,
    scopeEmpty,
  };
}

export const MY_TEAMS_EUROPE = { type: 'myTeams' as const, teams: [EUROPE], text: 'My teams' };
export const ALL_MARKETS = { type: 'all' as const, teams: [], text: 'All markets' };

export const ALEX_ROLES: PersonRoles = {
  person: ALEX,
  teams: [GLOBAL],
  roles: [assignment('RA-1001', ADMINISTRATOR, ALL_MARKETS)],
  permissions: ALL_KEYS.map((key) => ({ key, all: true, teams: [] })),
  version: 1,
};
export const DANIEL_ROLES: PersonRoles = {
  person: DANIEL,
  teams: [EUROPE],
  roles: [
    assignment('RA-1003', APPROVER, MY_TEAMS_EUROPE),
    assignment('RA-1002', PLANNER, MY_TEAMS_EUROPE),
  ],
  permissions: [],
  version: 3,
};
export const SARA_ROLES: PersonRoles = {
  person: SARA,
  teams: [EUROPE, CHINA],
  roles: [
    assignment('RA-1006', VIEWER, {
      type: 'myTeams',
      teams: [EUROPE, CHINA],
      text: 'My teams',
    }),
  ],
  permissions: [
    { key: 'plans.view', all: false, teams: [EUROPE, CHINA] },
    { key: 'finance.view', all: false, teams: [EUROPE, CHINA] },
  ],
  version: 2,
};
export const PRIYA_ROLES: PersonRoles = {
  person: PRIYA,
  teams: [EUROPE],
  roles: [],
  permissions: [],
  version: 1,
};

export const PEOPLE_ACCESS = {
  items: [ALEX_ROLES, DANIEL_ROLES, PRIYA_ROLES, SARA_ROLES],
  total: 4,
};

export const CATALOGUE: PermissionCatalogue = {
  groups: [
    {
      name: 'Plans and planning',
      permissions: PERMISSIONS.slice(0, 11).map(({ key, label, scoped }) => ({
        key,
        label,
        scoped,
      })),
    },
    {
      name: 'Financial data',
      permissions: PERMISSIONS.slice(11, 14).map(({ key, label, scoped }) => ({
        key,
        label,
        scoped,
      })),
    },
    {
      name: 'Approvals',
      permissions: PERMISSIONS.slice(14, 16).map(({ key, label, scoped }) => ({
        key,
        label,
        scoped,
      })),
    },
    {
      name: 'Global budgets',
      permissions: PERMISSIONS.slice(16, 18).map(({ key, label, scoped }) => ({
        key,
        label,
        scoped,
      })),
    },
    {
      name: 'Administration',
      permissions: PERMISSIONS.slice(18).map(({ key, label, scoped }) => ({ key, label, scoped })),
    },
  ],
};

export const RULES: AccessRules = {
  noSelfApproval: true,
  newPersonRole: { id: VIEWER.id, name: VIEWER.name, builtIn: VIEWER.builtIn },
  version: 0,
  updatedAt: null,
  updatedBy: null,
};

export const NO_CHANGES = { items: [], total: 0 };

/** Every list a page of Roles and access asks for, answered with the demo data. */
export const ACCESS_ANSWERS = [
  reply('GET /api/v1/roles', { body: { items: ROLES, total: ROLES.length } }),
  reply('GET /api/v1/access/people', { body: PEOPLE_ACCESS }),
  reply('GET /api/v1/access/permissions', { body: CATALOGUE }),
  reply('GET /api/v1/access-rules', { body: RULES }),
  reply('GET /api/v1/access/changes', { body: NO_CHANGES }),
  reply('GET /api/v1/teams', {
    body: {
      items: TEAMS.map((team) => ({
        ...team,
        description: '',
        members: [],
        plans: [],
        ...META,
      })),
      total: TEAMS.length,
    },
  }),
];
