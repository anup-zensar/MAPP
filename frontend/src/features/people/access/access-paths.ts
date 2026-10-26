/** The addresses of Roles and access (014 plan, "Routes"). */
export const ACCESS_PATH = '/settings/access';
export const NEW_ROLE_PATH = `${ACCESS_PATH}/roles/new`;

export function rolesTabPath(): string {
  return `${ACCESS_PATH}?tab=roles`;
}

export function rolePath(roleId: string): string {
  return `${ACCESS_PATH}/roles/${encodeURIComponent(roleId)}`;
}

export function roleEditPath(roleId: string): string {
  return `${rolePath(roleId)}/edit`;
}

/** New role, filled in from another role ("Copy of …"). */
export function roleCopyPath(roleId: string): string {
  return `${NEW_ROLE_PATH}?copyOf=${encodeURIComponent(roleId)}`;
}

/** The person's roles panel on the People tab. */
export function personRolesPath(personId: string): string {
  return `${ACCESS_PATH}?tab=people&person=${encodeURIComponent(personId)}`;
}
