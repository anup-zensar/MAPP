/**
 * The words of feature 014 that are put together from data (plan "User-visible messages", Table A):
 * permission labels, a role with its scope, the plain-language summary of what roles allow, and
 * the administrators to ask.
 */
import type {
  AccessScopeType,
  ActivityLogEntry,
  MyPermission,
  PermissionKey,
  RoleAssignment,
} from './api';

/** One permission of Table A: its label on the role page and whether a scope limits it. */
export interface PermissionWords {
  key: PermissionKey;
  label: string;
  /** True when the permission applies only to plans in the role's scope (FR-004). */
  scoped: boolean;
}

/** Table A, in its order (fixed by MAPP; the API's catalogue says the same). */
export const PERMISSIONS: PermissionWords[] = [
  { key: 'plans.view', label: 'View plans and everything in them', scoped: true },
  { key: 'plans.create', label: 'Create and copy plans', scoped: true },
  { key: 'plans.edit', label: 'Edit plan details', scoped: true },
  { key: 'plans.publish', label: 'Publish and unpublish plans', scoped: true },
  { key: 'plans.delete', label: 'Delete plans', scoped: true },
  {
    key: 'campaigns.edit',
    label: 'Add, edit, move and copy campaigns and initiatives',
    scoped: true,
  },
  { key: 'campaigns.delete', label: 'Delete campaigns and initiatives', scoped: true },
  {
    key: 'activities.edit',
    label: 'Add, edit, move and copy activities and line items',
    scoped: true,
  },
  { key: 'activities.delete', label: 'Delete activities and line items', scoped: true },
  {
    key: 'activities.submit',
    label: 'Submit activities for approval and recall them',
    scoped: true,
  },
  { key: 'data.export', label: 'Export from All work', scoped: true },
  { key: 'finance.view', label: 'See financial data', scoped: true },
  { key: 'finance.link', label: 'Link financial transactions to line items', scoped: true },
  { key: 'finance.import', label: 'Import GSAP extracts', scoped: false },
  { key: 'approvals.decide', label: 'Approve or decline when named as an approver', scoped: false },
  {
    key: 'approvals.decideOnBehalf',
    label: 'Approve or decline on behalf of the named approvers',
    scoped: true,
  },
  { key: 'globalBudgets.view', label: 'See global budgets and their allocations', scoped: true },
  {
    key: 'globalBudgets.manage',
    label: 'Create global budgets and allocate them to plans',
    scoped: true,
  },
  {
    key: 'settings.manage',
    label:
      'Change Settings lists (models, activity types, Shell attributes, finance budget items, field help)',
    scoped: false,
  },
  {
    key: 'exchangeRates.manage',
    label: 'Add currencies, change exchange rates and import rate files',
    scoped: false,
  },
  { key: 'workflows.manage', label: 'Create and change approval workflows', scoped: false },
  { key: 'people.manage', label: 'Add and change people and teams', scoped: false },
  { key: 'access.manage', label: 'Give roles and change roles and access rules', scoped: false },
  {
    key: 'activityLog.viewAll',
    label: 'See the whole activity log, including Settings, people and access changes',
    scoped: false,
  },
  { key: 'recycleBin.purge', label: 'Delete permanently and empty the recycle bin', scoped: false },
];

const BY_KEY = new Map(
  PERMISSIONS.map((permission, index) => [permission.key, { permission, index }]),
);

/** "Edit plan details"; the key itself for a permission this web app doesn't know yet. */
export function permissionLabel(key: PermissionKey): string {
  return BY_KEY.get(key)?.permission.label ?? key;
}

/** The scope choices of a role, as the People tab writes them. */
export const SCOPE_WORDS: Record<AccessScopeType, string> = {
  all: 'All markets',
  myTeams: 'My teams',
  teams: 'Chosen teams',
};

/** "Planner · My teams", "Planner · Mobility China", "Global planner · All markets". */
export function assignmentText(assignment: RoleAssignment): string {
  return `${assignment.role.name} · ${assignment.scope.text}`;
}

/** "Viewer · My teams, Planner · Mobility China"; "No role" when there is none. */
export function rolesText(assignments: RoleAssignment[]): string {
  return assignments.length > 0 ? assignments.map(assignmentText).join(', ') : 'No role';
}

/** "0 people", "1 person", "4 people": how many hold a role. */
export function peopleText(count: number): string {
  return count === 1 ? '1 person' : `${count} people`;
}

/** True when a scope limits the permission (Table A "scoped"); unknown keys count as scoped. */
export function isScoped(key: PermissionKey): boolean {
  return BY_KEY.get(key)?.permission.scoped ?? true;
}

/** Table A's order of a permission (unknown keys last). */
export function permissionOrder(key: PermissionKey): number {
  return BY_KEY.get(key)?.index ?? PERMISSIONS.length;
}

/** "A", "A and B", "A, B and C". */
export function andList(items: string[]): string {
  return joinList(items, 'and');
}

/** "Alex Morgan", "Alex Morgan or Nina Berg" — the administrators to ask. */
export function orList(items: string[]): string {
  return joinList(items, 'or');
}

function joinList(items: string[], word: 'and' | 'or'): string {
  if (items.length <= 1) {
    return items[0] ?? '';
  }
  return `${items.slice(0, -1).join(', ')} ${word} ${items[items.length - 1]}`;
}

/** "Export from All work" → "export from All work", for the middle of a sentence. */
function midSentence(label: string): string {
  return label.charAt(0).toLowerCase() + label.slice(1);
}

/**
 * What a set of permissions allows, in plain language (FR-017, FR-024; plan `summariseAccess`):
 * the scoped permissions grouped by where they apply, one line per group in Table A order — "In
 * Lubricants Europe and Mobility China: view plans and everything in them and export from All
 * work." — then "Everywhere: …" for the permissions no scope limits. No permission at all: "No
 * access yet."
 */
export function summariseAccess(permissions: MyPermission[]): string[] {
  if (permissions.length === 0) {
    return ['No access yet.'];
  }
  const ordered = [...permissions].sort(
    (a, b) => (BY_KEY.get(a.key)?.index ?? 99) - (BY_KEY.get(b.key)?.index ?? 99),
  );
  const groups = new Map<string, string[]>();
  const everywhere: string[] = [];
  for (const permission of ordered) {
    const label = midSentence(permissionLabel(permission.key));
    const scoped = BY_KEY.get(permission.key)?.permission.scoped ?? true;
    if (!scoped) {
      everywhere.push(label);
      continue;
    }
    const where = permission.all
      ? 'all markets'
      : andList(permission.teams.map((team) => team.name));
    if (!where) {
      continue;
    }
    groups.set(where, [...(groups.get(where) ?? []), label]);
  }
  const lines = [...groups].map(([where, labels]) => `In ${where}: ${andList(labels)}.`);
  if (everywhere.length > 0) {
    lines.push(`Everywhere: ${andList(everywhere)}.`);
  }
  return lines.length > 0 ? lines : ['No access yet.'];
}

/** Text 11: the note on the Administrator role. */
export const ADMINISTRATOR_LOCK =
  "The Administrator role always has every permission and can't be changed.";

/** Text 12: the Administrator role is never limited to some markets. */
export const ADMINISTRATOR_SCOPE = 'The Administrator role always applies to all markets.';

/** Text 37: a scope that covers no team any more. */
export const SCOPE_EMPTY = 'No teams left in this scope';

/** The rule of FR-026, as the card and the form say it. */
export const NO_SELF_APPROVAL = 'Nobody approves an activity they own or submitted';

/** "Alex Morgan changed person 'Sara Novak'", as the activity log writes it (FR-038). */
export function changeText(entry: ActivityLogEntry): string {
  return `${entry.actor.name} ${entry.verb} ${entry.recordTypeLabel} '${entry.recordName}'`;
}
