/**
 * A person's roles while they are being changed in the roles panel (014 plan, "People tab"): which
 * roles are ticked and the scope of each, the body PUT and the impact check send, and the preview
 * of what the roles would allow together (merged as the server's authz.Grants does).
 */
import type { TeamRef } from '@/ui/TeamPill/TeamPill';
import type {
  AccessScopeType,
  MyPermission,
  PersonRoles,
  PersonRolesUpdate,
  PermissionKey,
  Role,
} from './api';
import { isScoped, permissionOrder } from './access-words';

/** One ticked role and where it applies. */
export interface RoleDraft {
  roleId: string;
  scope: AccessScopeType;
  /** The chosen teams when the scope is "teams"; kept while another scope is chosen. */
  teamIds: string[];
}

/** The person's roles as stored, as drafts. */
export function draftsOf(person: PersonRoles): RoleDraft[] {
  return person.roles.map((assignment) => ({
    roleId: assignment.role.id,
    scope: assignment.scope.type,
    teamIds: assignment.scope.type === 'teams' ? assignment.scope.teams.map((team) => team.id) : [],
  }));
}

/** The body of PUT …/roles and of the impact check: the complete list, in the panel's order. */
export function rolesBody(drafts: RoleDraft[], version: number): PersonRolesUpdate {
  return {
    roles: drafts.map((draft) => ({
      roleId: draft.roleId,
      scope:
        draft.scope === 'teams' ? { type: 'teams', teamIds: draft.teamIds } : { type: draft.scope },
    })),
    version,
  };
}

/** True when the drafts say something else than the stored roles (order does not matter). */
export function draftsDiffer(a: RoleDraft[], b: RoleDraft[]): boolean {
  const key = (draft: RoleDraft) =>
    `${draft.roleId}:${draft.scope}:${draft.scope === 'teams' ? [...draft.teamIds].sort().join(',') : ''}`;
  const left = a.map(key).sort().join('|');
  const right = b.map(key).sort().join('|');
  return left !== right;
}

/**
 * What the ticked roles would allow together, for the live summary: every permission of every
 * ticked role, granted for all markets when a role's scope is All markets or the permission is not
 * scoped, otherwise for the person's teams (My teams) or the chosen teams. In Table A order, teams
 * sorted by name — as the API answers `PersonRoles.permissions` after saving.
 */
export function previewPermissions(
  drafts: RoleDraft[],
  roles: Role[],
  personTeams: TeamRef[],
  allTeams: TeamRef[],
): MyPermission[] {
  const merged = new Map<PermissionKey, { all: boolean; teams: Map<string, TeamRef> }>();
  for (const draft of drafts) {
    const role = roles.find((candidate) => candidate.id === draft.roleId);
    if (!role) {
      continue;
    }
    const where =
      draft.scope === 'myTeams'
        ? personTeams
        : draft.scope === 'teams'
          ? allTeams.filter((team) => draft.teamIds.includes(team.id))
          : [];
    for (const key of role.permissions) {
      const entry = merged.get(key) ?? { all: false, teams: new Map<string, TeamRef>() };
      if (draft.scope === 'all' || !isScoped(key)) {
        entry.all = true;
      } else {
        for (const team of where) {
          entry.teams.set(team.id, team);
        }
      }
      merged.set(key, entry);
    }
  }
  return [...merged]
    .map(([key, entry]) => ({
      key,
      all: entry.all,
      teams: entry.all
        ? []
        : [...entry.teams.values()].sort((a, b) => a.name.localeCompare(b.name)),
    }))
    .filter((permission) => permission.all || permission.teams.length > 0)
    .sort((a, b) => permissionOrder(a.key) - permissionOrder(b.key));
}

/** The permissions the ticked roles hold, whatever their scope. */
export function heldPermissions(drafts: RoleDraft[], roles: Role[]): Set<PermissionKey> {
  const held = new Set<PermissionKey>();
  for (const draft of drafts) {
    for (const key of roles.find((role) => role.id === draft.roleId)?.permissions ?? []) {
      held.add(key);
    }
  }
  return held;
}

/** True when a ticked role is the Administrator role. */
export function holdsAdministrator(drafts: RoleDraft[], roles: Role[]): boolean {
  return drafts.some(
    (draft) => roles.find((role) => role.id === draft.roleId)?.builtIn === 'administrator',
  );
}

/**
 * The change needs the impact check first (014 plan, FR-022): it takes away approving
 * (approvals.decide), or it takes the Administrator role from the person who is saving.
 */
export function needsImpactCheck(
  before: RoleDraft[],
  after: RoleDraft[],
  roles: Role[],
  isYou: boolean,
): boolean {
  const losesApproving =
    heldPermissions(before, roles).has('approvals.decide') &&
    !heldPermissions(after, roles).has('approvals.decide');
  const losesOwnAdministration =
    isYou && holdsAdministrator(before, roles) && !holdsAdministrator(after, roles);
  return losesApproving || losesOwnAdministration;
}
