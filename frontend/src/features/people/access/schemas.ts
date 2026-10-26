import { z } from 'zod';
import type { PermissionKey, RoleCreate, RoleUpdate } from './api';
import { permissionOrder } from './access-words';

const TABS = ['people', 'roles', 'rules'] as const;

export type AccessTab = (typeof TABS)[number];

const tabSchema = z.enum(TABS);
const textParamSchema = z.string();

/** The tab of `?tab=`: People when it names none. */
export function accessTabOf(params: URLSearchParams): AccessTab {
  const parsed = tabSchema.safeParse(params.get('tab'));
  return parsed.success ? parsed.data : 'people';
}

/** The address of another tab: People is the address without `?tab=`; no person stays open. */
export function withAccessTab(params: URLSearchParams, tab: AccessTab): URLSearchParams {
  const changed = new URLSearchParams(params);
  changed.delete('person');
  if (tab === 'people') {
    changed.delete('tab');
  } else {
    changed.set('tab', tab);
  }
  return changed;
}

export function openPersonOf(params: URLSearchParams): string {
  return textParam(params, 'person');
}

export function withOpenPerson(params: URLSearchParams, personId: string): URLSearchParams {
  const next = new URLSearchParams(params);
  if (personId) {
    next.set('person', personId);
  } else {
    next.delete('person');
  }
  return next;
}

export function copyOfParam(params: URLSearchParams): string {
  return textParam(params, 'copyOf');
}

function textParam(params: URLSearchParams, name: string): string {
  const parsed = textParamSchema.safeParse(params.get(name));
  return parsed.success ? parsed.data : '';
}

// The API checks these forms; the permission keys come from its catalogue.
const permissionsSchema = z.array(z.custom<PermissionKey>((key) => typeof key === 'string'));

export const roleSchema = z.object({
  name: z.string(),
  description: z.string(),
  defaultScope: z.enum(['all', 'myTeams']),
  permissions: permissionsSchema,
});

export type RoleValues = z.infer<typeof roleSchema>;

export const EMPTY_ROLE: RoleValues = {
  name: '',
  description: '',
  defaultScope: 'myTeams',
  permissions: [],
};

/** Table A's order: the form keeps its ticks so, as the API saves them. */
export function inTableOrder(permissions: PermissionKey[]): PermissionKey[] {
  return [...permissions].sort((a, b) => permissionOrder(a) - permissionOrder(b));
}

export function newRole(values: RoleValues): RoleCreate {
  return { ...values, permissions: inTableOrder(values.permissions) };
}

/** Only what changed, and the version (D-203, D-210). */
export function roleChanges(start: RoleValues, values: RoleValues, version: number): RoleUpdate {
  const permissions = inTableOrder(values.permissions);
  return {
    ...(values.name !== start.name ? { name: values.name } : {}),
    ...(values.description !== start.description ? { description: values.description } : {}),
    ...(values.defaultScope !== start.defaultScope ? { defaultScope: values.defaultScope } : {}),
    ...(permissions.join() !== inTableOrder(start.permissions).join() ? { permissions } : {}),
    version,
  };
}

export const personRolesSchema = z.object({
  roles: z.array(
    z.object({
      roleId: z.string(),
      scope: z.enum(['all', 'myTeams', 'teams']),
      teamIds: z.array(z.string()),
    }),
  ),
});

export type PersonRolesValues = z.infer<typeof personRolesSchema>;

export const rulesSchema = z.object({ noSelfApproval: z.boolean(), newPersonRoleId: z.string() });

export type RulesValues = z.infer<typeof rulesSchema>;
