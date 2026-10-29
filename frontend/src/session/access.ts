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
import type { components } from '@/api/generated/schema';
import { useSession } from './useSession';

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

/** "A", "A and B", "A, B and C" (or "or"); "" for nothing. */
export function joinWith(items: string[], word: 'and' | 'or'): string {
  if (items.length <= 1) {
    return items[0] ?? '';
  }
  return `${items.slice(0, -1).join(', ')} ${word} ${items[items.length - 1]}`;
}

/**
 * The items the person's roles open (feature 014, FR-012): items without a permission, and those
 * whose permission `has` grants.
 */
export function itemsFor<T extends { permission?: Permission }>(
  items: T[],
  has: (permission: Permission) => boolean,
): T[] {
  return items.filter((item) => !item.permission || has(item.permission));
}
