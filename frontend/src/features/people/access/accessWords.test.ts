import { describe, expect, it } from 'vitest';
import { andList, orList, peopleText, rolesText, summariseAccess } from './accessWords';
import { draftsDiffer, needsImpactCheck, previewPermissions, rolesBody } from './roleDrafts';
import {
  ADMINISTRATOR,
  APPROVER,
  assignment,
  CHINA,
  EUROPE,
  MY_TEAMS_EUROPE,
  PLANNER,
  ROLES,
  TEAMS,
  VIEWER,
} from '@/testing/fixtures/access';

describe('feature 014 words', () => {
  it('joins names with "and" and "or"', () => {
    expect(andList([])).toBe('');
    expect(andList(['A'])).toBe('A');
    expect(andList(['A', 'B', 'C'])).toBe('A, B and C');
    expect(orList(['Alex Morgan', 'Nina Berg'])).toBe('Alex Morgan or Nina Berg');
  });

  it('writes roles with their scope, and how many hold a role', () => {
    expect(rolesText([])).toBe('No role');
    expect(rolesText([assignment('RA-1', VIEWER, MY_TEAMS_EUROPE)])).toBe('Viewer · My teams');
    expect(peopleText(0)).toBe('0 people');
    expect(peopleText(1)).toBe('1 person');
  });

  it('summarises what roles allow by where it applies, then everywhere (FR-017, FR-024)', () => {
    expect(summariseAccess([])).toEqual(['No access yet.']);
    expect(
      summariseAccess([
        { key: 'data.export', all: false, teams: [EUROPE, CHINA] },
        { key: 'plans.view', all: false, teams: [EUROPE, CHINA] },
        { key: 'plans.create', all: false, teams: [CHINA] },
        { key: 'approvals.decide', all: true, teams: [] },
        { key: 'globalBudgets.view', all: true, teams: [] },
      ]),
    ).toEqual([
      'In Lubricants Europe and Mobility China: view plans and everything in them and export from All work.',
      'In Mobility China: create and copy plans.',
      'In all markets: see global budgets and their allocations.',
      'Everywhere: approve or decline when named as an approver.',
    ]);
  });
});

describe('a person’s roles while they change (014 plan, People tab)', () => {
  it('previews the merged permissions of the ticked roles and their scopes', () => {
    const preview = previewPermissions(
      [
        { roleId: VIEWER.id, scope: 'myTeams', teamIds: [] },
        { roleId: PLANNER.id, scope: 'teams', teamIds: [CHINA.id] },
      ],
      ROLES,
      [EUROPE],
      TEAMS,
    );

    expect(preview.find((permission) => permission.key === 'plans.view')).toEqual({
      key: 'plans.view',
      all: false,
      teams: [EUROPE, CHINA],
    });
    expect(preview.find((permission) => permission.key === 'plans.create')?.teams).toEqual([CHINA]);
    expect(
      previewPermissions(
        [{ roleId: ADMINISTRATOR.id, scope: 'all', teamIds: [] }],
        ROLES,
        [],
        TEAMS,
      ).every((permission) => permission.all),
    ).toBe(true);
  });

  it('sends the complete list, with teams only for chosen teams', () => {
    expect(
      rolesBody(
        [
          { roleId: PLANNER.id, scope: 'teams', teamIds: [CHINA.id] },
          { roleId: VIEWER.id, scope: 'myTeams', teamIds: [CHINA.id] },
        ],
        4,
      ),
    ).toEqual({
      roles: [
        { roleId: PLANNER.id, scope: { type: 'teams', teamIds: [CHINA.id] } },
        { roleId: VIEWER.id, scope: { type: 'myTeams' } },
      ],
      version: 4,
    });
  });

  it('checks the impact only when approving or your own administration is taken away (FR-022)', () => {
    const daniel = [
      { roleId: APPROVER.id, scope: 'myTeams' as const, teamIds: [] },
      { roleId: PLANNER.id, scope: 'myTeams' as const, teamIds: [] },
    ];
    expect(needsImpactCheck(daniel, daniel.slice(1), ROLES, false)).toBe(true);
    expect(needsImpactCheck(daniel.slice(1), daniel, ROLES, false)).toBe(false);
    const admin = [{ roleId: ADMINISTRATOR.id, scope: 'all' as const, teamIds: [] }];
    expect(needsImpactCheck(admin, [], ROLES, true)).toBe(true);
    expect(draftsDiffer(daniel, [...daniel].reverse())).toBe(false);
    expect(draftsDiffer(daniel, daniel.slice(1))).toBe(true);
  });
});
