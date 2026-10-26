/**
 * What the signed-in person may do (feature 014, D-229): the web app shows the server's decisions and
 * never works a permission out itself. Navigation, page-level gates and the View only note read the
 * person's access from GET /api/v1/me (`useAccess`); per-record actions read the records' own `access`
 * objects (`availabilityProps`).
 *
 * An answer without `access` (an API from before feature 014) counts as "allowed", as before
 * (consolidation D-298).
 */
import { ApiError } from '@/api/errors';
import type { components } from '@/api/schema';
import { useSession } from '@/app/useSession';

/** One of the fixed permissions of 014 Table A, such as "plans.edit". */
export type Permission = components['schemas']['PermissionKey'];
/** The signed-in person's access (GET /api/v1/me). */
export type MyAccess = components['schemas']['MyAccess'];
/** One permission a person holds, merged over their roles: for All markets or for some teams. */
export type MyPermission = components['schemas']['MyPermission'];
/** Whether one action is possible now, and why not (the records' `access` objects). */
export type Availability = components['schemas']['ActionAvailability'];
type TeamRef = components['schemas']['TeamRef'];
type RoleAssignment = components['schemas']['RoleAssignment'];
type PersonRef = components['schemas']['PersonRef'];

/** Where a person holds a permission: everywhere (All markets) or in the teams listed, by name. */
export interface PermissionScope {
  all: boolean;
  teams: TeamRef[];
}

/** What useAccess() gives a page. */
export interface Access {
  /** Whether a role of the person includes `permission`, in any scope. */
  has: (permission: Permission) => boolean;
  /** Where the person holds `permission`; null when they do not hold it. */
  scopeOf: (permission: Permission) => PermissionScope | null;
  /** The person's roles with their scopes, by role name. */
  roles: RoleAssignment[];
  /** The active administrators, by name: whom to ask. */
  administrators: PersonRef[];
  /** "Alex Morgan", "Alex Morgan or Nina Berg", "Alex Morgan, Nina Berg or Sam Lee"; "" for none. */
  adminNames: () => string;
  /** Nobody approves an activity they own or submitted (014 FR-026). */
  rulesNoSelfApproval: boolean;
  /** false while the API sends no access (before feature 014): everything counts as allowed. */
  known: boolean;
}

/** The signed-in person's access, from GET /api/v1/me (FR-024); everything allowed while it is unknown. */
export function useAccess(): Access {
  const { user } = useSession();
  return accessOf(user?.access);
}

/** The Access of `access` as GET /api/v1/me answers it; everything allowed when it is missing. */
export function accessOf(access: MyAccess | undefined): Access {
  const administrators = access?.administrators ?? [];
  const byKey = new Map((access?.permissions ?? []).map((p) => [p.key, p]));
  return {
    has: (permission) => !access || byKey.has(permission),
    scopeOf: (permission) => {
      if (!access) {
        return { all: true, teams: [] };
      }
      const held = byKey.get(permission);
      return held ? { all: held.all, teams: held.all ? [] : held.teams } : null;
    },
    roles: access?.roles ?? [],
    administrators,
    adminNames: () =>
      joinWith(
        administrators.map((a) => a.name),
        'or',
      ),
    rulesNoSelfApproval: access?.noSelfApproval ?? true,
    known: access !== undefined,
  };
}

/** What a disabled action shows: `disabled` and the reason as a hint read with it (design system §6.6). */
export interface AvailabilityProps {
  disabled: boolean;
  /** The server's reason, or undefined when the action is available or the screen hides it. */
  hint: string | undefined;
}

/**
 * Turns a record's `{available, reason}` into the props of an ActionsMenu item or a Button (FR-011):
 * an unavailable action is disabled and says why. A missing availability (an API from before 014)
 * counts as available.
 */
export function availabilityProps(availability: Availability | undefined): AvailabilityProps {
  if (!availability || availability.available) {
    return { disabled: false, hint: undefined };
  }
  return { disabled: true, hint: availability.reason || undefined };
}

/**
 * Whether the screen shows an action at all (FR-011, 008's rule): an available action, or an
 * unavailable one with a reason; an unavailable action without a reason is hidden.
 */
export function isShown(availability: Availability | undefined): boolean {
  return !availability || availability.available || availability.reason !== '';
}

/**
 * The server's sentence when `error` is a refusal by the person's roles (403 forbidden): what a page
 * that failed to load shows with NoAccess (FR-008). null for any other error.
 */
export function forbiddenMessage(error: unknown): string | null {
  return error instanceof ApiError && error.kind === 'forbidden' ? error.message : null;
}

/** The permissions of Table A, in its order, with their labels and whether they are scoped. */
const TABLE_A: { key: Permission; label: string; scoped: boolean }[] = [
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
  {
    key: 'approvals.decide',
    label: 'Approve or decline when named as an approver',
    scoped: false,
  },
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
  {
    key: 'access.manage',
    label: 'Give roles and change roles and access rules',
    scoped: false,
  },
  {
    key: 'activityLog.viewAll',
    label: 'See the whole activity log, including Settings, people and access changes',
    scoped: false,
  },
  { key: 'recycleBin.purge', label: 'Delete permanently and empty the recycle bin', scoped: false },
];

/** The label of a permission on the role page (Table A): "Edit plan details". */
export function permissionLabel(permission: Permission): string {
  return TABLE_A.find((p) => p.key === permission)?.label ?? permission;
}

/**
 * The plain-language summary of what permissions allow (FR-017, FR-024), one sentence per line: for
 * the scoped permissions, one line per place they apply — "In Lubricants Europe and Mobility China:
 * view plans and everything in them, export from All work and see financial data." — in Table A
 * order; then "Everywhere: …" for the permissions that are not scoped. "No access yet." for none.
 * Display text only: the rules are the server's.
 */
export function summariseAccess(permissions: MyPermission[]): string[] {
  const held = TABLE_A.flatMap((info) => {
    const p = permissions.find((candidate) => candidate.key === info.key);
    return p ? [{ ...info, held: p }] : [];
  });
  if (held.length === 0) {
    return ['No access yet.'];
  }
  const groups = new Map<string, string[]>(); // where → labels, in Table A order of their first label
  const everywhere: string[] = [];
  for (const p of held) {
    const label = lowerFirst(p.label);
    if (!p.scoped) {
      everywhere.push(label);
      continue;
    }
    const where = p.held.all
      ? 'all markets'
      : joinWith(
          p.held.teams.map((t) => t.name),
          'and',
        ) || 'your teams, once you are in one';
    groups.set(where, [...(groups.get(where) ?? []), label]);
  }
  const lines = [...groups].map(([where, labels]) => `In ${where}: ${joinWith(labels, 'and')}.`);
  if (everywhere.length > 0) {
    lines.push(`Everywhere: ${joinWith(everywhere, 'and')}.`);
  }
  return lines;
}

/** "A", "A and B", "A, B and C" (or "or"); "" for nothing. */
export function joinWith(items: string[], word: 'and' | 'or'): string {
  if (items.length <= 1) {
    return items[0] ?? '';
  }
  return `${items.slice(0, -1).join(', ')} ${word} ${items[items.length - 1]}`;
}

/** The text with its first letter in lower case: labels in the middle of a sentence. */
function lowerFirst(text: string): string {
  return text.charAt(0).toLowerCase() + text.slice(1);
}
