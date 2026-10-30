// What the signed-in person may do (D-229): the app shows the server's decisions and never works a
// permission out itself; an answer without `access` counts as allowed (D-298).
import { ApiError } from '@/api/errors';
import type { components } from '@/api/generated/schema';
import { useSession } from './useSession';

export type Permission = components['schemas']['PermissionKey'];
export type MyAccess = components['schemas']['MyAccess'];
export type MyPermission = components['schemas']['MyPermission'];
export type Availability = components['schemas']['ActionAvailability'];
type TeamRef = components['schemas']['TeamRef'];
type RoleAssignment = components['schemas']['RoleAssignment'];
type PersonRef = components['schemas']['PersonRef'];

export interface PermissionScope {
  all: boolean;
  teams: TeamRef[];
}

export interface Access {
  has: (permission: Permission) => boolean;
  scopeOf: (permission: Permission) => PermissionScope | null;
  roles: RoleAssignment[];
  administrators: PersonRef[];
  adminNames: () => string;
  rulesNoSelfApproval: boolean;
  /** false while the API sends no access (before feature 014): everything counts as allowed. */
  known: boolean;
}

export function useAccess(): Access {
  const { user } = useSession();
  return accessOf(user?.access);
}

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

export interface AvailabilityProps {
  disabled: boolean;
  hint: string | undefined;
}

export function availabilityProps(availability: Availability | undefined): AvailabilityProps {
  if (!availability || availability.available) {
    return { disabled: false, hint: undefined };
  }
  return { disabled: true, hint: availability.reason || undefined };
}

/** An unavailable action without a reason is hidden; one with a reason shows, disabled. */
export function isShown(availability: Availability | undefined): boolean {
  return !availability || availability.available || availability.reason !== '';
}

/** The server's sentence for a 403 refusal, shown with NoAccess; null for any other error. */
export function forbiddenMessage(error: unknown): string | null {
  return error instanceof ApiError && error.kind === 'forbidden' ? error.message : null;
}

export function joinWith(items: string[], word: 'and' | 'or'): string {
  if (items.length <= 1) {
    return items[0] ?? '';
  }
  return `${items.slice(0, -1).join(', ')} ${word} ${items[items.length - 1]}`;
}

export function itemsFor<T extends { permission?: Permission }>(
  items: T[],
  has: (permission: Permission) => boolean,
): T[] {
  return items.filter((item) => !item.permission || has(item.permission));
}
