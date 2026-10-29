import type { Access } from './access';

// The API's words (014 plan messages 1, 2) for an action shown disabled before the API is asked.
export function refusalText(noun: string, access: Access): string {
  const names = access.roles.map((assignment) => assignment.role.name);
  if (names.length === 0) {
    return `${noun} needs a role, and you don't have one yet. Ask an administrator.`;
  }
  const roles = names.length === 1 ? `your role (${names[0]})` : `your roles (${names.join(', ')})`;
  return `${noun} isn't part of ${roles}. Ask an administrator if you need it.`;
}
