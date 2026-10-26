import type { ReactNode } from 'react';
import { NoAccess } from '@/ui/NoAccess/NoAccess';
import { useAccess } from './access';
import type { Permission } from './access';

export interface RequirePermissionProps {
  /** What a role of the person must include, in any scope, to open the page. */
  permission: Permission;
  children: ReactNode;
}

/**
 * The gate of a page that only some roles open (feature 014, FR-012): Settings › Roles and access
 * needs "Give roles and change roles and access rules", Global budgets "See global budgets". Anyone
 * else who opens its address sees the no-access page. The server refuses the page's calls anyway;
 * this only spares people a page of errors. Use it inside RequireSignIn:
 *
 *   { path: '/settings/access', element: <RequirePermission permission="access.manage"><AccessPage /></RequirePermission> }
 */
export function RequirePermission({ permission, children }: RequirePermissionProps) {
  const { has } = useAccess();
  return has(permission) ? children : <NoAccess />;
}
